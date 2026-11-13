package gitea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "code.gitea.io/sdk/gitea"
)

func TestRenderIssueBody(t *testing.T) {
	for _, subpath := range []string{"", "/gitea"} {
		t.Run("subpath="+subpath, func(t *testing.T) {
			var requests int
			var fail bool
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == subpath+"/api/v1/version" {
					_, _ = w.Write([]byte(`{"version":"1.25.0"}`))
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != subpath+"/api/v1/markup" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				requests++
				var opt sdk.MarkupOption
				if err := json.NewDecoder(r.Body).Decode(&opt); err != nil {
					t.Error(err)
				}
				if opt.Text != "#10 abcdef123 @alice" || opt.Mode != "comment" || opt.Context != subpath+"/alice/project" {
					t.Errorf("incorrect rendering context: %#v", opt)
				}
				if r.Header.Get("Authorization") != "token test-token" {
					t.Error("rendering must use the feed's token")
				}
				if fail {
					http.Error(w, "unavailable", http.StatusBadGateway)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<p>rendered</p>"))
			}))
			defer api.Close()
			client, err := NewClient(api.URL+subpath, "test-token")
			if err != nil {
				t.Fatal(err)
			}
			issue := &sdk.Issue{ID: 1, Body: "#10 abcdef123 @alice", Repository: &sdk.RepositoryMeta{FullName: "alice/project"}}
			if html, err := client.RenderIssueBody(issue); err != nil || html != "<p>rendered</p>" {
				t.Fatalf("rendered body: %q, %v", html, err)
			}
			fail = true
			if _, err := client.RenderIssueBody(issue); err == nil {
				t.Fatal("expected rendering error")
			}
			issue.Repository = nil
			if _, err := client.RenderIssueBody(issue); err == nil || !strings.Contains(err.Error(), "missing repository") {
				t.Fatalf("missing repository must fail: %v", err)
			}
			issue.Body = " \r\n "
			if html, err := client.RenderIssueBody(issue); err != nil || html != "" || requests != 2 {
				t.Fatalf("empty body must not request rendering: %q, %v, requests=%d", html, err, requests)
			}
		})
	}
}

func TestIssuesWithDueDatePaginationAndFailure(t *testing.T) {
	deadline := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	var failPageTwo atomic.Bool
	failPageTwo.Store(true)
	var pages atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/version" {
			_, _ = w.Write([]byte(`{"version":"1.25.0"}`))
			return
		}
		if r.URL.Path != "/api/v1/repos/issues/search" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		if query.Get("type") != "issues" || query.Get("state") != "open" || query.Get("since") != "" {
			t.Errorf("expected full open-issue query excluding PRs: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "token test-token" {
			t.Error("request must use the account token")
		}
		pages.Add(1)
		if query.Get("page") != "2" {
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/api/v1/repos/issues/search?page=2>; rel="next"`, r.Host))
			issues := make([]*sdk.Issue, pageSize)
			for i := range issues {
				issues[i] = &sdk.Issue{ID: int64(i + 10)}
			}
			issues[0] = &sdk.Issue{ID: 1, Deadline: &deadline}
			_ = json.NewEncoder(w).Encode(issues)
			return
		}
		if failPageTwo.Load() {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode([]*sdk.Issue{{ID: 2, Deadline: &deadline}, {ID: 3}})
	}))
	defer api.Close()
	c, err := NewClient(api.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	if issues, err := c.IssuesWithDueDate(); err == nil || issues != nil {
		t.Fatal("second-page failure must not return partial results")
	}
	failPageTwo.Store(false)
	issues, err := c.IssuesWithDueDate()
	if err != nil || len(issues) != 2 || issues[0].ID != 1 || issues[1].ID != 2 {
		t.Fatalf("expected due-date issues from both pages: issues=%v error=%v", issues, err)
	}
	if pages.Load() != 4 {
		t.Fatalf("expected two full queries of two pages, got %d requests", pages.Load())
	}
}

func TestCurrentUser(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			_, _ = w.Write([]byte(`{"version":"1.25.0"}`))
		case "/api/v1/user":
			_, _ = w.Write([]byte(`{"login":"alice"}`))
		default:
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	c, err := NewClient(api.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	if user, err := c.CurrentUser(); err != nil || user.UserName != "alice" {
		t.Fatalf("current user: user=%v error=%v", user, err)
	}
}
