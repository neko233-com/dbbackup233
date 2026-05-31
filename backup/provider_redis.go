package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RedisBackupProvider struct{}

func (RedisBackupProvider) Extension(SourceConfig, string) string {
	return "rdb"
}

func (RedisBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Redis
	if cfg.Mode == "copy_rdb" {
		if cfg.RDBPath == "" {
			return fmt.Errorf("redis copy_rdb requires rdb_path")
		}
		return copyFile(cfg.RDBPath, artifactPath)
	}
	args := []string{"-h", cfg.Host, "-p", portArg(cfg.Port)}
	if cfg.Password != "" {
		args = append(args, "-a", cfg.Password)
	}
	if cfg.DB > 0 {
		args = append(args, "-n", portArg(cfg.DB))
	}
	args = append(args, "BGSAVE")
	spec := CommandSpec{Path: cfg.CLIPath, Args: args, Env: os.Environ()}
	ctx.Logf("run: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
		return err
	}
	if cfg.RDBPath == "" {
		return fmt.Errorf("redis bgsave mode requires rdb_path to copy generated dump.rdb")
	}
	return copyFile(cfg.RDBPath, artifactPath)
}

func (RedisBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	dst := source.Redis.RestorePath
	if dst == "" {
		dst = source.Redis.RDBPath
	}
	if dst == "" {
		return fmt.Errorf("redis restore requires restore_path or rdb_path")
	}
	ctx.Logf("copy redis rdb: %s -> %s", artifactPath, dst)
	return copyFile(artifactPath, dst)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
