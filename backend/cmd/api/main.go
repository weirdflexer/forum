package main

import (
	"context"
	"fmt"
	"forum/internal/forum"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func env(k, fallback string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return fallback
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required; see .env.example")
	}
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, err := forum.Open(startup, dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "migrate":
		return forum.Migrate(startup, db)
	case "seed":
		return forum.Seed(startup, db)
	case "create-staff":
		return forum.CreateStaff(startup, db, os.Getenv("STAFF_LOGIN"), os.Getenv("STAFF_PASSWORD"), env("STAFF_ROLE", "admin"))
	case "serve":
	default:
		return fmt.Errorf("command must be serve, migrate, seed or create-staff")
	}
	secret := os.Getenv("SESSION_SECRET")
	if len(secret) < 32 {
		return fmt.Errorf("SESSION_SECRET must have at least 32 characters")
	}
	origin := env("APP_ORIGIN", "http://localhost:5173")
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.Path != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("APP_ORIGIN must be an origin without a path or trailing slash")
	}
	secure, err := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil {
		return err
	}
	if !secure && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
		return fmt.Errorf("non-local deployment requires COOKIE_SECURE=true")
	}
	if secure && u.Scheme != "https" {
		return fmt.Errorf("secure cookies require https APP_ORIGIN")
	}
	writeLimit, err := strconv.Atoi(env("NETWORK_WRITE_LIMIT", "120"))
	if err != nil || writeLimit < 1 {
		return fmt.Errorf("invalid NETWORK_WRITE_LIMIT")
	}
	sessionLimit, err := strconv.Atoi(env("NETWORK_SESSION_LIMIT", "20"))
	if err != nil || sessionLimit < 1 {
		return fmt.Errorf("invalid NETWORK_SESSION_LIMIT")
	}
	app := forum.New(db, forum.Config{Origin: origin, Secret: secret, SecureCookies: secure, StaticDir: os.Getenv("STATIC_DIR"), NetworkWriteLimit: writeLimit, NetworkSessionLimit: sessionLimit})
	srv := &http.Server{Addr: env("API_ADDR", "127.0.0.1:8080"), Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	errc := make(chan error, 1)
	go func() {
		slog.Info("forum started", "address", srv.Addr, "origin", origin)
		errc <- srv.ListenAndServe()
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		app.Cleanup(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app.Cleanup(ctx)
			}
		}
	}()
	select {
	case err = <-errc:
		if err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(c)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		slog.Error(strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}
