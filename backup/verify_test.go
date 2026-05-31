package backup

import (
	"path/filepath"
	"testing"
)

func TestVerifyConfigWarnsMissingTools(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Defaults: Defaults{WorkDir: filepath.Join(root, "work"), ManifestPath: filepath.Join(root, "work", "manifest.jsonl")},
		Sources:  []SourceConfig{{Name: "game", Type: "mysql", MySQL: MySQLConfig{Host: "127.0.0.1", User: "root", Database: "game", DumpTool: "missing-dbbackup233-tool", RestoreTool: "missing-dbbackup233-restore"}}},
		Targets:  []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}},
		Jobs:     []JobConfig{{Name: "game", Source: "game", Targets: []string{"local"}}},
	}
	result := VerifyConfig(cfg)
	if !result.OK {
		t.Fatalf("verify should keep missing tools as warnings, got errors: %v", result.Errors)
	}
	if len(result.Warnings) < 2 {
		t.Fatalf("expected missing tool warnings, got %v", result.Warnings)
	}
}
