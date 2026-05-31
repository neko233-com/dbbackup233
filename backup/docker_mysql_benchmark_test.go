//go:build docker_benchmark

package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestDockerMySQLBackupBenchmark(t *testing.T) {
	requireCommand(t, "docker")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	rows := envInt(t, "DBBACKUP233_BENCH_ROWS", 50000)
	payloadBytes := envInt(t, "DBBACKUP233_BENCH_PAYLOAD_BYTES", 256)
	port := dockerMySQLPort(t)
	containerName := fmt.Sprintf("dbbackup233-mysql-bench-%d", port)
	assertPortAvailable(t, port)
	startDockerMySQL(t, ctx, containerName, port)
	defer removeDockerContainer(t, containerName)
	waitForMySQL(t, ctx, port)

	rootDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "")
	defer rootDB.Close()
	execMySQL(t, rootDB, "CREATE DATABASE bench_src CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci")

	srcDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "bench_src")
	defer srcDB.Close()
	createBenchmarkSchema(t, srcDB)
	insertBenchmarkRows(t, srcDB, rows, payloadBytes)

	root := t.TempDir()
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", TimestampFormat: "20060102-150405", Concurrency: 1, WorkDir: filepath.Join(root, "backups"), ManifestPath: filepath.Join(root, "backups", "manifest.jsonl")},
		Sources: []SourceConfig{{
			Name: "bench", Type: "mysql",
			MySQL: MySQLConfig{
				Host: "127.0.0.1", Port: 3306, User: "root", Password: "root", Database: "bench_src", Version: "mysql80",
				DumpTool: "docker", RestoreTool: "docker", SingleTransaction: true, Quick: true, SetGTIDPurged: "OFF",
				ExtraArgs: []string{"exec", "-i", containerName, "env", "MYSQL_PWD=root", "mysqldump"},
			},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}},
		Jobs:    []JobConfig{{Name: "bench", Source: "bench", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Backup(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	arts, err := ListArtifacts(cfg.Defaults.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(arts))
	}
	mb := float64(arts[0].Size) / 1024 / 1024
	rowsPerSec := float64(rows) / elapsed.Seconds()
	mbPerSec := mb / elapsed.Seconds()
	t.Logf("BENCH rows=%d payload_bytes=%d artifact=%.2fMiB elapsed=%s rows_per_sec=%.0f compressed_mib_per_sec=%.2f sha256=%s",
		rows, payloadBytes, mb, elapsed.Round(time.Millisecond), rowsPerSec, mbPerSec, arts[0].SHA256)
}

func createBenchmarkSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	execMySQL(t, db, `CREATE TABLE bench_events (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  player_id BIGINT NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  gold BIGINT NOT NULL,
  payload JSON NOT NULL,
  note TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  KEY idx_player_created (player_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
}

func insertBenchmarkRows(t *testing.T, db *sql.DB, rows int, payloadBytes int) {
	t.Helper()
	payload := makePayload(payloadBytes)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO bench_events(player_id, event_type, gold, payload, note, created_at) VALUES (?, ?, ?, JSON_OBJECT('payload', ?), ?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	for i := 0; i < rows; i++ {
		if _, err := stmt.Exec(i%10000, "login", i*10, payload, payload, now.Add(time.Duration(i)*time.Millisecond)); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if i > 0 && i%5000 == 0 {
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			tx, err = db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			stmt, err = tx.Prepare(`INSERT INTO bench_events(player_id, event_type, gold, payload, note, created_at) VALUES (?, ?, ?, JSON_OBJECT('payload', ?), ?, ?)`)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func makePayload(size int) string {
	if size <= 0 {
		return ""
	}
	raw := make([]byte, size)
	for i := range raw {
		raw[i] = byte('a' + i%26)
	}
	return string(raw)
}

func envInt(t *testing.T, key string, fallback int) int {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
