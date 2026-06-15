package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ClickHouseBackupProvider struct{}

func (ClickHouseBackupProvider) Extension(SourceConfig, string) string {
	return "clickhouse-backup.json"
}

func (ClickHouseBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.ClickHouse
	query := buildClickHouseBackupQuery(cfg)
	spec := BuildClickHouseCommand(cfg, query, cfg.ExtraArgs)
	ctx.Logf("run clickhouse backup: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
		return err
	}
	receipt := map[string]any{
		"engine":      "clickhouse-backup",
		"mode":        cfg.Mode,
		"database":    cfg.Database,
		"table":       cfg.Table,
		"backup_name": cfg.BackupName,
		"destination": cfg.BackupDestination,
		"base_backup": cfg.BaseBackup,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
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

func (ClickHouseBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.ClickHouse
	query := buildClickHouseRestoreQuery(cfg)
	spec := BuildClickHouseCommand(cfg, query, cfg.RestoreExtraArgs)
	ctx.Logf("run clickhouse restore: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
}

func BuildClickHouseCommand(cfg ClickHouseConfig, query string, extra []string) CommandSpec {
	args := []string{
		"--host", cfg.Host,
		"--port", portArg(cfg.Port),
		"--user", cfg.User,
		"--query", query,
	}
	if cfg.Password != "" {
		args = append(args, "--password", cfg.Password)
	}
	args = append(extra, args...)
	return CommandSpec{Path: cfg.ClientTool, Args: args, Env: os.Environ()}
}

func buildClickHouseBackupQuery(cfg ClickHouseConfig) string {
	target := "DATABASE " + quoteClickHouseIdent(cfg.Database)
	if cfg.Table != "" {
		target = "TABLE " + quoteClickHouseIdent(cfg.Database) + "." + quoteClickHouseIdent(cfg.Table)
	}
	query := fmt.Sprintf("BACKUP %s TO %s", target, cfg.BackupDestination)
	if cfg.Mode == "incremental" && cfg.BaseBackup != "" {
		query += " SETTINGS base_backup = " + cfg.BaseBackup
	}
	return query
}

func buildClickHouseRestoreQuery(cfg ClickHouseConfig) string {
	target := "DATABASE " + quoteClickHouseIdent(cfg.Database)
	if cfg.Table != "" {
		target = "TABLE " + quoteClickHouseIdent(cfg.Database) + "." + quoteClickHouseIdent(cfg.Table)
	}
	return fmt.Sprintf("RESTORE %s FROM %s", target, cfg.BackupDestination)
}

func quoteClickHouseIdent(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}
