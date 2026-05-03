// Package storage provides repository implementations for PostgreSQL and MongoDB.
package storage

import (
	"fmt"

	"db_sync/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// InitDB opens and validates a PostgreSQL connection using the global SQLConnect config.
func InitDB() (*sqlx.DB, error) {
	cfg := config.SQLConnect

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return db, nil
}
