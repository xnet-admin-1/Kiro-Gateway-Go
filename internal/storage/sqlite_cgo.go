// +build !nocgo

package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteStoreWithCGO creates a new SQLite store with CGO support
func NewSQLiteStoreWithCGO(db *sql.DB) (*SQLiteStore, error) {
	return NewSQLiteStore(db)
}
