//go:build docker_integration

package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestDockerPostgresBackupRestoreAndHistory(t *testing.T) {
	requireCommand(t, "docker")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	port := 25432
	containerName := fmt.Sprintf("dbbackup233-postgres-it-%d", port)
	removeDockerContainer(t, containerName)
	startDockerPostgres(t, ctx, containerName, port)
	defer removeDockerContainer(t, containerName)
	waitForPostgres(t, ctx, port)

	admin := openPostgresForTest(t, port, "postgres")
	defer admin.Close()
	execPostgres(t, admin, "CREATE DATABASE game_src")
	execPostgres(t, admin, "CREATE DATABASE game_restore")

	src := openPostgresForTest(t, port, "game_src")
	defer src.Close()
	execSQLFile(t, src, filepathFromRoot("tests/sql/postgres_schema.sql"))
	execSQLFile(t, src, filepathFromRoot("tests/sql/postgres_seed.sql"))

	root := t.TempDir()
	cfg := Config{
		Defaults: Defaults{Compress: "gzip", TimestampFormat: "20060102-150405", Concurrency: 1, WorkDir: root, ManifestPath: root + "/manifest.jsonl"},
		Sources: []SourceConfig{{
			Name: "pg", Type: "postgres",
			Postgres: PostgresConfig{
				Host: "127.0.0.1", Port: 5432, User: "postgres", Password: "postgres", Database: "game_src",
				RestoreDatabase: "game_restore",
				DumpTool:        "docker", RestoreTool: "docker", Format: "plain",
				ExtraArgs:        []string{"exec", "-i", containerName, "env", "PGPASSWORD=postgres", "pg_dump"},
				RestoreExtraArgs: []string{"exec", "-i", containerName, "env", "PGPASSWORD=postgres", "psql"},
			},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: root + "/local"}}},
		Jobs:    []JobConfig{{Name: "pg", Source: "pg", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	cfg.Sources[0].Postgres.Port = 5432
	cfg.Sources[0].Postgres.Database = "game_src"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Backup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Restore(ctx, "pg", ""); err != nil {
		t.Fatal(err)
	}
	restore := openPostgresForTest(t, port, "game_restore")
	defer restore.Close()
	var count int
	if err := restore.QueryRow("SELECT COUNT(*) FROM players").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 players, got %d", count)
	}
}

func startDockerPostgres(t *testing.T, ctx context.Context, containerName string, port int) {
	t.Helper()
	args := []string{"run", "-d", "--name", containerName, "-e", "POSTGRES_PASSWORD=postgres", "-p", fmt.Sprintf("%d:5432", port), "postgres:16-alpine"}
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run postgres failed: %v\n%s", err, out)
	}
}

func waitForPostgres(t *testing.T, ctx context.Context, port int) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		db := openPostgresNoFatal(port, "postgres")
		if db != nil {
			if err := db.PingContext(ctx); err == nil {
				_ = db.Close()
				return
			}
			_ = db.Close()
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("postgres container did not become ready")
}

func openPostgresForTest(t *testing.T, port int, database string) *sql.DB {
	t.Helper()
	db := openPostgresNoFatal(port, database)
	if db == nil {
		t.Fatal("open postgres failed")
	}
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	return db
}

func openPostgresNoFatal(port int, database string) *sql.DB {
	db, err := sql.Open("postgres", fmt.Sprintf("host=127.0.0.1 port=%d user=postgres password=postgres dbname=%s sslmode=disable", port, database))
	if err != nil {
		return nil
	}
	return db
}

func execPostgres(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatal(err)
	}
}
