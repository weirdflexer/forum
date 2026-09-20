package main

import (
	"strings"
	"testing"
)

func TestServeConfig(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantError string
	}{
		{name: "defaults"},
		{name: "secure remote", env: map[string]string{"APP_ORIGIN": "https://forum.example", "COOKIE_SECURE": "true"}},
		{name: "short secret", env: map[string]string{"SESSION_SECRET": "short"}, wantError: "SESSION_SECRET"},
		{name: "insecure remote", env: map[string]string{"APP_ORIGIN": "http://forum.example"}, wantError: "COOKIE_SECURE"},
		{name: "secure local HTTP", env: map[string]string{"COOKIE_SECURE": "true"}, wantError: "https"},
		{name: "invalid boolean", env: map[string]string{"COOKIE_SECURE": "yes"}, wantError: "COOKIE_SECURE"},
		{name: "zero writes", env: map[string]string{"NETWORK_WRITE_LIMIT": "0"}, wantError: "NETWORK_WRITE_LIMIT"},
		{name: "negative sessions", env: map[string]string{"NETWORK_SESSION_LIMIT": "-1"}, wantError: "NETWORK_SESSION_LIMIT"},
		{name: "invalid sessions", env: map[string]string{"NETWORK_SESSION_LIMIT": "lots"}, wantError: "NETWORK_SESSION_LIMIT"},
	}
	for _, origin := range []string{"http://localhost:", "http://localhost:80", "https://localhost:443", "http://localhost:0", "http://localhost:65536", "http://localhost:abc", "http://localhost/", "http://localhost/path", "http://localhost?q=1", "http://localhost?", "http://localhost#part", "http://localhost#", "http://user:pass@localhost", "http://:8080", "ftp://localhost", "//localhost", "://broken"} {
		tests = append(tests, struct {
			name      string
			env       map[string]string
			wantError string
		}{name: "origin " + origin, env: map[string]string{"APP_ORIGIN": origin}, wantError: "APP_ORIGIN"})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string {
				if value, ok := tc.env[key]; ok {
					return value
				}
				if key == "SESSION_SECRET" {
					return strings.Repeat("s", 32)
				}
				return ""
			}
			cfg, err := loadServeConfig(getenv)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error about %s, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Addr != "127.0.0.1:8080" || cfg.Forum.NetworkWriteLimit != 120 || cfg.Forum.NetworkSessionLimit != 20 {
				t.Fatalf("unexpected defaults: %#v", cfg)
			}
		})
	}
}

func TestServeConfigOverrides(t *testing.T) {
	values := map[string]string{"SESSION_SECRET": strings.Repeat("s", 32), "API_ADDR": "127.0.0.1:18080", "APP_ORIGIN": "http://127.0.0.1:18080", "STATIC_DIR": "./dist", "NETWORK_WRITE_LIMIT": "50", "NETWORK_SESSION_LIMIT": "7"}
	cfg, err := loadServeConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != values["API_ADDR"] || cfg.Forum.Origin != values["APP_ORIGIN"] || cfg.Forum.StaticDir != "./dist" || cfg.Forum.NetworkWriteLimit != 50 || cfg.Forum.NetworkSessionLimit != 7 {
		t.Fatalf("overrides lost: %#v", cfg)
	}
}

func TestParseCommand(t *testing.T) {
	for _, command := range []string{"serve", "migrate", "seed", "create-staff"} {
		got, err := parseCommand([]string{command})
		if err != nil || got != command {
			t.Fatalf("%s: %q %v", command, got, err)
		}
	}
	if got, err := parseCommand(nil); err != nil || got != "serve" {
		t.Fatalf("default: %q %v", got, err)
	}
	if _, err := parseCommand([]string{"invalid"}); err == nil {
		t.Fatal("unknown command accepted")
	}
}
