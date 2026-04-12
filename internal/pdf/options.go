package pdf

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Options — where fonts are located and where to save the PDF.
type Options struct {
	OutBaseDir string // root folder Invoices/
	FontsDir   string // folder with fonts (kept for compatibility even if not used)
	Currency   string // e.g. "EUR"
	Locale     string
}

// normalizePath ensures the path is absolute and clean.
func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	// Handle macOS-specific case where paths might be missing leading slash
	if strings.HasPrefix(path, "Users/") {
		path = "/" + path
	}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return filepath.Clean(path), nil
}
