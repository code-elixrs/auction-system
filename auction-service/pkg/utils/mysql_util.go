package utils

import (
	"auction-system/internal/config"
	"auction-system/pkg/logger"
	"context"
	"database/sql"
	"os"
)

func InitializeMysql(cfg *config.Config, log logger.Logger, ctx context.Context) *sql.DB {
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		log.Error("Failed to connect to MySQL", "error", err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)

	// Test MySQL connection
	if err := db.PingContext(ctx); err != nil {
		log.Error("Failed to ping MySQL", "error", err)
		os.Exit(1)
	}
	return db
}
