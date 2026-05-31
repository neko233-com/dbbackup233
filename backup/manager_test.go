package backup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSelectJobs(t *testing.T) {
	jobs := []JobConfig{
		{Name: "mysql-game"},
		{Name: "files"},
		{Name: "pg"},
	}
	got, err := selectJobs(jobs, []string{"files", "mysql-game"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "mysql-game" || got[1].Name != "files" {
		t.Fatalf("unexpected selected jobs: %#v", got)
	}
}

func TestSelectJobsRejectsUnknown(t *testing.T) {
	_, err := selectJobs([]JobConfig{{Name: "mysql-game"}}, []string{"missing"})
	if err == nil || !strings.Contains(err.Error(), "unknown backup job") {
		t.Fatalf("expected unknown job error, got %v", err)
	}
}

func TestRestoreRejectsChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "files.tar.gz")
	if err := os.WriteFile(artifact, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := SHA256File(artifact)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "manifest.jsonl")
	if err := AppendManifest(manifest, BackupArtifact{
		Version:    "20260531-120000",
		JobName:    "files",
		SourceName: "files",
		SourceType: "file",
		FilePath:   artifact,
		SHA256:     sum,
		CreatedAt:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("corrupted"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Defaults: Defaults{ManifestPath: manifest, WorkDir: root, Compress: "gzip", Concurrency: 1},
		Sources:  []SourceConfig{{Name: "files", Type: "file", File: FileConfig{RestoreDir: filepath.Join(root, "restore")}}},
		Targets:  []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}},
		Jobs:     []JobConfig{{Name: "files", Source: "files", Targets: []string{"local"}}},
	}
	err = NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Restore(context.Background(), "files", "")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}
