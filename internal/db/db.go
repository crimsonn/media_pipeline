package db

import (
	"context"
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

func OpenDatabase() (*sql.DB, error) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "storage.db")
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, err
	}

	return db, nil
}
