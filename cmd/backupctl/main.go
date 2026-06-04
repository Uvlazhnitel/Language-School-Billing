package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	appruntime "langschool/internal/runtime"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "create-full":
		if err := createFull(); err != nil {
			log.Fatal(err)
		}
	case "restore-full":
		if err := restoreFull(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func createFull() error {
	cfg := appruntime.LoadConfig(appruntime.UserHome())
	dirs, err := appruntime.ResolveDirs(cfg)
	if err != nil {
		return err
	}

	archivePath, err := appruntime.FullBackupNow(filepath.Join(dirs.Data, "app.sqlite"), dirs.Invoices, dirs.Backups)
	if err != nil {
		return err
	}
	if err := appruntime.CleanupOldFullBackups(dirs.Backups, appruntime.FullBackupLimit); err != nil {
		return err
	}

	fmt.Println(archivePath)
	return nil
}

func restoreFull(args []string) error {
	fs := flag.NewFlagSet("restore-full", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var archivePath string
	fs.StringVar(&archivePath, "archive", "", "path to full backup archive")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if archivePath == "" {
		return fmt.Errorf("--archive is required")
	}

	cfg := appruntime.LoadConfig(appruntime.UserHome())
	dirs, err := appruntime.ResolveDirs(cfg)
	if err != nil {
		return err
	}

	preRestorePath, err := appruntime.RestoreFullBackup(archivePath, filepath.Join(dirs.Data, "app.sqlite"), dirs.Invoices, dirs.Backups)
	if err != nil {
		return err
	}

	if preRestorePath != "" {
		fmt.Fprintf(os.Stderr, "Pre-restore backup created: %s\n", preRestorePath)
	}
	fmt.Println("Restore completed from", archivePath)
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  langschool-backupctl create-full")
	fmt.Fprintln(os.Stderr, "  langschool-backupctl restore-full --archive /path/to/full-YYYYMMDD-HHMMSS.tar.gz")
}
