package db

import (
	"path/filepath"
	"testing"
	"database/sql"
	"context"
)

func TestOpenAppliesMigrationsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repeat.db")
	first, err := Open(path)
	if err != nil { t.Fatal(err) }
	if err := first.Close(); err != nil { t.Fatal(err) }

	second, err := Open(path)
	if err != nil { t.Fatalf("reopen database: %v", err) }
	defer second.Close()

	var count int
	if err := second.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 { t.Fatalf("applied migrations=%d, want 2", count) }
	var columnCount int
	if err := second.QueryRow("SELECT COUNT(*) FROM pragma_table_info('questions') WHERE name='external_key'").Scan(&columnCount); err != nil {
		t.Fatal(err)
	}
	if columnCount != 1 { t.Fatalf("external_key column count=%d, want 1", columnCount) }
}

func TestPragmasApplyToEveryPhysicalConnection(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil { t.Fatal(err) }
	defer database.Close()

	first, err := database.Conn(context.Background())
	if err != nil { t.Fatal(err) }
	defer first.Close()
	second, err := database.Conn(t.Context())
	if err != nil { t.Fatal(err) }
	defer second.Close()

	for index, conn := range []*sql.Conn{first, second} {
		var foreignKeys, busyTimeout, synchronous int
		var journalMode string
		if err := conn.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil { t.Fatal(err) }
		if err := conn.QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil { t.Fatal(err) }
		if err := conn.QueryRowContext(t.Context(), "PRAGMA synchronous").Scan(&synchronous); err != nil { t.Fatal(err) }
		if err := conn.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&journalMode); err != nil { t.Fatal(err) }
		if foreignKeys != 1 || busyTimeout != 5000 || synchronous != 1 || journalMode != "wal" {
			t.Fatalf("connection %d pragmas: foreign_keys=%d busy_timeout=%d synchronous=%d journal_mode=%s", index, foreignKeys, busyTimeout, synchronous, journalMode)
		}
	}
}
