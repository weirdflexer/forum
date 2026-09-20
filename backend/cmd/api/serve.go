package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"forum/internal/forum"
)

func serve(ctx context.Context, app *forum.Server, cfg serveConfig) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	srv := &http.Server{
		Addr: cfg.Addr, Handler: app.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 * 1024,
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("forum started", "address", srv.Addr, "origin", cfg.Forum.Origin)
		errc <- srv.ListenAndServe()
	}()
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		cleanupLoop(ctx, app)
	}()
	defer func() {
		cancel()
		<-cleanupDone // The database stays open until the worker exits.
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return err
		}
	}
	return nil
}

func cleanupLoop(ctx context.Context, app *forum.Server) {
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
}
