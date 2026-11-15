package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSchemaIsIdempotentDDL(t *testing.T) {
	for _, table := range []string{"projects", "deliveries", "incidents", "error_groups", "error_events",
		"workspaces", "users", "workspace_members", "sessions"} {
		if !strings.Contains(schemaSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("missing idempotent DDL for %s", table)
		}
	}
	// The Stage 2 migration must keep existing Stage 1 rows, so the one legal
	// drop is the old global unique slug constraint (replaced by a per
	// workspace unique index). Everything else must stay untouched.
	for _, line := range strings.Split(schemaSQL, "\n") {
		if strings.Contains(line, "DROP ") && !strings.Contains(line, "DROP CONSTRAINT IF EXISTS projects_slug_key") {
			t.Fatalf("schema must not drop anything beyond the slug constraint: %s", line)
		}
	}
}

// TestMigrateIntegration runs against a real Postgres when DATABASE_URL is
// set, and skips otherwise so go test ./... stays green everywhere.
func TestMigrateIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping Postgres integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate twice must be idempotent: %v", err)
	}
}
