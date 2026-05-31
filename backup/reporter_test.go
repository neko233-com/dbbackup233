package backup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupReporterSendsStartAndComplete(t *testing.T) {
	var events []ReportEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header %q", got)
		}
		var event ReportEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Defaults: Defaults{
			Compress:        "gzip",
			TimestampFormat: "20060102-150405",
			Concurrency:     1,
			WorkDir:         filepath.Join(root, "backups"),
			ManifestPath:    filepath.Join(root, "backups", "manifest.jsonl"),
		},
		Report: ReportConfig{Enabled: true, URL: server.URL, Token: "test-token"},
		Sources: []SourceConfig{{
			Name: "files", Type: "file", File: FileConfig{Paths: []string{src}},
		}},
		Targets: []TargetConfig{{Name: "local", Type: "local", Local: LocalTarget{Path: filepath.Join(root, "local")}}},
		Jobs:    []JobConfig{{Name: "files", Source: "files", Targets: []string{"local"}}},
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := NewBackupManager(cfg, RunnerOptions{Logf: t.Logf}).Backup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 report events, got %d", len(events))
	}
	if events[0].Event != "backup.started" || events[1].Event != "backup.completed" || events[1].Status != "success" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestReporterDisabledDoesNotCallEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	reporter := NewReporter(ReportConfig{Enabled: false, URL: server.URL})
	if err := reporter.Send(context.Background(), ReportEvent{Event: "backup.started"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("disabled reporter called endpoint")
	}
}

func TestReporterRetriesAndAddsIdentity(t *testing.T) {
	calls := 0
	var got ReportEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewDecoder(r.Body).Decode(&got)
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	reporter := NewReporter(
		ReportConfig{Enabled: true, URL: server.URL, Retry: RetryConfig{Attempts: 2, Backoff: "1ms"}},
		IdentityConfig{Project: "game", Cluster: "s1", MachineID: "m1"},
	)
	if err := reporter.Send(context.Background(), ReportEvent{Event: "backup.started", Status: "started"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if got.Project != "game" || got.Cluster != "s1" || got.MachineID != "m1" {
		t.Fatalf("identity not sent: %+v", got)
	}
}
