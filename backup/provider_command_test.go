package backup

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandProviderBackupRestore(t *testing.T) {
	root := t.TempDir()
	restore := filepath.Join(root, "restore.txt")
	source := SourceConfig{
		Name: "any",
		Type: "command",
		Command: CommandConfig{
			Extension:      "txt",
			BackupCommand:  commandShell("printf game-data"),
			RestoreCommand: commandRestoreShell(restore),
			CaptureStdout:  true,
		},
	}
	provider := CommandBackupProvider{}
	artifact := filepath.Join(root, "artifact."+provider.Extension(source, "gzip"))
	if err := provider.Backup(Context{Logf: t.Logf}, source, artifact); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "game-data" {
		t.Fatalf("artifact content = %q", raw)
	}
	if err := provider.Restore(Context{Logf: t.Logf}, source, artifact); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(restore)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "game-data" {
		t.Fatalf("restore content = %q", restored)
	}
}

func commandShell(script string) []string {
	if runtime.GOOS == "windows" {
		if script == "printf game-data" {
			script = "[Console]::Out.Write('game-data')"
		}
		return []string{"powershell.exe", "-NoProfile", "-Command", script}
	}
	return []string{"sh", "-c", script}
}

func commandRestoreShell(path string) []string {
	if runtime.GOOS == "windows" {
		return []string{"powershell.exe", "-NoProfile", "-Command", "Copy-Item -LiteralPath $env:ARTIFACT_PATH -Destination " + quotePS(path) + " -Force"}
	}
	return []string{"sh", "-c", "cat \"$ARTIFACT_PATH\" > '" + path + "'"}
}
