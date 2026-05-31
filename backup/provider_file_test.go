package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileBackupRestoreExtractsContent(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	restore := filepath.Join(root, "restore")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("server_id=s1\nunicode=中文\n")
	if err := os.WriteFile(filepath.Join(src, "config.txt"), want, 0o644); err != nil {
		t.Fatal(err)
	}

	source := SourceConfig{Name: "files", Type: "file", File: FileConfig{Paths: []string{src}, RestoreDir: restore}}
	cfg := Config{Defaults: Defaults{Compress: "gzip"}, Sources: []SourceConfig{source}, Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}}, Jobs: []JobConfig{{Name: "files", Source: "files", Targets: []string{"local"}}}}
	cfg.ApplyDefaults()
	source = cfg.Sources[0]
	provider := FileBackupProvider{}
	artifact := filepath.Join(root, "files."+provider.Extension(source, cfg.Defaults.Compress))
	if err := provider.Backup(Context{Logf: t.Logf}, source, artifact); err != nil {
		t.Fatal(err)
	}
	if err := provider.Restore(Context{Logf: t.Logf}, source, artifact); err != nil {
		t.Fatal(err)
	}

	got, err := findRestoredFile(restore, "config.txt")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(want) {
		t.Fatalf("restored content mismatch: %q", raw)
	}
}

func findRestoredFile(root, name string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == name {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}
