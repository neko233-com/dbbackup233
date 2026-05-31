package backup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ConfigBackupProvider struct{}

func (ConfigBackupProvider) Extension(SourceConfig, string) string {
	if runtime.GOOS == "windows" {
		return "zip"
	}
	return "tar.gz"
}

func (ConfigBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.ConfigRepo
	if cfg.Mode == "git_tag" {
		tag := cfg.TagPrefix + "-" + strings.TrimSuffix(filepath.Base(artifactPath), filepath.Ext(artifactPath))
		spec := CommandSpec{Path: "git", Args: []string{"-C", cfg.Path, "tag", tag}, Env: os.Environ()}
		if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
			return err
		}
	}
	fileSource := SourceConfig{Type: "file", File: FileConfig{Paths: []string{cfg.Path}, ArchiveTool: cfg.ArchiveTool}}
	return FileBackupProvider{}.Backup(ctx, fileSource, artifactPath)
}

func (ConfigBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	fileSource := SourceConfig{Type: "file", File: FileConfig{RestoreDir: source.ConfigRepo.RestoreDir, ArchiveTool: source.ConfigRepo.ArchiveTool}}
	return FileBackupProvider{}.Restore(ctx, fileSource, artifactPath)
}
