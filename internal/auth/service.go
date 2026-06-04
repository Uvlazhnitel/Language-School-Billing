package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/user"
	"langschool/ent/websession"

	"golang.org/x/crypto/bcrypt"
)

const (
	CookieName        = "langschool_session"
	DefaultSessionTTL = 7 * 24 * time.Hour
	RoleAdmin         = "admin"
	RoleStaff         = "staff"
	DefaultAdminRole  = RoleAdmin
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrForbidden = errors.New("forbidden")

type UserInfo struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserRecord struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"isActive"`
}

type Service struct {
	client        *ent.Client
	adminEmail    string
	adminPassword string
	sessionSecret string
	baseURL       string
	now           func() time.Time
}

func New(client *ent.Client, adminEmail, adminPassword, sessionSecret, baseURL string) *Service {
	return &Service{
		client:        client,
		adminEmail:    strings.TrimSpace(strings.ToLower(adminEmail)),
		adminPassword: adminPassword,
		sessionSecret: strings.TrimSpace(sessionSecret),
		baseURL:       strings.TrimSpace(baseURL),
		now:           time.Now,
	}
}

func (s *Service) BootstrapAdmin(ctx context.Context) error {
	if s == nil || s.client == nil || s.adminEmail == "" || s.adminPassword == "" {
		return nil
	}

	passwordHash, err := hashPassword(s.adminPassword)
	if err != nil {
		return err
	}

	existing, err := s.client.User.Query().Where(user.EmailEQ(s.adminEmail)).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = s.client.User.Create().
			SetEmail(s.adminEmail).
			SetPasswordHash(passwordHash).
			SetRole(DefaultAdminRole).
			SetIsActive(true).
			Save(ctx)
		return err
	}
	if err != nil {
		return err
	}

	_, err = s.client.User.UpdateOneID(existing.ID).
		SetPasswordHash(passwordHash).
		SetRole(DefaultAdminRole).
		SetIsActive(true).
		Save(ctx)
	return err
}

func (s *Service) ListUsers(ctx context.Context) ([]UserRecord, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnauthorized
	}
	items, err := s.client.User.Query().Order(ent.Asc(user.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UserRecord, 0, len(items))
	for _, item := range items {
		out = append(out, userRecordFromEnt(item))
	}
	return out, nil
}

func (s *Service) CreateUser(ctx context.Context, email, password, role string) (*UserRecord, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnauthorized
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	role, err := normalizeRole(role)
	if err != nil {
		return nil, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	created, err := s.client.User.Create().
		SetEmail(email).
		SetPasswordHash(passwordHash).
		SetRole(role).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	record := userRecordFromEnt(created)
	return &record, nil
}

func (s *Service) UpdateUser(ctx context.Context, id int, email, role string, isActive bool) (*UserRecord, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnauthorized
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	role, err := normalizeRole(role)
	if err != nil {
		return nil, err
	}
	item, err := s.client.User.UpdateOneID(id).
		SetEmail(email).
		SetRole(role).
		SetIsActive(isActive).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	record := userRecordFromEnt(item)
	return &record, nil
}

func (s *Service) SetUserPassword(ctx context.Context, id int, password string) error {
	if s == nil || s.client == nil {
		return ErrUnauthorized
	}
	if password == "" {
		return errors.New("password is required")
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.client.User.UpdateOneID(id).SetPasswordHash(passwordHash).Exec(ctx)
}

func (s *Service) SetUserActive(ctx context.Context, id int, isActive bool) (*UserRecord, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnauthorized
	}
	item, err := s.client.User.UpdateOneID(id).SetIsActive(isActive).Save(ctx)
	if err != nil {
		return nil, err
	}
	if !isActive {
		_, _ = s.client.WebSession.Delete().Where(websession.HasUserWith(user.IDEQ(id))).Exec(ctx)
	}
	record := userRecordFromEnt(item)
	return &record, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*UserInfo, string, time.Time, error) {
	if s == nil || s.client == nil {
		return nil, "", time.Time{}, ErrUnauthorized
	}
	if s.sessionSecret == "" {
		return nil, "", time.Time{}, errors.New("SESSION_SECRET is required for web authentication")
	}
	if strings.TrimSpace(email) == "" || password == "" {
		return nil, "", time.Time{}, ErrUnauthorized
	}

	u, err := s.client.User.Query().
		Where(user.EmailEQ(strings.TrimSpace(strings.ToLower(email))), user.IsActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, "", time.Time{}, ErrUnauthorized
		}
		return nil, "", time.Time{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", time.Time{}, ErrUnauthorized
	}

	rawToken, err := randomToken(32)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	expiresAt := s.now().Add(DefaultSessionTTL)

	if _, err := s.client.WebSession.Create().
		SetTokenHash(hashToken(rawToken)).
		SetExpiresAt(expiresAt).
		SetUserID(u.ID).
		Save(ctx); err != nil {
		return nil, "", time.Time{}, err
	}

	return userInfoFromEnt(u), s.signToken(rawToken), expiresAt, nil
}

func (s *Service) Session(ctx context.Context, signedToken string) (*UserInfo, error) {
	rawToken, err := s.verifySignedToken(signedToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	record, err := s.client.WebSession.Query().
		Where(websession.TokenHashEQ(hashToken(rawToken))).
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	now := s.now()
	if !record.ExpiresAt.After(now) || record.Edges.User == nil || !record.Edges.User.IsActive {
		_ = s.client.WebSession.DeleteOneID(record.ID).Exec(ctx)
		return nil, ErrUnauthorized
	}

	_, err = s.client.WebSession.UpdateOneID(record.ID).SetLastSeenAt(now).Save(ctx)
	if err != nil {
		return nil, err
	}

	return userInfoFromEnt(record.Edges.User), nil
}

func (s *Service) Logout(ctx context.Context, signedToken string) error {
	rawToken, err := s.verifySignedToken(signedToken)
	if err != nil {
		return nil
	}
	_, err = s.client.WebSession.Delete().
		Where(websession.TokenHashEQ(hashToken(rawToken))).
		Exec(ctx)
	return err
}

func (s *Service) SessionCookie(signedToken string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    signedToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure(),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

func (s *Service) ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure(),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func (s *Service) cookieSecure() bool {
	if s.baseURL == "" {
		return false
	}
	parsed, err := url.Parse(s.baseURL)
	return err == nil && strings.EqualFold(parsed.Scheme, "https")
}

func (s *Service) signToken(rawToken string) string {
	mac := hmac.New(sha256.New, []byte(s.sessionSecret))
	_, _ = mac.Write([]byte(rawToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return rawToken + "." + signature
}

func (s *Service) verifySignedToken(signedToken string) (string, error) {
	if s == nil || s.sessionSecret == "" {
		return "", ErrUnauthorized
	}
	parts := strings.Split(strings.TrimSpace(signedToken), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ErrUnauthorized
	}
	rawToken := parts[0]
	expected := s.signToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.TrimSpace(signedToken))) != 1 {
		return "", ErrUnauthorized
	}
	return rawToken, nil
}

func userInfoFromEnt(u *ent.User) *UserInfo {
	if u == nil {
		return nil
	}
	return &UserInfo{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	}
}

func userRecordFromEnt(u *ent.User) UserRecord {
	return UserRecord{
		ID:       u.ID,
		Email:    u.Email,
		Role:     u.Role,
		IsActive: u.IsActive,
	}
}

func normalizeRole(role string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case RoleAdmin:
		return RoleAdmin, nil
	case RoleStaff:
		return RoleStaff, nil
	default:
		return "", errors.New("role must be admin or staff")
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
