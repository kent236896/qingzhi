package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestBaselineCreatesNoTables(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one migration file")
	}
	for _, entry := range entries {
		content, err := fs.ReadFile(FS, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		upper := strings.ToUpper(string(content))
		if strings.Contains(upper, "CREATE TABLE") {
			t.Fatalf("%s creates a table, phase 1 must not", entry.Name())
		}
	}
}
