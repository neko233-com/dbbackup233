package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMySQLXtraBackupFullCommand(t *testing.T) {
	cfg := MySQLConfig{
		Host: "127.0.0.1", Port: 3306, User: "backup", Password: "secret", Database: "game",
		Mode: "xtrabackup-full", XtraBackupTool: "xtrabackup",
	}

	spec := BuildMySQLXtraBackupCommand(cfg, "/backup/full")
	got := strings.Join(spec.Args, " ")
	for _, want := range []string{"--backup", "--host=127.0.0.1", "--port=3306", "--user=backup", "--databases=game", "--target-dir=/backup/full"} {
		if !strings.Contains(got, want) {
			t.Fatalf("args %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "--incremental-basedir=") {
		t.Fatalf("full backup should not include incremental base: %q", got)
	}
}

func TestMySQLXtraBackupIncrementalCommand(t *testing.T) {
	cfg := MySQLConfig{
		Host: "127.0.0.1", Port: 3306, User: "backup", Database: "game",
		Mode: "xtrabackup-incremental", XtraBackupTool: "xtrabackup", IncrementalBaseDir: "/backup/full",
	}

	spec := BuildMySQLXtraBackupCommand(cfg, "/backup/inc1")
	got := strings.Join(spec.Args, " ")
	if !strings.Contains(got, "--incremental-basedir=/backup/full") {
		t.Fatalf("args %q missing incremental base", got)
	}
}

func TestMySQLXtraCopyBackCommand(t *testing.T) {
	cfg := MySQLConfig{XtraBackupTool: "xtrabackup", RestoreDatadir: "/var/lib/mysql"}
	spec := BuildMySQLXtraCopyBackCommand(cfg, "/restore/mysql")
	got := strings.Join(spec.Args, " ")
	for _, want := range []string{"--copy-back", "--target-dir=/restore/mysql", "--datadir=/var/lib/mysql"} {
		if !strings.Contains(got, want) {
			t.Fatalf("args %q missing %q", got, want)
		}
	}
}

func TestMySQLXtraBackupExtension(t *testing.T) {
	source := SourceConfig{Type: "mysql", MySQL: MySQLConfig{Mode: "full"}}
	if got := (MySQLBackupProvider{}).Extension(source, "gzip"); got != "xb.zip" {
		t.Fatalf("got %q", got)
	}
	source.MySQL.PhysicalFormat = "tar.gz"
	if got := (MySQLBackupProvider{}).Extension(source, "gzip"); got != "xb.tar.gz" {
		t.Fatalf("got %q", got)
	}
}

func TestXtraBackupLatestBase(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "mysql-game-full")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "xtrabackup_checkpoints"), []byte("backup_type = full-backuped\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeXtraLatestBase(dir, base); err != nil {
		t.Fatal(err)
	}
	got, err := readXtraLatestBase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != base {
		t.Fatalf("got %q, want %q", got, base)
	}
	if err := ValidateXtraBackupChain(MySQLConfig{Mode: "xtrabackup-incremental", XtraBackupDir: dir}); err != nil {
		t.Fatal(err)
	}
}
