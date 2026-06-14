package runtime

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"langschool/internal/paths"
)

const (
	DefaultServerBaseDir = "/var/lib/langschool"
	AppDirName           = "StudentDesk"
	LegacyAppDirName     = "LangSchool"
)

type Config struct {
	BaseDir       string
	DataDir       string
	BackupsDir    string
	InvoicesDir   string
	ExportsDir    string
	FontsDir      string
	BaseURL       string
	AdminUsername string
	AdminPassword string
	SessionSecret string
}

func LoadConfig(_ string) Config {
	base := defaultServerBaseDir()
	cfg := Config{
		BaseDir:       base,
		DataDir:       envOrDefault("APP_DATA_DIR", filepath.Join(base, "data")),
		BackupsDir:    envOrDefault("BACKUPS_DIR", filepath.Join(base, "backups")),
		InvoicesDir:   envOrDefault("INVOICES_DIR", filepath.Join(base, "invoices")),
		ExportsDir:    filepath.Join(base, "exports"),
		FontsDir:      strings.TrimSpace(os.Getenv("LS_FONTS_DIR")),
		BaseURL:       strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		AdminUsername: firstNonEmpty(os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_EMAIL")),
		AdminPassword: strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		SessionSecret: strings.TrimSpace(os.Getenv("SESSION_SECRET")),
	}
	return cfg
}

func defaultServerBaseDir() string {
	return envOrDefault("APP_BASE_DIR", DefaultServerBaseDir)
}

func ResolveAppBaseDir(home string) string {
	base := filepath.Join(home, AppDirName)
	legacyBase := filepath.Join(home, LegacyAppDirName)

	if info, err := os.Stat(base); err == nil && info.IsDir() {
		return base
	}

	if info, err := os.Stat(legacyBase); err == nil && info.IsDir() {
		if err := os.Rename(legacyBase, base); err == nil {
			log.Printf("Migrated app data directory from %s to %s", legacyBase, base)
			return base
		}
		log.Printf("Using legacy app data directory %s because migration to %s failed", legacyBase, base)
		return legacyBase
	}

	return base
}

func UserHome() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func ResolveDirs(cfg Config) (paths.Dirs, error) {
	dirs := paths.Dirs{
		Base:     cfg.BaseDir,
		Data:     cfg.DataDir,
		Backups:  cfg.BackupsDir,
		Invoices: cfg.InvoicesDir,
		Exports:  cfg.ExportsDir,
	}

	return paths.EnsureLayout(dirs)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
