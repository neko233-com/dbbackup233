package backup

import (
	"context"
	"os"
	"strings"
)

type PostgreSQLBackupProvider struct{}

func (PostgreSQLBackupProvider) Extension(source SourceConfig, compress string) string {
	if source.Postgres.Format == "plain" {
		if compress == "gzip" {
			return "sql.gz"
		}
		return "sql"
	}
	return "dump"
}

func (PostgreSQLBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	spec := BuildPostgresDumpCommand(source.Postgres)
	return writeCommandOutput(ctx, spec, artifactPath, strings.HasSuffix(artifactPath, ".gz"))
}

func (PostgreSQLBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	input, err := commandInputFromArtifact(artifactPath)
	if err != nil {
		return err
	}
	defer input.Close()
	spec := BuildPostgresRestoreCommand(source.Postgres, artifactPath)
	ctx.Logf("run restore: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, input)
}

func BuildPostgresDumpCommand(cfg PostgresConfig) CommandSpec {
	path := cfg.DumpTool
	prefix := cfg.ExtraArgs
	args := []string{
		"--host=" + cfg.Host,
		"--port=" + portArg(cfg.Port),
		"--username=" + cfg.User,
		"--dbname=" + cfg.Database,
		"--format=" + cfg.Format,
		"--no-password",
	}
	if cfg.Format == "directory" && cfg.Jobs > 1 {
		args = append(args, "--jobs="+portArg(cfg.Jobs))
	}
	args = append(prefix, args...)
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}
	return CommandSpec{Path: path, Args: args, Env: env}
}

func BuildPostgresRestoreCommand(cfg PostgresConfig, artifactPath string) CommandSpec {
	tool := cfg.RestoreTool
	prefix := cfg.RestoreExtraArgs
	args := []string{
		"--host=" + cfg.Host,
		"--port=" + portArg(cfg.Port),
		"--username=" + cfg.User,
	}
	restoreDB := cfg.RestoreDatabase
	if restoreDB == "" {
		restoreDB = cfg.Database
	}
	args = append(args, "--dbname="+restoreDB)
	if (cfg.Format == "plain" || strings.HasSuffix(artifactPath, ".sql") || strings.HasSuffix(artifactPath, ".sql.gz")) && (cfg.RestoreTool == "" || cfg.RestoreTool == DefaultDumpToolName("pg_restore")) {
		tool = DefaultDumpToolName("psql")
		args = append(args, "--no-password")
	} else {
		args = append(args, "--no-password")
	}
	args = append(prefix, args...)
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}
	return CommandSpec{Path: tool, Args: args, Env: env}
}
