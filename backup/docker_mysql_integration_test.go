//go:build docker_integration

package backup

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDockerMySQLBackupRestoreAndHistory(t *testing.T) {
	requireCommand(t, "docker")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	port := dockerMySQLPort(t)
	containerName := fmt.Sprintf("dbbackup233-mysql-it-%d", port)
	assertPortAvailable(t, port)
	startDockerMySQL(t, ctx, containerName, port)
	defer removeDockerContainer(t, containerName)
	waitForMySQL(t, ctx, port)

	rootDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "")
	defer rootDB.Close()
	execMySQL(t, rootDB, "CREATE DATABASE game_src CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci")
	execMySQL(t, rootDB, "CREATE DATABASE game_restore CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci")

	srcDB := openMySQLForTest(t, "127.0.0.1", fmt.Sprint(port), "root", "root", "game_src")
	defer srcDB.Close()
	execSQLFile(t, srcDB, filepathFromRoot("tests/sql/mysql_schema.sql"))
	execSQLFile(t, srcDB, filepathFromRoot("tests/sql/mysql_seed.sql"))

	workDir := t.TempDir()
	mock := newS3Mock(t)
	defer mock.Close()
	cfg := Config{
		Defaults: Defaults{
			Compress:        "gzip",
			TimestampFormat: "20060102-150405",
			Concurrency:     2,
			WorkDir:         filepath.Join(workDir, "backups"),
			ManifestPath:    filepath.Join(workDir, "backups", "manifest.jsonl"),
		},
		Sources: []SourceConfig{
			{
				Name: "mysql-game",
				Type: "mysql",
				MySQL: MySQLConfig{
					Host:              "127.0.0.1",
					Port:              3306,
					User:              "root",
					Password:          "root",
					Database:          "game_src",
					RestoreDatabase:   "game_restore",
					Version:           "mysql80",
					DumpTool:          "docker",
					RestoreTool:       "docker",
					SingleTransaction: true,
					Quick:             true,
					Routines:          true,
					Triggers:          true,
					Events:            true,
					SetGTIDPurged:     "OFF",
					NoCreateDB:        true,
					ExtraArgs:         []string{"exec", "-i", containerName, "env", "MYSQL_PWD=root", "mysqldump"},
					RestoreExtraArgs:  []string{"exec", "-i", containerName, "env", "MYSQL_PWD=root", "mysql"},
				},
			},
			{
				Name: "files",
				Type: "file",
				File: FileConfig{Paths: []string{writeFixtureDir(t, workDir)}, RestoreDir: filepath.Join(workDir, "restore-files")},
			},
		},
		Targets: []TargetConfig{
			{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(workDir, "local")}},
			{Name: "oss-mock", Type: "oss", S3: S3Target{Endpoint: strings.TrimPrefix(mock.URL, "http://"), Bucket: "bucket", AccessKey: "ak", SecretKey: "sk", Prefix: "dbbackup233", UseSSL: false, ForcePathStyle: true}},
		},
		Jobs: []JobConfig{
			{Name: "mysql-game", Source: "mysql-game", Targets: []string{"local", "oss-mock"}},
			{Name: "files", Source: "files", Targets: []string{"local"}},
		},
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Backup(ctx); err != nil {
		t.Fatal(err)
	}

	arts, err := ListArtifacts(cfg.Defaults.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("expected 2 manifest artifacts, got %d", len(arts))
	}

	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Restore(ctx, "mysql-game", ""); err != nil {
		t.Fatal(err)
	}

	assertMySQLCountsEqual(t, rootDB)
	assertMySQLDumpReadable(t, filepath.Join(cfg.Defaults.WorkDir, "mysql"))
	if mock.PutCount() == 0 {
		t.Fatal("expected mock oss/s3 upload to be called")
	}
	mock.AssertUploadedGzipContains(t, "CREATE TABLE `players`")
}

func startDockerMySQL(t *testing.T, ctx context.Context, containerName string, port int) {
	t.Helper()
	removeDockerContainer(t, containerName)
	args := []string{
		"run", "-d", "--name", containerName,
		"-e", "MYSQL_ROOT_PASSWORD=root",
		"-p", fmt.Sprintf("%d:3306", port),
		"mysql:8.0",
	}
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker run failed: %v\n%s", err, out)
	}
}

func waitForMySQL(t *testing.T, ctx context.Context, port int) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		db, err := sql.Open("mysql", mysqlDSNForTest("127.0.0.1", fmt.Sprint(port), "root", "root", ""))
		if err == nil {
			if pingErr := db.PingContext(ctx); pingErr == nil {
				_ = db.Close()
				return
			}
			_ = db.Close()
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("mysql container did not become ready")
}

func assertMySQLCountsEqual(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"guilds", "players", "inventory_items"} {
		var src, dst int
		if err := db.QueryRow("SELECT COUNT(*) FROM game_src." + table).Scan(&src); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM game_restore." + table).Scan(&dst); err != nil {
			t.Fatal(err)
		}
		if src != dst {
			t.Fatalf("unexpected restore count for %s src=%d dst=%d", table, src, dst)
		}
	}
	var sumSrc, sumDst sql.NullString
	if err := db.QueryRow("SELECT CAST(SUM(gold) AS CHAR) FROM game_src.players").Scan(&sumSrc); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT CAST(SUM(gold) AS CHAR) FROM game_restore.players").Scan(&sumDst); err != nil {
		t.Fatal(err)
	}
	if sumSrc.String != sumDst.String {
		t.Fatalf("gold sum mismatch src=%s dst=%s", sumSrc.String, sumDst.String)
	}
}

func assertMySQLDumpReadable(t *testing.T, dir string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one mysql dump, got %d", len(files))
	}
	file, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	raw, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "CREATE TABLE `players`") {
		t.Fatalf("dump did not contain players table")
	}
}

func requireCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Fatalf("%s is required: %v", name, err)
	}
}

func removeDockerContainer(t *testing.T, name string) {
	t.Helper()
	_ = exec.Command("docker", "rm", "-f", name).Run()
}

func dockerMySQLPort(t *testing.T) int {
	t.Helper()
	if value := os.Getenv("DBBACKUP233_DOCKER_MYSQL_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	return freeNonDefaultPort(t)
}

func assertPortAvailable(t *testing.T, port int) {
	t.Helper()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port %d is not available; set DBBACKUP233_DOCKER_MYSQL_PORT to another non-3306 port: %v", port, err)
	}
	_ = l.Close()
}

func freeNonDefaultPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	if port == 3306 {
		return freeNonDefaultPort(t)
	}
	return port
}

func writeFixtureDir(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "fixture-files")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepathFromRoot("tests/sql/files_fixture.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.txt"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

type s3Mock struct {
	*httptest.Server
	bodies [][]byte
}

func newS3Mock(t *testing.T) *s3Mock {
	t.Helper()
	mock := &s3Mock{}
	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
			return
		}
		if r.Method == http.MethodPut {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			mock.bodies = append(mock.bodies, raw)
			w.Header().Set("ETag", `"dbbackup233-test-etag"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	return mock
}

func (m *s3Mock) PutCount() int {
	return len(m.bodies)
}

func (m *s3Mock) AssertUploadedGzipContains(t *testing.T, needle string) {
	t.Helper()
	for _, body := range m.bodies {
		body = decodeAWSChunkedIfNeeded(body)
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(gz)
		_ = gz.Close()
		if err == nil && strings.Contains(string(raw), needle) {
			return
		}
	}
	t.Fatalf("no uploaded gzip body contained %q", needle)
}

func decodeAWSChunkedIfNeeded(raw []byte) []byte {
	if !bytes.Contains(raw[:min(len(raw), 128)], []byte("chunk-signature=")) {
		return raw
	}
	reader := bufio.NewReader(bytes.NewReader(raw))
	var out bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return raw
		}
		line = strings.TrimSpace(line)
		sizeText := line
		if idx := strings.Index(sizeText, ";"); idx >= 0 {
			sizeText = sizeText[:idx]
		}
		size, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil {
			return raw
		}
		if size == 0 {
			return out.Bytes()
		}
		if _, err := io.CopyN(&out, reader, size); err != nil {
			return raw
		}
		_, _ = reader.ReadString('\n')
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
