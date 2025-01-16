package pkg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite" // nolint: revive
)

func InitDB(dbName string) (*sql.DB, error) {
	slog.Info("Initializing database", "dbName", dbName)
	var err error
	db, err := sql.Open("sqlite", dbName)
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

func GetUniqDBName(f Flags) string {
	suffix := Hash(fmt.Sprintf("%s-%s-%s-%s-%d", f.FilePath, f.Match, f.Ignore, f.MSTeamsHook, f.Every)) + ".sqlite"
	dbName := f.DBPath + "." + suffix
	return dbName
}

func DeleteDB(dbName string) error {
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
			updated_at DATETIME
		)
	`)
	if err != nil {
		slog.Error("Error creating state table", "error", err.Error())
		return err
	}

	return nil
}
