package db

import (
	"database/sql"
	"fmt"
	"log"

	"jevon/internal/config"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection pool and verifies connectivity.
func Connect(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	log.Printf("✅ PostgreSQL connected — %s:%s/%s", cfg.Host, cfg.Port, cfg.Name)
	return db, nil
}
