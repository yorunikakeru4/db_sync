package storage

import (
	"database/sql"
	"fmt"

	"db_sync/internal/config"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// OpenBunDB opens and validates a bun PostgreSQL connection using the global SQLConnect config.
func OpenBunDB() (*bun.DB, error) {
	cfg := config.SQLConnect

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	))))

	if err := sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("failed to connect to postgres with bun: %w", err)
	}

	return bun.NewDB(sqldb, pgdialect.New()), nil
}
