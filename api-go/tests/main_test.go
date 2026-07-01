package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/dionisvl/avi/api-go/internal/migrations"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// Get test database DSN from environment or use default (for Docker)
	testDSN := os.Getenv("TEST_DB_DSN")
	if testDSN == "" {
		// Default: running inside Docker container
		testDSN = "postgres://avi:avi@db:5432/avi_test?sslmode=disable"
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// 1. Create test database if not exists
	if err := ensureTestDB(ctx, testDSN, logger); err != nil {
		logger.Error("failed to ensure test db", "error", err)
		os.Exit(1)
	}

	// 2. Wait for database to be ready (retry with backoff)
	if err := waitForDatabase(ctx, testDSN, logger); err != nil {
		logger.Error("database not ready", "error", err)
		os.Exit(1)
	}

	// 3. Connect to test database
	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		logger.Error("failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	testPool = pool

	// 4. Run migrations (00001_init seeds categories/cities/items inline)
	if err := runMigrations(ctx, testDSN, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	logger.Info("test database ready", "dsn", testDSN)

	// 5. Run tests
	code := m.Run()

	os.Exit(code)
}

// waitForDatabase waits for the database to become available with exponential backoff
func waitForDatabase(ctx context.Context, testDSN string, logger *slog.Logger) error {
	backoff := 100 * time.Millisecond
	maxRetries := 10

	for i := range maxRetries {
		cfg, err := pgxpool.ParseConfig(testDSN)
		if err != nil {
			return fmt.Errorf("parse dsn: %w", err)
		}

		conn, err := pgx.ConnectConfig(ctx, cfg.ConnConfig)
		if err == nil {
			_ = conn.Close(ctx) //nolint:errcheck
			logger.Info("database is ready")
			return nil
		}

		if i < maxRetries-1 {
			logger.Debug("waiting for database", "attempt", i+1, "error", err)
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return fmt.Errorf("database not ready after %d retries", maxRetries)
}

// ensureTestDB creates avi_test database if it doesn't exist.
// It connects to the default 'postgres' database to run CREATE DATABASE command.
func ensureTestDB(ctx context.Context, testDSN string, logger *slog.Logger) error {
	// Parse the test DSN to get connection parameters
	// Expected format: postgres://user:password@host:port/dbname?sslmode=disable
	// We need to extract host, port, user, password and connect to 'postgres' database

	// Use pgxpool to parse the config, then override the database name
	cfg, err := pgxpool.ParseConfig(testDSN)
	if err != nil {
		return fmt.Errorf("parse test dsn: %w", err)
	}

	// Get original db name
	testDBName := cfg.ConnConfig.Database

	// Connect to 'postgres' database instead
	cfg.ConnConfig.Database = "postgres"
	adminPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to postgres db: %w", err)
	}
	defer adminPool.Close()

	// Check if database exists first
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s');", testDBName)
	err = adminPool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database existence: %w", err)
	}

	if !exists {
		// Create test database (use TEMPLATE=template0 to avoid collation issues in Docker)
		createQuery := fmt.Sprintf("CREATE DATABASE %s TEMPLATE=template0;", testDBName)
		_, err = adminPool.Exec(ctx, createQuery)
		if err != nil {
			return fmt.Errorf("create test database: %w", err)
		}
		logger.Info("test database created", "db", testDBName)
	} else {
		logger.Info("test database already exists", "db", testDBName)
	}

	// Try to connect to verify it's ready
	cfg.ConnConfig.Database = testDBName
	testPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("verify database connection: %w", err)
	}
	defer testPool.Close()

	logger.Info("database verified ready", "db", testDBName)
	return nil
}

// runMigrations runs all pending migrations using goose
func runMigrations(ctx context.Context, testDSN string, logger *slog.Logger) error {
	_ = pgx.Connect // Suppress unused import warning for pgx
	_ = stdlib.GetDefaultDriver

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	// Use sql.Open with pgx driver registered by stdlib
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Recreate the public schema on every run so tests always start from a fresh DB
	// even when older migrations were edited during active development.
	if _, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		return fmt.Errorf("reset public schema: %w", err)
	}

	// Run migrations
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}
