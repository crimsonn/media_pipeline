package db

import (
	"context"
	"fmt"
	"strings"

	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var ddl string

func OpenDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := applySchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func applySchema(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range strings.Split(ddl, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}
	return nil
}
