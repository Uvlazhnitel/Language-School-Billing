package backend

import (
	"context"
	"errors"
	"net/http"
	"time"

	"langschool/internal/auth"
	appruntime "langschool/internal/runtime"
)

func (s *Service) Meta(ctx context.Context) (*Meta, error) {
	locale, err := s.SettingsGetLocale(ctx)
	if err != nil {
		return nil, err
	}
	return &Meta{
		Ready:        s.Ready(),
		Locale:       locale,
		Capabilities: CapabilitiesForRole(auth.RoleAdmin),
	}, nil
}

func (s *Service) SessionState(ctx context.Context, currentUser *auth.UserInfo) (*SessionDTO, error) {
	locale := "lv-LV"
	if currentUser != nil {
		locale = normalizeUILocale(currentUser.UILocale)
	}
	return &SessionDTO{
		Authenticated: currentUser != nil,
		User:          currentUser,
		Locale:        locale,
		Capabilities:  capabilitiesForCurrentUser(currentUser),
		Ready:         s.Ready(),
	}, nil
}

func (s *Service) Login(ctx context.Context, username, password string, rememberMe bool) (*auth.UserInfo, string, time.Time, bool, error) {
	if s.rt == nil || s.rt.Auth == nil {
		return nil, "", time.Time{}, false, auth.ErrUnauthorized
	}
	return s.rt.Auth.Login(ctx, username, password, rememberMe)
}

func (s *Service) Session(ctx context.Context, signedToken string) (*auth.UserInfo, error) {
	if s.rt == nil || s.rt.Auth == nil {
		return nil, auth.ErrUnauthorized
	}
	return s.rt.Auth.Session(ctx, signedToken)
}

func (s *Service) Logout(ctx context.Context, signedToken string) error {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.Logout(ctx, signedToken)
}

func (s *Service) SessionCookie(signedToken string, expiresAt time.Time, persistent bool) *http.Cookie {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.SessionCookie(signedToken, expiresAt, persistent)
}

func (s *Service) ClearSessionCookie() *http.Cookie {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.ClearSessionCookie()
}

func (s *Service) BackupNow() (string, error) {
	return appruntime.BackupNow(s.rt.AppDBPath, s.rt.Dirs.Backups)
}

func (s *Service) FullBackupNow() (string, error) {
	path, err := appruntime.FullBackupNow(s.rt.AppDBPath, s.rt.Dirs.Invoices, s.rt.Dirs.Backups)
	if err != nil {
		return "", err
	}
	if err := appruntime.CleanupOldFullBackups(s.rt.Dirs.Backups, appruntime.FullBackupLimit); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) UserList(ctx context.Context) ([]UserDTO, error) {
	return s.rt.Auth.ListUsers(ctx)
}

func (s *Service) UserCreate(ctx context.Context, username, password, role string) (*UserDTO, error) {
	return s.rt.Auth.CreateUser(ctx, username, password, role)
}

func (s *Service) UserUpdate(ctx context.Context, id int, username, role string, isActive bool) (*UserDTO, error) {
	return s.rt.Auth.UpdateUser(ctx, id, username, role, isActive)
}

func (s *Service) UserSetPassword(ctx context.Context, id int, password string) error {
	return s.rt.Auth.SetUserPassword(ctx, id, password)
}

func (s *Service) UserSetActive(ctx context.Context, id int, active bool) (*UserDTO, error) {
	return s.rt.Auth.SetUserActive(ctx, id, active)
}

func (s *Service) UserDelete(ctx context.Context, currentUserID, targetUserID int) error {
	return s.rt.Auth.DeleteUser(ctx, currentUserID, targetUserID)
}

func (s *Service) UserGetLocale(ctx context.Context, userID int) (string, error) {
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "lv-LV", nil
	}
	u, err := s.rt.DB.Ent.User.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return normalizeUILocale(u.UILocale), nil
}

func (s *Service) UserSetLocale(ctx context.Context, userID int, loc string) (string, error) {
	if err := validateUILocale(loc); err != nil {
		return "", err
	}
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "", errors.New("user store unavailable")
	}
	item, err := s.rt.DB.Ent.User.UpdateOneID(userID).SetUILocale(loc).Save(ctx)
	if err != nil {
		return "", err
	}
	return normalizeUILocale(item.UILocale), nil
}
