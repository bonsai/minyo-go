package db

import (
	"path/filepath"
	"testing"
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
