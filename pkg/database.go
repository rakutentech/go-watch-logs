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
	if db != nil {
		// do a db ping to check if the connection is still alive
		if err := db.Ping(); err == nil {
			slog.Info("Reusing database connection", "dbName", dbName)
			return db, nil
		} else {
			slog.Warn("Closing database connection", "error", err.Error())
			db.Close()
		}
	}

	slog.Info("Initializing database", "dbName", dbName)
	var err error
	db, err = sql.Open("sqlite", dbName)
	if err != nil {
		slog.Error("Error opening database", "error", err.Error())
		return nil, err
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
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
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
