package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MySQLBackupProvider struct{}

func (MySQLBackupProvider) Extension(source SourceConfig, _ string) string {
	if isXtraBackupMode(source.MySQL.Mode) {
		return "xb.tar.gz"
	}
	return "sql.gz"
}

func (MySQLBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	if isXtraBackupMode(source.MySQL.Mode) {
		return backupMySQLXtraBackup(ctx, source.MySQL, artifactPath)
	}
	spec := BuildMySQLDumpCommand(source.MySQL)
	return writeCommandOutput(ctx, spec, artifactPath, strings.HasSuffix(artifactPath, ".gz"))
}

func (MySQLBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	if isXtraBackupMode(source.MySQL.Mode) {
		return restoreMySQLXtraBackup(ctx, source.MySQL, artifactPath)
	}
	input, err := commandInputFromArtifact(artifactPath)
	if err != nil {
		return err
	}
	defer input.Close()
	spec := BuildMySQLRestoreCommand(source.MySQL)
	ctx.Logf("run restore: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, input)
}

func backupMySQLXtraBackup(ctx Context, cfg MySQLConfig, artifactPath string) error {
	if err := os.MkdirAll(cfg.XtraBackupDir, 0o755); err != nil {
		return err
	}
	if cfg.Mode == "xtrabackup-incremental" && cfg.IncrementalBaseDir == "" {
		baseDir, err := readXtraLatestBase(cfg.XtraBackupDir)
		if err != nil {
			return err
		}
		cfg.IncrementalBaseDir = baseDir
	}
	name := filepath.Base(strings.TrimSuffix(strings.TrimSuffix(artifactPath, ".tar.gz"), ".xb"))
	targetDir := filepath.Join(cfg.XtraBackupDir, name)
	commandTargetDir := targetDir
	if cfg.XtraCommandDir != "" {
		commandTargetDir = filepath.ToSlash(filepath.Join(cfg.XtraCommandDir, name))
		cfg.IncrementalBaseDir = mapXtraCommandPath(cfg, cfg.IncrementalBaseDir)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	spec := BuildMySQLXtraBackupCommand(cfg, commandTargetDir)
	ctx.Logf("run xtrabackup: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
		return err
	}
	if err := archiveDirectory(targetDir, artifactPath); err != nil {
		return err
	}
	return writeXtraLatestBase(cfg.XtraBackupDir, targetDir)
}

func mapXtraCommandPath(cfg MySQLConfig, hostPath string) string {
	if cfg.XtraCommandDir == "" || hostPath == "" {
		return hostPath
	}
	rel, err := filepath.Rel(cfg.XtraBackupDir, hostPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return hostPath
	}
	return filepath.ToSlash(filepath.Join(cfg.XtraCommandDir, rel))
}

func restoreMySQLXtraBackup(ctx Context, cfg MySQLConfig, artifactPath string) error {
	restoreDir := cfg.XtraRestoreDir
	if restoreDir == "" {
		restoreDir = strings.TrimSuffix(artifactPath, ".tar.gz")
		restoreDir = restoreDir + "-restore"
	}
	if err := os.RemoveAll(restoreDir); err != nil {
		return err
	}
	if err := os.MkdirAll(restoreDir, 0o755); err != nil {
		return err
	}
	if err := extractTarGzip(artifactPath, restoreDir); err != nil {
		return err
	}
	spec := BuildMySQLXtraPrepareCommand(cfg, restoreDir)
	ctx.Logf("run xtrabackup prepare: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
		return err
	}
	ctx.Logf("prepared xtrabackup restore dir: %s", restoreDir)
	ctx.Logf("manual restore required: stop mysql, run xtrabackup --copy-back --target-dir=%s, then fix datadir ownership", restoreDir)
	return nil
}

func BuildMySQLDumpCommand(cfg MySQLConfig) CommandSpec {
	path := cfg.DumpTool
	prefix := cfg.ExtraArgs
	args := []string{
		"--host=" + cfg.Host,
		"--port=" + portArg(cfg.Port),
		"--user=" + cfg.User,
		"--default-character-set=utf8mb4",
	}
	if cfg.NoCreateDB {
		args = append(args, cfg.Database)
	} else {
		args = append(args, "--databases", cfg.Database)
	}
	args = append(args, boolArg(cfg.SingleTransaction, "--single-transaction")...)
	args = append(args, boolArg(cfg.Quick, "--quick")...)
	args = append(args, boolArg(cfg.Routines, "--routines")...)
	args = append(args, boolArg(cfg.Triggers, "--triggers")...)
	args = append(args, boolArg(cfg.Events, "--events")...)
	if cfg.SetGTIDPurged != "" {
		args = append(args, "--set-gtid-purged="+cfg.SetGTIDPurged)
	}
	args = append(prefix, args...)
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+cfg.Password)
	}
	return CommandSpec{Path: path, Args: args, Env: env}
}

func BuildMySQLRestoreCommand(cfg MySQLConfig) CommandSpec {
	path := cfg.RestoreTool
	prefix := cfg.RestoreExtraArgs
	args := []string{
		"--host=" + cfg.Host,
		"--port=" + portArg(cfg.Port),
		"--user=" + cfg.User,
		"--default-character-set=utf8mb4",
	}
	if cfg.RestoreDatabase != "" {
		args = append(args, cfg.RestoreDatabase)
	}
	args = append(prefix, args...)
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+cfg.Password)
	}
	return CommandSpec{Path: path, Args: args, Env: env}
}

func BuildMySQLXtraBackupCommand(cfg MySQLConfig, targetDir string) CommandSpec {
	args := []string{
		"--backup",
		"--host=" + cfg.Host,
		"--port=" + portArg(cfg.Port),
		"--user=" + cfg.User,
		"--target-dir=" + targetDir,
	}
	if cfg.Password != "" {
		args = append(args, "--password="+cfg.Password)
	}
	if cfg.Database != "" {
		args = append(args, "--databases="+cfg.Database)
	}
	if cfg.Mode == "xtrabackup-incremental" {
		args = append(args, "--incremental-basedir="+cfg.IncrementalBaseDir)
	}
	args = append(cfg.XtraExtraArgs, args...)
	return CommandSpec{Path: cfg.XtraBackupTool, Args: args, Env: mysqlEnv(cfg)}
}

func BuildMySQLXtraPrepareCommand(cfg MySQLConfig, targetDir string) CommandSpec {
	args := []string{
		"--prepare",
		"--target-dir=" + targetDir,
	}
	args = append(cfg.PrepareExtraArgs, args...)
	return CommandSpec{Path: cfg.XtraBackupTool, Args: args, Env: mysqlEnv(cfg)}
}

func mysqlEnv(cfg MySQLConfig) []string {
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+cfg.Password)
	}
	return env
}

func ValidateXtraBackupChain(cfg MySQLConfig) error {
	if cfg.Mode != "xtrabackup-incremental" {
		return nil
	}
	baseDir := cfg.IncrementalBaseDir
	if baseDir == "" {
		var err error
		baseDir, err = readXtraLatestBase(cfg.XtraBackupDir)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(baseDir, "xtrabackup_checkpoints")); err != nil {
		return fmt.Errorf("incremental base dir %s has no xtrabackup_checkpoints: %w", baseDir, err)
	}
	return nil
}

func readXtraLatestBase(xtraBackupDir string) (string, error) {
	path := filepath.Join(xtraBackupDir, "latest-base.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read xtrabackup latest base %s: %w", path, err)
	}
	baseDir := strings.TrimSpace(string(raw))
	if baseDir == "" {
		return "", fmt.Errorf("xtrabackup latest base %s is empty", path)
	}
	return baseDir, nil
}

func writeXtraLatestBase(xtraBackupDir, targetDir string) error {
	path := filepath.Join(xtraBackupDir, "latest-base.txt")
	return os.WriteFile(path, []byte(targetDir+"\n"), 0o600)
}
