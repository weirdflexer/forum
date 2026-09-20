package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"forum/internal/forum"
)

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command, err := parseCommand(os.Args[1:])
	if err != nil {
		return err
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required; see .env.example")
	}
	var cfg serveConfig
	if command == "serve" {
		cfg, err = loadServeConfig(os.Getenv)
		if err != nil {
			return err
		}
	}

	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, err := forum.Open(startup, dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	switch command {
	case "migrate":
		return forum.Migrate(startup, db)
	case "seed":
		return forum.Seed(startup, db)
	case "create-staff":
		return forum.CreateStaff(startup, db, os.Getenv("STAFF_LOGIN"), os.Getenv("STAFF_PASSWORD"), envOr(os.Getenv, "STAFF_ROLE", "admin"))
	default:
		cancel() // The server has its own request and shutdown deadlines.
		return serve(ctx, forum.New(db, cfg.Forum), cfg)
	}
}

func parseCommand(args []string) (string, error) {
	if len(args) == 0 {
		return "serve", nil
	}
	switch args[0] {
	case "serve", "migrate", "seed", "create-staff":
		return args[0], nil
	default:
		return "", fmt.Errorf("command must be serve, migrate, seed or create-staff")
	}
}

func main() {
	if err := run(); err != nil {
		slog.Error(strings.TrimSpace(err.Error()))
		os.Exit(1)
	}
}
