package web

import (
	"bytes"
	"langschool/internal/app/releasenotice"
	"net/http"
	"testing"
)

func TestReleaseNoticeHTTPRequiresAccountAndSharesDismissal(t *testing.T) {
	env := newAnonymousTestServer(t)
	defer env.Close()
	for _, path := range []string{"/api/me/release-notices/current/visit", "/api/me/release-notices/test/dismiss"} {
		res, err := env.Client.Post(env.Server.URL+path, "application/json", bytes.NewBufferString("{}"))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: %d", path, res.StatusCode)
		}
	}
	env.login(t)
	first := postJSON[releasenotice.Result](t, env.Client, env.Server.URL, "/api/me/release-notices/current/visit", map[string]any{})
	if first.Notice == nil {
		t.Fatal("missing announcement")
	}
	res, err := env.Client.Post(env.Server.URL+"/api/me/release-notices/"+first.Notice.ReleaseID+"/dismiss", "application/json", bytes.NewBufferString("{}"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("dismiss: %d", res.StatusCode)
	}
	env.login(t)
	again := postJSON[releasenotice.Result](t, env.Client, env.Server.URL, "/api/me/release-notices/current/visit", map[string]any{})
	if again.Notice == nil || again.Notice.DismissedAt == nil || !again.Notice.FirstSeenAt.Equal(first.Notice.FirstSeenAt) {
		t.Fatal("new session lost state")
	}
}
