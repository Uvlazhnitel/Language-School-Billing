package runtime

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"langschool/ent"
	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
	"langschool/internal/app/attendance"
	"langschool/internal/app/audit"
	invsvc "langschool/internal/app/invoice"
	paysvc "langschool/internal/app/payment"
	"langschool/internal/auth"
	"langschool/internal/infra"
	"langschool/internal/paths"
)

const (
	DefaultSchoolDisplayName = "ArtLab"
	DefaultSchoolAddress     = "Latgales iela 260, Rīga, Latvija"
	PreMigrationBackupLimit  = 30
)

type Runtime struct {
	Config    Config
	Dirs      paths.Dirs
	DB        *infra.DB
	AppDBPath string

	Attendance *attendance.Service
	Audit      *audit.Service
	Invoice    *invsvc.Service
	Payment    *paysvc.Service
	Auth       *auth.Service
}

func Start(ctx context.Context, cfg Config) (*Runtime, error) {
	dirs, err := ResolveDirs(cfg)
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dirs.Data, "app.sqlite")
	log.Println("Data path:", dbPath)

	if backupPath, err := BackupBeforeMigration(dbPath, dirs.Backups); err != nil {
		return nil, err
	} else if backupPath != "" {
		log.Println("Pre-migration backup created:", backupPath)
		if err := CleanupOldPreMigrationBackups(dirs.Backups, PreMigrationBackupLimit); err != nil {
			return nil, err
		}
	}

	db, err := infra.Open(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	if err := MigrateLegacyCourseTeachers(ctx, db.Ent); err != nil {
		_ = db.Ent.Close()
		return nil, err
	}

	if err := ensureSettings(ctx, db.Ent); err != nil {
		_ = db.Ent.Close()
		return nil, err
	}

	authService := auth.New(db.Ent, cfg.AdminUsername, cfg.AdminPassword, cfg.SessionSecret, cfg.BaseURL)
	if err := authService.BootstrapAdmin(ctx); err != nil {
		_ = db.Ent.Close()
		return nil, err
	}

	return &Runtime{
		Config:     cfg,
		Dirs:       dirs,
		DB:         db,
		AppDBPath:  dbPath,
		Attendance: attendance.New(db.Ent),
		Audit:      audit.New(db.Ent),
		Invoice:    invsvc.New(db.Ent),
		Payment:    paysvc.New(db.Ent),
		Auth:       authService,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.DB == nil || r.DB.Ent == nil {
		return nil
	}
	return r.DB.Ent.Close()
}

func ensureSettings(ctx context.Context, client *ent.Client) error {
	exists, err := client.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		_, err := client.Settings.
			Create().
			SetSingletonID(sharedapp.SettingsSingletonID).
			SetOrgName(DefaultSchoolDisplayName).
			SetAddress(DefaultSchoolAddress).
			SetInvoicePrefix("LS").
			SetNextSeq(1).
			SetInvoiceDayOfMonth(1).
			SetCurrency("EUR").
			SetLocale("en-US").
			Save(ctx)
		return err
	}

	st, err := client.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		return err
	}

	upd := client.Settings.
		Update().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		SetCurrency("EUR")

	orgName := strings.TrimSpace(st.OrgName)
	if orgName == "" || strings.EqualFold(orgName, "North Star Language Studio") {
		upd.SetOrgName(DefaultSchoolDisplayName)
	}

	address := strings.TrimSpace(st.Address)
	if address == "" || strings.EqualFold(address, "Brivibas iela 88, Riga, Latvia") {
		upd.SetAddress(DefaultSchoolAddress)
	}

	_, err = upd.Save(ctx)
	return err
}
