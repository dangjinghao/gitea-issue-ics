package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gitea-issue-ics/internal/calendar"
	"gitea-issue-ics/internal/config"
	"gitea-issue-ics/internal/gitea"
)

// Thunderbird derives the default ICS subscription name from the URL's last
// path segment rather than NAME/X-WR-CALNAME, so use a readable filename.
const feedFilename = "issue-deadlines.ics"

type feed struct {
	path        string
	client      *gitea.Client
	calName     string
	refresh     time.Duration
	icalRefresh time.Duration

	mu        sync.Mutex
	ics       string
	fetchedAt time.Time
}

type Server struct {
	feeds []*feed
}

func New(cfg config.Config) (*Server, error) {
	if cfg.ICalRefresh == 0 {
		cfg.ICalRefresh = time.Hour
	}
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Gitea base URL: %w", err)
	}
	if baseURL.Hostname() == "" {
		return nil, fmt.Errorf("Gitea base URL must include a hostname")
	}
	s := &Server{}
	seen := make(map[string]bool)
	for i, token := range cfg.Tokens {
		path := feedPath(cfg.BaseURL, token, cfg.FeedURLSalt)
		if seen[path] {
			continue
		}
		seen[path] = true
		client, err := gitea.NewClient(cfg.BaseURL, token)
		if err != nil {
			return nil, err
		}
		user, err := client.CurrentUser()
		if err != nil {
			return nil, fmt.Errorf("token #%d: %w", i+1, err)
		}
		f := &feed{
			path:        path,
			client:      client,
			calName:     fmt.Sprintf("%s@%s/issue-deadlines", user.UserName, baseURL.Hostname()),
			refresh:     cfg.Refresh,
			icalRefresh: cfg.ICalRefresh,
		}
		s.feeds = append(s.feeds, f)
		log.Printf("token #%d (%s): subscribe with GET /%s/%s", i+1, user.UserName, f.path, feedFilename)
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	for _, f := range s.feeds {
		mux.HandleFunc(fmt.Sprintf("GET /%s/%s", f.path, feedFilename), f.serve)
	}
	return mux
}

func (f *feed) serve(w http.ResponseWriter, r *http.Request) {
	data, err := f.get()
	if err != nil {
		log.Printf("feed %s: %v", f.path, err)
	}
	if data == "" {
		http.Error(w, "calendar unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+feedFilename+`"`)
	_, _ = w.Write([]byte(data))
}

func (f *feed) get() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ics != "" && time.Since(f.fetchedAt) < f.refresh {
		return f.ics, nil
	}

	issues, err := f.client.IssuesWithDueDate()
	if err != nil {
		if f.ics != "" {
			return f.ics, fmt.Errorf("refresh failed, serving stale: %w", err)
		}
		return "", fmt.Errorf("refresh failed: %w", err)
	}

	data := calendar.Build(f.calName, issues, f.icalRefresh)

	f.ics = data
	f.fetchedAt = time.Now()
	return data, nil
}

func feedPath(baseURL, token, salt string) string {
	// The token is the secret key; the context separates feed URLs from other uses.
	mac := hmac.New(sha256.New, []byte(token))
	fmt.Fprintf(mac, "gitea-issue-ics/feed/v1\x00%s\x00%s", strings.TrimRight(baseURL, "/"), salt)
	return hex.EncodeToString(mac.Sum(nil))
}
