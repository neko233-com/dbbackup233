package backup

import (
	"context"
	"os"
	"strings"
)

type MySQLBackupProvider struct{}

func (MySQLBackupProvider) Extension(SourceConfig, string) string {
	return "sql.gz"
}

func (MySQLBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	spec := BuildMySQLDumpCommand(source.MySQL)
	return writeCommandOutput(ctx, spec, artifactPath, strings.HasSuffix(artifactPath, ".gz"))
}

func (MySQLBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	input, err := commandInputFromArtifact(artifactPath)
	if err != nil {
		return err
	}
	defer input.Close()
	spec := BuildMySQLRestoreCommand(source.MySQL)
	ctx.Logf("run restore: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, input)
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
