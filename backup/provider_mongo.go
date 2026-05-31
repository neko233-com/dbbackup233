package backup

import (
	"context"
	"os"
	"strings"
)

type MongoBackupProvider struct{}

func (MongoBackupProvider) Extension(SourceConfig, string) string {
	return "archive.gz"
}

func (MongoBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Mongo
	args := []string{"--uri=" + cfg.URI, "--archive"}
	if cfg.Database != "" {
		args = append(args, "--db="+cfg.Database)
	}
	args = append(args, "--gzip")
	args = append(args, cfg.ExtraArgs...)
	spec := CommandSpec{Path: cfg.DumpTool, Args: args, Env: os.Environ()}
	return writeCommandOutput(ctx, spec, artifactPath, false)
}

func (MongoBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	input, err := commandInputFromArtifact(artifactPath)
	if err != nil {
		return err
	}
	defer input.Close()
	cfg := source.Mongo
	args := []string{"--uri=" + cfg.URI, "--archive", "--gzip"}
	if cfg.Database != "" {
		args = append(args, "--nsInclude="+cfg.Database+".*")
	}
	args = append(args, cfg.RestoreExtraArgs...)
	spec := CommandSpec{Path: cfg.RestoreTool, Args: args, Env: os.Environ()}
	ctx.Logf("run restore: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, input)
}
