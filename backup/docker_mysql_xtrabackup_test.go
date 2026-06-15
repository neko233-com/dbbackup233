//go:build docker_xtrabackup

package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDockerMySQLXtraBackupFullIncrementalAndWrites(t *testing.T) {
	requireCommand(t, "docker")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	rows := envIntXtra(t, "DBBACKUP233_XTRA_ROWS", 100000)
	payloadBytes := envIntXtra(t, "DBBACKUP233_XTRA_PAYLOAD_BYTES", 256)
	port := dockerMySQLPort(t)
	containerName := fmt.Sprintf("dbbackup233-mysql-xtra-%d", time.Now().UnixNano())
	dataVolume := containerName + "-data"
	if port > 0 {
		assertPortAvailable(t, port)
	}
	port = startDockerMySQLWithVolume(t, ctx, containerName, dataVolume, port)
	defer removeDockerContainer(t, containerName)
	defer removeDockerVolume(t, dataVolume)
	waitForMySQL(t, ctx, port)

	rootDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "")
	defer rootDB.Close()
	execMySQL(t, rootDB, "CREATE DATABASE game_src CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci")
	srcDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "game_src")
	defer srcDB.Close()
	createBenchmarkSchema(t, srcDB)
	insertBenchmarkRows(t, srcDB, rows, payloadBytes)

	root := t.TempDir()
	xtraDir := filepath.Join(root, "xtrabackup")
	cfg := xtraBackupDockerConfig(root, xtraDir, containerName, dataVolume, "full", "")
	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Backup(ctx); err != nil {
		t.Fatal(err)
	}

	var writes atomic.Int64
	var writeErr atomic.Value
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		payload := makePayload(64)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, err := srcDB.Exec(`INSERT INTO bench_events(player_id, event_type, gold, payload, note, created_at)
				VALUES (?, 'xtra-live', ?, JSON_OBJECT('payload', ?), ?, NOW(6))`, writes.Load()%10000, writes.Load(), payload, payload)
			if err != nil {
				writeErr.Store(err)
				return
			}
			writes.Add(1)
		}
	}()

	incCfg := xtraBackupDockerConfig(root, xtraDir, containerName, dataVolume, "incremental", "")
	if err := NewBackupManager(incCfg, RunnerOptions{Logf: t.Logf}).Backup(ctx); err != nil {
		close(stop)
		<-done
		t.Fatal(err)
	}
	close(stop)
	<-done
	if value := writeErr.Load(); value != nil {
		t.Fatalf("concurrent write failed during xtrabackup: %v", value)
	}
	if writes.Load() == 0 {
		t.Fatal("expected concurrent writes during xtrabackup")
	}

	arts, err := ListArtifacts(filepath.Join(root, "backups", "manifest.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("expected full + incremental artifacts, got %d", len(arts))
	}
	var fullArt, incArt BackupArtifact
	for _, art := range arts {
		if !strings.HasSuffix(art.FilePath, ".xb.zip") {
			t.Fatalf("expected xtrabackup artifact, got %s", art.FilePath)
		}
		if art.Size == 0 || art.SHA256 == "" {
			t.Fatalf("bad artifact metadata: %+v", art)
		}
		if strings.Contains(art.JobName, "full") {
			fullArt = art
		}
		if strings.Contains(art.JobName, "incremental") {
			incArt = art
		}
	}
	latest, err := readXtraLatestBase(xtraDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(latest, "xtrabackup_checkpoints")); err != nil {
		t.Fatalf("latest base missing checkpoints: %v", err)
	}
	t.Logf("XTRABACKUP rows=%d payload_bytes=%d concurrent_writes=%d full=%s inc=%s latest_base=%s",
		rows, payloadBytes, writes.Load(), fullArt.FilePath, incArt.FilePath, latest)
}

func xtraBackupDockerConfig(root, xtraDir, mysqlContainer, dataVolume, mode, baseDir string) Config {
	job := strings.TrimPrefix(mode, "xtrabackup-")
	return Config{
		Defaults: Defaults{
			Compress:        "gzip",
			TimestampFormat: "20060102-150405",
			Concurrency:     1,
			WorkDir:         filepath.Join(root, "backups"),
			ManifestPath:    filepath.Join(root, "backups", "manifest.jsonl"),
		},
		Sources: []SourceConfig{{
			Name: "mysql-game-" + job,
			Type: "mysql",
			MySQL: MySQLConfig{
				Host:               "127.0.0.1",
				Port:               3306,
				User:               "root",
				Password:           "root",
				Database:           "game_src",
				Version:            "mysql80",
				Mode:               mode,
				XtraBackupTool:     "docker",
				XtraBackupDir:      xtraDir,
				XtraCommandDir:     "/xtrabackup",
				IncrementalBaseDir: baseDir,
				XtraExtraArgs: []string{
					"run", "--rm",
					"--user", "0:0",
					"--network", "container:" + mysqlContainer,
					"-e", "MYSQL_PWD=root",
					"-v", dataVolume + ":/var/lib/mysql",
					"-v", xtraDir + ":/xtrabackup",
					"percona/percona-xtrabackup:8.0",
					"xtrabackup",
				},
			},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}},
		Jobs:    []JobConfig{{Name: "mysql-game-" + job, Source: "mysql-game-" + job, Targets: []string{"local"}}},
	}
}

func startDockerMySQLWithVolume(t *testing.T, ctx context.Context, containerName, volumeName string, port int) int {
	t.Helper()
	removeDockerContainer(t, containerName)
	removeDockerVolume(t, volumeName)
	out, err := exec.CommandContext(ctx, "docker", "volume", "create", volumeName).CombinedOutput()
	if err != nil {
		t.Fatalf("docker volume create failed: %v\n%s", err, out)
	}
	publish := "127.0.0.1::3306"
	if port > 0 {
		publish = fmt.Sprintf("%d:3306", port)
	}
	args := []string{
		"run", "-d", "--name", containerName,
		"-e", "MYSQL_ROOT_PASSWORD=root",
		"-p", publish,
		"-v", volumeName + ":/var/lib/mysql",
		"mysql:8.0",
	}
	out, err = exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run failed: %v\n%s", err, out)
	}
	if port > 0 {
		return port
	}
	out, err = exec.CommandContext(ctx, "docker", "port", containerName, "3306/tcp").CombinedOutput()
	if err != nil {
		t.Fatalf("docker port failed: %v\n%s", err, out)
	}
	text := strings.TrimSpace(string(out))
	_, portText, ok := strings.Cut(text, ":")
	if !ok {
		t.Fatalf("unexpected docker port output: %q", text)
	}
	parsed, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("unexpected docker port output: %q", text)
	}
	return parsed
}

func removeDockerVolume(t *testing.T, name string) {
	t.Helper()
	_ = exec.Command("docker", "volume", "rm", "-f", name).Run()
}

func envIntXtra(t *testing.T, key string, fallback int) int {
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
