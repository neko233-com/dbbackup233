package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ElasticBackupProvider struct{}

func (ElasticBackupProvider) Extension(SourceConfig, string) string {
	return "es-snapshot.json"
}

func (ElasticBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Elastic
	snapshot := cfg.Snapshot
	body := map[string]any{
		"indices":              strings.Join(cfg.Indices, ","),
		"include_global_state": cfg.IncludeGlobalState,
	}
	for key, value := range cfg.ExtraBody {
		body[key] = value
	}
	if len(cfg.Indices) == 0 {
		delete(body, "indices")
	}
	if err := elasticRequest(cfg, http.MethodPut, fmt.Sprintf("/_snapshot/%s/%s?wait_for_completion=true", cfg.Repository, snapshot), body, os.Stdout); err != nil {
		return err
	}
	receipt := map[string]any{
		"engine":     "elasticsearch-snapshot",
		"repository": cfg.Repository,
		"snapshot":   snapshot,
		"url":        cfg.URL,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"mode":       cfg.Mode,
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(artifactPath, raw, 0o600)
}

func (ElasticBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Elastic
	body := map[string]any{}
	for key, value := range cfg.RestoreExtraBody {
		body[key] = value
	}
	ctx.Logf("restore elasticsearch snapshot: repository=%s snapshot=%s", cfg.Repository, cfg.Snapshot)
	return elasticRequest(cfg, http.MethodPost, fmt.Sprintf("/_snapshot/%s/%s/_restore?wait_for_completion=true", cfg.Repository, cfg.Snapshot), body, os.Stdout)
}

func elasticRequest(cfg ElasticConfig, method, path string, body map[string]any, out io.Writer) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := strings.TrimRight(cfg.URL, "/") + path
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}
	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if out != nil && len(respBody) > 0 {
		_, _ = out.Write(respBody)
		_, _ = out.Write([]byte("\n"))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch request %s %s failed: %s: %s", method, url, resp.Status, string(respBody))
	}
	return nil
}
