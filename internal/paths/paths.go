// Package paths provides utilities for managing application directory structure.
package paths

import (
	"os"
	"path/filepath"
)

// Dirs holds paths to all application directories.
// These directories are created under a base directory (typically ~/StudentDesk).
type Dirs struct {
	Base     string // Base directory for all application data
	Data     string // Directory for database files
	Backups  string // Directory for database backups
	Invoices string // Directory for generated invoice PDFs
	Exports  string // Directory for exported data
}

// Ensure creates all required application directories under the base path.
// If the directories already exist, they are left unchanged.
// Returns the Dirs structure with all paths set, or an error if directory
// creation fails. Directories are created with permissions 0o755 (rwxr-xr-x).
func Ensure(base string) (Dirs, error) {
	d := Dirs{
		Base:     base,
		Data:     filepath.Join(base, "Data"),
		Backups:  filepath.Join(base, "Backups"),
		Invoices: filepath.Join(base, "Invoices"),
		Exports:  filepath.Join(base, "Exports"),
	}

	if err := migrateLegacySubdirs(base, d); err != nil {
		return d, err
	}

	for _, p := range []string{d.Data, d.Backups, d.Invoices, d.Exports} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return d, err
		}
	}
	return d, nil
}

func migrateLegacySubdirs(base string, d Dirs) error {
	legacyToCanonical := map[string]string{
		filepath.Join(base, "data"):     d.Data,
		filepath.Join(base, "backups"):  d.Backups,
		filepath.Join(base, "invoices"): d.Invoices,
		filepath.Join(base, "exports"):  d.Exports,
	}

	for legacy, canonical := range legacyToCanonical {
		if err := mergeDirContents(legacy, canonical); err != nil {
			return err
		}
	}

	return nil
}

func mergeDirContents(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return os.Rename(src, dst)
	} else if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			if err := os.Rename(srcPath, dstPath); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}

		if entry.IsDir() {
			if err := mergeDirContents(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	remaining, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
