package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type FileBackupProvider struct{}

func (FileBackupProvider) Extension(SourceConfig, string) string {
	if runtime.GOOS == "windows" {
		return "zip"
	}
	return "tar.gz"
}

func (FileBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.File
	if runtime.GOOS == "windows" {
		quotedPaths := make([]string, len(cfg.Paths))
		for i, value := range cfg.Paths {
			quotedPaths[i] = quotePS(value)
		}
		args := []string{"-NoProfile", "-Command", "Compress-Archive -Path " + strings.Join(quotedPaths, ",") + " -DestinationPath " + quotePS(artifactPath) + " -Force"}
		spec := CommandSpec{Path: cfg.ArchiveTool, Args: args, Env: os.Environ()}
		ctx.Logf("run: %s %s", spec.Path, strings.Join(spec.Args, " "))
		return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
	}
	args := []string{"-czf", artifactPath}
	for _, exclude := range cfg.Exclude {
		args = append(args, "--exclude="+exclude)
	}
	args = append(args, cfg.Paths...)
	spec := CommandSpec{Path: cfg.ArchiveTool, Args: args, Env: os.Environ()}
	ctx.Logf("run: %s %s", spec.Path, strings.Join(spec.Args, " "))
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
}

func (FileBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	restoreDir := source.File.RestoreDir
	if restoreDir == "" {
		return fmt.Errorf("file restore requires restore_dir")
	}
	if err := ensureDir(restoreDir); err != nil {
		return err
	}
	if runtime.GOOS == "windows" || strings.HasSuffix(artifactPath, ".zip") {
		args := []string{"-NoProfile", "-Command", "Expand-Archive -Path " + quotePS(artifactPath) + " -DestinationPath " + quotePS(restoreDir) + " -Force"}
		spec := CommandSpec{Path: source.File.ArchiveTool, Args: args, Env: os.Environ()}
		return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
	}
	spec := CommandSpec{Path: source.File.ArchiveTool, Args: []string{"-xzf", artifactPath, "-C", restoreDir}, Env: os.Environ()}
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
}

func quotePS(value string) string {
	return "'" + strings.ReplaceAll(filepath.Clean(value), "'", "''") + "'"
}
