package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://gitea.com"
	defaultAddr    = ":8080"
	defaultRefresh = 10 * time.Minute
)

type Config struct {
	BaseURL     string
	Tokens      []string
	Addr        string
	Refresh     time.Duration
	FeedURLSalt string
	RenderHTML  bool
}

func Load() (Config, error) {
	cfg := Config{
		BaseURL:     envDefault("GITEA_BASE_URL", defaultBaseURL),
		Addr:        envDefault("LISTEN_ADDR", defaultAddr),
		Refresh:     defaultRefresh,
		Tokens:      splitTokens(os.Getenv("GITEA_ACCESS_TOKENS")),
		FeedURLSalt: os.Getenv("FEED_URL_SALT"),
	}

	if raw := os.Getenv("REFRESH_INTERVAL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid REFRESH_INTERVAL %q: %w", raw, err)
		}
		cfg.Refresh = d
	}

	if len(cfg.Tokens) == 0 {
		return Config{}, fmt.Errorf("no access token: set GITEA_ACCESS_TOKENS (comma-separated for multiple tokens)")
	}
	if raw := os.Getenv("RENDER_HTML"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid RENDER_HTML %q: %w", raw, err)
		}
		cfg.RenderHTML = enabled
	}

	return cfg, nil
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitTokens(raw string) []string {
	var tokens []string
	for part := range strings.SplitSeq(raw, ",") {
		if t := strings.TrimSpace(part); t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}
