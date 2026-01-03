// Package paths provides utilities for managing application directory structure.
package paths

import (
	"os"
	"path/filepath"
)

// Dirs holds paths to all application directories.
// These directories are created under a base directory (typically ~/LangSchool).
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
		Data:     filepath.Join(base, "data"),
		Backups:  filepath.Join(base, "backups"),
		Invoices: filepath.Join(base, "invoices"),
		Exports:  filepath.Join(base, "exports"),
	}
	for _, p := range []string{d.Data, d.Backups, d.Invoices, d.Exports} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return d, err
		}
	}
	return d, nil
}
