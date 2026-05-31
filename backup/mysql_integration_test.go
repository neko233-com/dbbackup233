//go:build mysql_integration

package backup

import (
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLocalMySQL80Backup(t *testing.T) {
	if _, err := exec.LookPath(DefaultDumpToolName("mysqldump")); err != nil {
		t.Fatalf("mysqldump is required for this test; run `dbbackup233 install mysqldump`: %v", err)
	}

	host := envOr("DBBACKUP233_MYSQL_HOST", "127.0.0.1")
	port := envOr("DBBACKUP233_MYSQL_PORT", "3306")
	user := envOr("DBBACKUP233_MYSQL_USER", "root")
	password := envOr("DBBACKUP233_MYSQL_PASSWORD", "root")
	database := envOr("DBBACKUP233_MYSQL_DATABASE", "dbbackup233_it")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	adminDB := openMySQLForTest(t, host, port, user, password, "")
	defer adminDB.Close()

	execMySQL(t, adminDB, "DROP DATABASE IF EXISTS `"+database+"`")
	execMySQL(t, adminDB, "CREATE DATABASE `"+database+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci")
	defer execMySQL(t, adminDB, "DROP DATABASE IF EXISTS `"+database+"`")

	gameDB := openMySQLForTest(t, host, port, user, password, database)
	defer gameDB.Close()
	execMySQL(t, gameDB, `CREATE TABLE players (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  level INT NOT NULL,
  payload JSON NULL,
  avatar BLOB NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB`)
	execMySQL(t, gameDB, `INSERT INTO players(name, level, payload, avatar) VALUES
  ('alice', 10, JSON_OBJECT('gold', 100, 'server', 's1'), X'CAFE'),
  ('bob', 20, JSON_OBJECT('gold', 250, 'server', 's1'), X'BABE')`)

	backupDir := t.TempDir()
	cfg := Config{
		Defaults: Defaults{
			Compress:        "gzip",
			TimestampFormat: "20060102-150405",
			Concurrency:     1,
			WorkDir:         backupDir,
		},
		Sources: []SourceConfig{{
			Name: "mysql80-local",
			Type: "mysql",
			MySQL: MySQLConfig{
				Host:              host,
				Port:              3306,
				User:              user,
				Password:          password,
				Database:          database,
				Version:           "8.0",
				DumpTool:          DefaultDumpToolName("mysqldump"),
				RestoreTool:       DefaultDumpToolName("mysql"),
				SingleTransaction: true,
				Quick:             true,
				Routines:          true,
				Triggers:          true,
				Events:            true,
				SetGTIDPurged:     "OFF",
			},
		}},
		Targets: []TargetConfig{{
			Name:  "local",
			Type:  "local",
			Local: LocalTarget{Path: backupDir},
		}},
		Jobs: []JobConfig{{
			Name:    "mysql80-local",
			Source:  "mysql80-local",
			Targets: []string{"local"},
		}},
	}
	if port != "3306" {
		cfg.Sources[0].MySQL.Port = mustAtoi(t, port)
	}

	if err := RunConfig(ctx, cfg, RunnerOptions{Logf: t.Logf}); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(backupDir, "mysql", "*.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one gzip backup, got %d", len(files))
	}
	dump := readGzipText(t, files[0])
	for _, want := range []string{"CREATE TABLE `players`", "INSERT INTO `players`", "alice"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("dump missing %q:\n%s", want, dump)
		}
	}
}

func readGzipText(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
