package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat konfigurasi database: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("gagal ping database: %w", err)
	}

	log.Println("Database connection pool berhasil dibuat")
	return pool, nil
}

func RunMigrations(connString string, migrationsPath string) error {
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(sourceURL, connString)
	if err != nil {
		return fmt.Errorf("gagal menginisialisasi database migration: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("gagal menjalankan database migration: %w", err)
	}

	log.Println("Database migrations berhasil dijalankan")
	return nil
}
