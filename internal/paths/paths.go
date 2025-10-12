package paths

import (
	"os"
	"path/filepath"
)

type Dirs struct {
	Base, Data, Backups, Invoices, Exports string
}

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
