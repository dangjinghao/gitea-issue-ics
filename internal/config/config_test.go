package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func resetConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GITEA_BASE_URL", "GITEA_ACCESS_TOKENS", "LISTEN_ADDR", "REFRESH_INTERVAL", "FEED_URL_SALT", "ICAL_REFRESH_INTERVAL"} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("GITEA_ACCESS_TOKENS", " first , , second, ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://gitea.com" || cfg.Addr != ":8080" || cfg.Refresh != 10*time.Minute || cfg.FeedURLSalt != "" || cfg.ICalRefresh != time.Hour {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
	if !reflect.DeepEqual(cfg.Tokens, []string{"first", "second"}) {
		t.Fatalf("unexpected tokens: %v", cfg.Tokens)
	}
}

func TestLoadOverrides(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("GITEA_ACCESS_TOKENS", "token")
	t.Setenv("GITEA_BASE_URL", "https://git.example/gitea")
	t.Setenv("LISTEN_ADDR", "127.0.0.1:9090")
	t.Setenv("REFRESH_INTERVAL", "5m")
	t.Setenv("ICAL_REFRESH_INTERVAL", "90m")
	t.Setenv("FEED_URL_SALT", "deployment-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://git.example/gitea" || cfg.Addr != "127.0.0.1:9090" || cfg.Refresh != 5*time.Minute || cfg.FeedURLSalt != "deployment-secret" || cfg.ICalRefresh != 90*time.Minute {
		t.Fatalf("overrides not applied: %#v", cfg)
	}
}

func TestLoadErrors(t *testing.T) {
	for _, raw := range []string{"invalid", "PT1H", "0s", "-1h", "500ms", "1.5s"} {
		t.Run("calendar interval="+raw, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("GITEA_ACCESS_TOKENS", "token")
			t.Setenv("ICAL_REFRESH_INTERVAL", raw)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ICAL_REFRESH_INTERVAL") {
				t.Fatalf("expected calendar interval validation error, got %v", err)
			}
		})
	}
	t.Run("missing tokens", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("GITEA_ACCESS_TOKENS", " , ")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "no access token") {
			t.Fatalf("expected missing-token error, got %v", err)
		}
	})
	t.Run("invalid interval", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("GITEA_ACCESS_TOKENS", "token")
		t.Setenv("REFRESH_INTERVAL", "not-a-duration")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "invalid REFRESH_INTERVAL") {
			t.Fatalf("expected duration error, got %v", err)
		}
	})
}
