package backend

import (
	"context"

	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
)

func (s *Service) SettingsSetLocale(ctx context.Context, loc string) error {
	_, err := s.rt.DB.Ent.Settings.
		Update().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		SetLocale(loc).
		Save(ctx)
	return err
}

func (s *Service) SettingsGetLocale(ctx context.Context) (string, error) {
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "lv-LV", nil
	}
	st, err := s.rt.DB.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	return st.Locale, nil
}
