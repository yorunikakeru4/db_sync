// Package storage provides backend write-side PostgreSQL persistence helpers.
package storage

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// OpenBunDB opens and validates a bun PostgreSQL connection.
func OpenBunDB(dsn string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	if err := sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("open bun postgres: %w", err)
	}

	return bun.NewDB(sqldb, pgdialect.New()), nil
}
