package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"oauth2-proxy/pkg/oauth2/store"
)

type Store struct {
	db *sql.DB
}

func New(config *Config) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.DBPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Remove existing database for clean setup
	if _, err := os.Stat(config.DBPath); err == nil {
		if err := os.Remove(config.DBPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", config.DBPath+"?_foreign_keys=ON&_journal=WAL&_cache_size=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Run migrations
	if err := store.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// NewOAuth2Store creates an OAuth2 store backed by SQLite
func (s *Store) NewOAuth2Store() store.Store {
	return store.NewSQLiteStore(s.db)
}
