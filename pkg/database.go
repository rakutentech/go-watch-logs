package pkg

import (
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite" // nolint: revive
)

var (
	db *sql.DB
)

func InitDB(dbName string) (*sql.DB, error) {
	var err error
	if db == nil {
		slog.Info("Initializing database", "dbName", dbName)
		db, err = sql.Open("sqlite", dbName)
		if err != nil {
			slog.Error("Error opening database", "error", err.Error())
			return nil, err
		}
	} else {
		slog.Info("Database already initialized")
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value INTEGER
		)
	`)
	if err != nil {
		slog.Error("Error creating state table", "error", err.Error())
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS plot (
			key TEXT PRIMARY KEY,
			value INTEGER
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		slog.Error("Error creating plot table", "error", err.Error())
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}
func Vacuum(dbName string) error {
	if _, err := os.Stat(dbName); err == nil {
		if err := os.Remove(dbName); err != nil {
			return err
		}
	}
	return nil
}
