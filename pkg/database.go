package pkg

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite" // nolint: revive
)

func InitDB(dbName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value INTEGER
		)
	`)
	if err != nil {
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
