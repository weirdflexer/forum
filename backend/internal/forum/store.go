package forum

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL")
	}
	cfg.MaxConns = 12
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL unavailable: %w", err)
	}
	return db, nil
}
