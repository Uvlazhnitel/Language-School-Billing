package releasenotice

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"langschool/ent"
	"langschool/ent/enttest"
)

func TestAccountWindowDismissalAndNextRelease(t *testing.T) {
	ctx := context.Background()
	db := enttest.Open(t, "sqlite3", "file:notices?mode=memory&_fk=1")
	defer db.Close()
	a := db.User.Create().SetUsername("a").SetPasswordHash("hash").SaveX(ctx)
	b := db.User.Create().SetUsername("b").SetPasswordHash("hash").SaveX(ctx)
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	svc := New(db)
	svc.now = func() time.Time { return now }
	first, err := svc.Visit(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Notice == nil || first.Notice.DismissedAt != nil || !first.Notice.ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("first = %+v", first)
	}
	id := first.Notice.ReleaseID
	for _, locale := range []string{"ru-RU", "en-US", "lv-LV"} {
		if content := first.Notice.Content[locale]; content.Title == "" || len(content.Sections) != 6 {
			t.Fatalf("bad content: %s", locale)
		}
	}
	now = now.Add(time.Hour)
	secondDevice := New(db)
	secondDevice.now = svc.now
	again, err := secondDevice.Visit(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Notice.FirstSeenAt.Equal(first.Notice.FirstSeenAt) {
		t.Fatal("window reset on another device")
	}
	if err := svc.Dismiss(ctx, a.ID, id); err != nil {
		t.Fatal(err)
	}
	dismissedAt := now
	now = now.Add(time.Hour)
	if err := svc.Dismiss(ctx, a.ID, id); err != nil {
		t.Fatal(err)
	}
	again, err = secondDevice.Visit(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Notice.DismissedAt == nil || !again.Notice.DismissedAt.Equal(dismissedAt) {
		t.Fatal("dismissal not shared/idempotent")
	}
	other, err := svc.Visit(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if other.Notice.DismissedAt != nil || !other.Notice.FirstSeenAt.Equal(now) {
		t.Fatal("accounts share state")
	}
	now = first.Notice.ExpiresAt.Add(-time.Nanosecond)
	before, err := svc.Visit(ctx, a.ID)
	if err != nil || before.Notice == nil {
		t.Fatal("expired early", err)
	}
	now = first.Notice.ExpiresAt
	expired, err := svc.Visit(ctx, a.ID)
	if err != nil || expired.Notice != nil {
		t.Fatal("available at boundary", err)
	}
	nextRelease := New(db)
	nextRelease.now = svc.now
	nextRelease.manifest.ID = "next-release"
	next, err := nextRelease.Visit(ctx, a.ID)
	if err != nil || next.Notice == nil || next.Notice.DismissedAt != nil || !next.Notice.FirstSeenAt.Equal(now) {
		t.Fatal("next release did not start fresh", err)
	}
	// Deleting a user must not be blocked by announcement bookkeeping.
	if err := db.User.DeleteOneID(a.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if count := db.ReleaseNoticeView.Query().CountX(ctx); count != 1 {
		t.Fatalf("cascade left %d rows", count)
	}
}
func TestConcurrentVisitsAndRestartKeepOriginalWindow(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "notices.sqlite") + "?_fk=1&_busy_timeout=5000"
	db := enttest.Open(t, "sqlite3", dsn)
	u := db.User.Create().SetUsername("user").SetPasswordHash("hash").SaveX(ctx)
	svc := New(db)
	var wg sync.WaitGroup
	results := make(chan *Result, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := svc.Visit(ctx, u.ID)
			if err != nil {
				failures <- err
			} else {
				results <- result
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	var first *Result
	for result := range results {
		if first == nil {
			first = result
		}
		if !result.Notice.FirstSeenAt.Equal(first.Notice.FirstSeenAt) {
			t.Fatal("concurrent visit reset deadline")
		}
	}
	if count := db.ReleaseNoticeView.Query().CountX(ctx); count != 1 {
		t.Fatalf("duplicate views: %d", count)
	}
	db.Close()
	reopened, err := ent.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := New(reopened).Visit(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Notice.ExpiresAt.Equal(first.Notice.ExpiresAt) {
		t.Fatal("restart reset deadline")
	}
}
func TestDismissUnknownDoesNotStartWindow(t *testing.T) {
	ctx := context.Background()
	db := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&_fk=1", t.Name()))
	defer db.Close()
	u := db.User.Create().SetUsername("user").SetPasswordHash("hash").SaveX(ctx)
	if err := New(db).Dismiss(ctx, u.ID, "unknown"); err != nil {
		t.Fatal(err)
	}
	if db.ReleaseNoticeView.Query().CountX(ctx) != 0 {
		t.Fatal("dismiss created visit")
	}
}
