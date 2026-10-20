// Package releasenotice serves release announcements and account-wide display windows.
package releasenotice

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"time"

	"langschool/ent"
	"langschool/ent/releasenoticeview"
)

// Change the ID only when publishing a new announcement. Rebuilding the same
// release must never restart a user's 24-hour window.
//
//go:embed current.json
var currentJSON []byte

type Section struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}
type Content struct {
	Title    string    `json:"title"`
	Intro    string    `json:"intro"`
	Sections []Section `json:"sections"`
}
type Manifest struct {
	ID      string             `json:"id"`
	Content map[string]Content `json:"content"`
}
type Notice struct {
	ReleaseID   string             `json:"releaseId"`
	FirstSeenAt time.Time          `json:"firstSeenAt"`
	ExpiresAt   time.Time          `json:"expiresAt"`
	DismissedAt *time.Time         `json:"dismissedAt"`
	Content     map[string]Content `json:"content"`
}
type Result struct {
	ServerNow time.Time `json:"serverNow"`
	Notice    *Notice   `json:"notice"`
}
type Service struct {
	db       *ent.Client
	now      func() time.Time
	manifest Manifest
}

func New(db *ent.Client) *Service {
	var manifest Manifest
	if err := json.Unmarshal(currentJSON, &manifest); err != nil {
		panic(err)
	}
	return &Service{db: db, now: time.Now, manifest: manifest}
}
func (s *Service) Visit(ctx context.Context, userID int) (*Result, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Take the timestamp after acquiring SQLite's writer, so concurrent visits
	// return the original start instead of resetting it or using a stale deadline.
	now := s.now().UTC()
	row, err := tx.ReleaseNoticeView.Query().Where(releasenoticeview.UserIDEQ(userID), releasenoticeview.ReleaseIDEQ(s.manifest.ID)).Only(ctx)
	if ent.IsNotFound(err) {
		row, err = tx.ReleaseNoticeView.Create().SetUserID(userID).SetReleaseID(s.manifest.ID).SetFirstSeenAt(now).Save(ctx)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result := &Result{ServerNow: now}
	expires := row.FirstSeenAt.Add(24 * time.Hour)
	if now.Before(expires) {
		result.Notice = &Notice{ReleaseID: s.manifest.ID, FirstSeenAt: row.FirstSeenAt, ExpiresAt: expires, DismissedAt: row.DismissedAt, Content: s.manifest.Content}
	}
	return result, nil
}
func (s *Service) Dismiss(ctx context.Context, userID int, releaseID string) error {
	// An old tab may close an earlier announcement after a new deployment.
	// Only update this account's existing view; never create or reset a window.
	_, err := s.db.ReleaseNoticeView.Update().Where(releasenoticeview.UserIDEQ(userID), releasenoticeview.ReleaseIDEQ(releaseID), releasenoticeview.DismissedAtIsNil()).SetDismissedAt(s.now().UTC()).Save(ctx)
	return err
}
