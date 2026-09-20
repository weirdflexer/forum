package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"forum/internal/forum"
)

type serveConfig struct {
	Addr  string
	Forum forum.Config
}

func envOr(getenv func(string) string, key, fallback string) string {
	if value := getenv(key); value != "" {
		return value
	}
	return fallback
}

// loadServeConfig validates configuration before opening a database connection.
func loadServeConfig(getenv func(string) string) (serveConfig, error) {
	cfg := serveConfig{Addr: envOr(getenv, "API_ADDR", "127.0.0.1:8080")}
	cfg.Forum.Secret = getenv("SESSION_SECRET")
	if len(cfg.Forum.Secret) < 32 {
		return cfg, fmt.Errorf("SESSION_SECRET must have at least 32 characters")
	}
	cfg.Forum.Origin = envOr(getenv, "APP_ORIGIN", "http://localhost:5173")
	u, err := url.Parse(cfg.Forum.Origin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return cfg, fmt.Errorf("APP_ORIGIN must be an origin without credentials, a path, query or fragment")
	}
	// Reject even an empty fragment, which net/url otherwise discards.
	if u.String() != cfg.Forum.Origin {
		return cfg, fmt.Errorf("APP_ORIGIN must be an origin without credentials, a path, query or fragment")
	}
	// Browser Origin omits default ports. Reject forms that could never match
	// the exact Origin guard rather than starting a server that denies writes.
	if strings.HasSuffix(u.Host, ":") || (u.Scheme == "http" && u.Port() == "80") || (u.Scheme == "https" && u.Port() == "443") {
		return cfg, fmt.Errorf("APP_ORIGIN must omit empty or default ports")
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return cfg, fmt.Errorf("invalid APP_ORIGIN port")
		}
	}
	cfg.Forum.SecureCookies, err = strconv.ParseBool(envOr(getenv, "COOKIE_SECURE", "false"))
	if err != nil {
		return cfg, fmt.Errorf("invalid COOKIE_SECURE")
	}
	if !cfg.Forum.SecureCookies && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		return cfg, fmt.Errorf("non-local deployment requires COOKIE_SECURE=true")
	}
	if cfg.Forum.SecureCookies && u.Scheme != "https" {
		return cfg, fmt.Errorf("secure cookies require https APP_ORIGIN")
	}
	cfg.Forum.NetworkWriteLimit, err = positiveEnv(getenv, "NETWORK_WRITE_LIMIT", "120")
	if err != nil {
		return cfg, err
	}
	cfg.Forum.NetworkSessionLimit, err = positiveEnv(getenv, "NETWORK_SESSION_LIMIT", "20")
	if err != nil {
		return cfg, err
	}
	cfg.Forum.StaticDir = getenv("STATIC_DIR")
	return cfg, nil
}

func positiveEnv(getenv func(string) string, key, fallback string) (int, error) {
	value, err := strconv.Atoi(envOr(getenv, key, fallback))
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return value, nil
}
