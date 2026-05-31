package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneManifestKeepsLatestPerJob(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.jsonl")
	for i := 0; i < 3; i++ {
		file := filepath.Join(root, "a-"+string(rune('0'+i))+".gz")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		art := BackupArtifact{
			Version:   "v" + string(rune('0'+i)),
			JobName:   "job",
			FilePath:  file,
			CreatedAt: time.Date(2026, 5, 31, 10+i, 0, 0, 0, time.UTC),
		}
		if err := AppendManifest(manifest, art); err != nil {
			t.Fatal(err)
		}
	}
	result, err := PruneManifest(manifest, RetentionConfig{KeepLast: 1}, PruneOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Kept) != 1 || result.Kept[0].Version != "v2" {
		t.Fatalf("unexpected kept: %+v", result.Kept)
	}
	if len(result.Deleted) != 2 {
		t.Fatalf("unexpected deleted: %+v", result.Deleted)
	}
	arts, err := ListArtifacts(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 || arts[0].Version != "v2" {
		t.Fatalf("manifest not pruned: %+v", arts)
	}
}
