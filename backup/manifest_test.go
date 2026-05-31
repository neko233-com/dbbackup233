package backup

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManifestFindLatestAndVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.jsonl")
	oldArt := BackupArtifact{Version: "20260531-100000", JobName: "mysql-game", SourceName: "mysql-game", SourceType: "mysql", FilePath: "old.sql.gz", CreatedAt: time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)}
	newArt := BackupArtifact{Version: "20260531-110000", JobName: "mysql-game", SourceName: "mysql-game", SourceType: "mysql", FilePath: "new.sql.gz", CreatedAt: time.Date(2026, 5, 31, 11, 0, 0, 0, time.UTC)}
	if err := AppendManifest(path, oldArt); err != nil {
		t.Fatal(err)
	}
	if err := AppendManifest(path, newArt); err != nil {
		t.Fatal(err)
	}

	latest, err := FindArtifact(path, "mysql-game", "")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != newArt.Version {
		t.Fatalf("latest got %s", latest.Version)
	}

	selected, err := FindArtifact(path, "mysql-game", oldArt.Version)
	if err != nil {
		t.Fatal(err)
	}
	if selected.FilePath != oldArt.FilePath {
		t.Fatalf("selected got %s", selected.FilePath)
	}
}
