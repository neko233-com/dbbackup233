package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func execSQLFile(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(raw)); err != nil {
		t.Fatal(err)
	}
}

func filepathFromRoot(parts ...string) string {
	all := append([]string{".."}, parts...)
	return filepath.Join(all...)
}
