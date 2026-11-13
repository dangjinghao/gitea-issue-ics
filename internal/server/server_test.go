package server

import (
	"encoding/hex"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitea-issue-ics/internal/config"
)

func TestFeedPath(t *testing.T) {
	path := feedPath("https://gitea.example", "secret-token", "deployment-salt")
	decoded, err := hex.DecodeString(path)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("expected SHA-256 hex path, got %q", path)
	}
	if feedPath("https://gitea.example/", "secret-token", "deployment-salt") != path {
		t.Fatal("trailing slash changed path")
	}
	for _, changed := range []string{
		feedPath("https://other.example", "secret-token", "deployment-salt"),
		feedPath("https://gitea.example", "other-token", "deployment-salt"),
		feedPath("https://gitea.example", "secret-token", "other-salt"),
	} {
		if changed == path {
			t.Fatal("changing base URL, token, or salt must change path")
		}
	}
}

func TestSubscriptionSurvivesRestart(t *testing.T) {
	api := newTestAPI(t)
	for _, salt := range []string{"", "deployment-salt"} {
		t.Run("salt="+salt, func(t *testing.T) {
			cfg := config.Config{BaseURL: api.URL, Tokens: []string{"first-token", "second-token"}, Refresh: time.Minute, FeedURLSalt: salt}
			before, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Tokens = []string{"second-token", "first-token", "second-token"}
			after, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if len(after.feeds) != 2 || before.feeds[0].path != after.feeds[1].path {
				t.Fatal("restart, token reordering or duplicates changed subscriptions")
			}
			path := "/" + before.feeds[0].path + "/issue-deadlines.ics"
			handler := after.Handler()
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/calendar; charset=utf-8" {
				t.Fatalf("subscription returned %d: %s", response.Code, response.Body.String())
			}
			body := strings.ReplaceAll(response.Body.String(), "\r\n", "\n")
			for _, property := range []string{"NAME:", "X-WR-CALNAME:"} {
				if !strings.Contains(body, "\n"+property+"alice@127.0.0.1/issue-deadlines\n") {
					t.Fatalf("calendar name missing: %s", body)
				}
			}
			_, params, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
			if err != nil || params["filename"] != "issue-deadlines.ics" {
				t.Fatalf("unexpected download filename: %v, %v", params, err)
			}
			for _, missing := range []string{"/", "/unknown/issue-deadlines.ics", "/" + before.feeds[0].path} {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, missing, nil))
				if response.Code != http.StatusNotFound {
					t.Errorf("unexpected status for %s: %d", missing, response.Code)
				}
			}
		})
	}
}

func TestHTTPStaleFallback(t *testing.T) {
	api := newTestAPI(t)
	cfg := config.Config{BaseURL: api.URL, Tokens: []string{"test-token"}, Refresh: -time.Second}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := "/" + s.feeds[0].path + "/issue-deadlines.ics"
	handler := s.Handler()
	request := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	first := request()
	if first.Code != http.StatusOK {
		t.Fatalf("initial response: %d", first.Code)
	}
	api.fail.Store(true)
	stale := request()
	if stale.Code != http.StatusOK || stale.Body.String() != first.Body.String() {
		t.Fatal("failed refresh must serve the previous calendar")
	}
	uncached, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	handler = uncached.Handler()
	if response := request(); response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without cache, got %d", response.Code)
	}
}

func TestFeedCacheAndStaleFallback(t *testing.T) {
	api := newTestAPI(t)
	cfg := config.Config{BaseURL: api.URL, Tokens: []string{"test-token"}, Refresh: time.Hour}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	f := s.feeds[0]
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if data, err := f.get(); err != nil || !strings.Contains(data, "BEGIN:VCALENDAR") {
				t.Errorf("initial fetch: data=%q error=%v", data, err)
			}
		}()
	}
	wg.Wait()
	if api.issueRequests.Load() != 1 {
		t.Fatalf("concurrent requests fetched issues %d times", api.issueRequests.Load())
	}
	cached, err := f.get()
	if err != nil || api.issueRequests.Load() != 1 {
		t.Fatal("fresh cache must not fetch issues")
	}
	f.mu.Lock()
	f.fetchedAt = time.Now().Add(-2 * cfg.Refresh)
	previous := f.fetchedAt
	f.mu.Unlock()
	api.fail.Store(true)
	if data, err := f.get(); data != cached || err == nil || !strings.Contains(err.Error(), "serving stale") {
		t.Fatalf("expected stale calendar on refresh failure: data=%q error=%v", data, err)
	}
	if !f.fetchedAt.Equal(previous) {
		t.Fatal("failed refresh must not mark old cache as fresh")
	}
	api.fail.Store(false)
	if data, err := f.get(); err != nil || data == "" || !f.fetchedAt.After(previous) {
		t.Fatalf("feed did not recover after failed refresh: %v", err)
	}
}

type testAPI struct {
	*httptest.Server
	issueRequests atomic.Int32
	fail          atomic.Bool
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	api := &testAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_, _ = w.Write([]byte(`{"version":"1.25.0"}`))
		case "/api/v1/user":
			_, _ = w.Write([]byte(`{"id":1,"login":"alice"}`))
		case "/api/v1/repos/issues/search":
			api.issueRequests.Add(1)
			if api.fail.Load() {
				http.Error(w, "unavailable", http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Errorf("unexpected Gitea request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)
	return api
}
