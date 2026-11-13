package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	// pgx registers the "pgx" driver via its stdlib adapter. The import is
	// used for its side effect of registering the driver.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect opens a connection pool to Postgres and pings it, retrying for up
// to timeout. The DB container may still be warming up even after its
// healthcheck passes, so a short retry loop avoids startup races.
func Connect(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := db.PingContext(ctx); err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			slog.Info("connected to database")
			return db, nil
		} else {
			lastErr = err
			slog.Warn("db ping failed, retrying", "error", err)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	_ = db.Close()
	return nil, fmt.Errorf("db connect timeout: %w", lastErr)
}
