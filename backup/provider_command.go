package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CommandBackupProvider struct{}

func (CommandBackupProvider) Extension(source SourceConfig, _ string) string {
	return strings.TrimPrefix(source.Command.Extension, ".")
}

func (CommandBackupProvider) Backup(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Command
	if len(cfg.BackupCommand) == 0 {
		return fmt.Errorf("command backup requires backup_command")
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		return err
	}
	spec := commandSpecFromConfig(cfg.BackupCommand, cfg.Env, artifactPath)
	ctx.Logf("run: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if cfg.CaptureStdout {
		file, err := os.Create(artifactPath)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := runCommand(context.Background(), spec, file, os.Stderr, nil); err != nil {
			return err
		}
		return file.Sync()
	}
	if err := runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil); err != nil {
		return err
	}
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("command backup did not create artifact %q: %w", artifactPath, err)
	}
	return nil
}

func (CommandBackupProvider) Restore(ctx Context, source SourceConfig, artifactPath string) error {
	cfg := source.Command
	if len(cfg.RestoreCommand) == 0 {
		return fmt.Errorf("command restore requires restore_command")
	}
	spec := commandSpecFromConfig(cfg.RestoreCommand, cfg.Env, artifactPath)
	ctx.Logf("run: %s %s", spec.Path, strings.Join(maskDumpArgs(spec.Args), " "))
	if cfg.RestoreStdin {
		file, err := os.Open(artifactPath)
		if err != nil {
			return err
		}
		defer file.Close()
		return runCommand(context.Background(), spec, os.Stdout, os.Stderr, file)
	}
	return runCommand(context.Background(), spec, os.Stdout, os.Stderr, nil)
}

func commandSpecFromConfig(command []string, env map[string]string, artifactPath string) CommandSpec {
	replacer := strings.NewReplacer(
		"${ARTIFACT_PATH}", artifactPath,
		"$ARTIFACT_PATH", artifactPath,
		"{{ARTIFACT_PATH}}", artifactPath,
	)
	args := make([]string, 0, len(command)-1)
	for _, arg := range command[1:] {
		args = append(args, replacer.Replace(arg))
	}
	outEnv := append([]string{}, os.Environ()...)
	outEnv = append(outEnv, "ARTIFACT_PATH="+artifactPath)
	for key, value := range env {
		outEnv = append(outEnv, key+"="+replacer.Replace(value))
	}
	return CommandSpec{Path: replacer.Replace(command[0]), Args: args, Env: outEnv}
}
