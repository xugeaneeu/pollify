package postgrestest

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	platformpg "xugeaneeu/pollify/internal/platform/postgres"
)

func OpenTestDB(t testing.TB) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	schemaName := testSchemaName(t.Name())
	pkgPool, err := openPool(context.Background(), databaseURL, schemaName)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pkgPool.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+schemaName+` CASCADE`); err != nil {
			t.Fatalf("drop schema %s: %v", schemaName, err)
		}
		pkgPool.Close()
	})

	ensureSchema(t, pkgPool, schemaName)
	ensureExtensions(t, pkgPool)
	applyMigrations(t, pkgPool)

	return pkgPool
}

func openPool(ctx context.Context, databaseURL string, schemaName string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func ensureSchema(t testing.TB, pool *pgxpool.Pool, schemaName string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `CREATE SCHEMA IF NOT EXISTS `+schemaName)
	if err != nil {
		t.Fatalf("create schema %s: %v", schemaName, err)
	}
	_, err = pool.Exec(context.Background(), `SET search_path TO `+schemaName+`,public`)
	if err != nil {
		t.Fatalf("set search_path for %s: %v", schemaName, err)
	}
}

func ensureExtensions(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()

	const lockKey int64 = 1776941384248
	if _, err := pool.Exec(context.Background(), `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		t.Fatalf("lock extension setup: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey); err != nil {
			t.Fatalf("unlock extension setup: %v", err)
		}
	}()

	if _, err := pool.Exec(context.Background(), `CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public`); err != nil {
		t.Fatalf("ensure pgcrypto extension: %v", err)
	}
}

func applyMigrations(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()

	_, currentFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	if err := platformpg.ApplyMigrations(context.Background(), pool, filepath.Join(root, "migrations")); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
}

func testSchemaName(name string) string {
	var builder strings.Builder
	builder.WriteString("test_")
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	builder.WriteByte('_')
	builder.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	return builder.String()
}
