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
		err := db.Ping()
		if err == nil {
			slog.Info("Reusing database connection", "dbName", dbName)
			return db, nil
		}
		slog.Warn("Closing database connection", "error", err.Error())
		db.Close()
	}

	slog.Info("Initializing database", "dbName", dbName)
	var err error
	db, err = sql.Open("sqlite", dbName)
	if err != nil {
		slog.Error("Error opening database", "error", err.Error())
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := createTables(db); err != nil {
		return nil, err
	}

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

func createTables(db *sql.DB) error {
	slog.Info("Creating tables if not exist")
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value INTEGER,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		slog.Error("Error creating state table", "error", err.Error())
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS anomaly (
			key TEXT PRIMARY KEY,
			value INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		slog.Error("Error creating plot table", "error", err.Error())
		return err
	}

	return nil
}
