package pkg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite" // nolint: revive
)

var (
	db *sql.DB
)

func InitDB(dbName string) (*sql.DB, error) {
	// return db if already initialized
	if db != nil {
		err := db.Ping()
		if err == nil {
			slog.Info("Reusing database connection", "dbName", dbName)
			if err := vaccumIfOver(db, dbName, 100); err != nil {
				slog.Error("Error vacuuming database", "error", err.Error())
				return nil, err
			}
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
func DeleteDB(dbName string) error {
	if _, err := os.Stat(dbName); err == nil {
		if err := os.Remove(dbName); err != nil {
			return err
		}
	}
	return nil
}

func vaccumIfOver(db *sql.DB, dbName string, mb int64) error {
	// check if dbName file size is over 100MB
	fileInfo, err := os.Stat(dbName)
	if err != nil {
		return err
	}
	slog.Info("DB", "size", humanReadableSize(fileInfo.Size()))
	if fileInfo.Size() < mb*1024*1024 {
		return nil
	}
	slog.Info("Vacuuming database")
	_, err = db.Exec("VACUUM")
	if err != nil {
		return err
	}
	return nil
}

func humanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
