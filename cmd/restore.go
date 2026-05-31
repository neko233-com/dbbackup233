package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

func restoreCmd() *cobra.Command {
	var configPath string
	var version string
	var dryRun bool
	var yes bool
	var timeout time.Duration

	c := &cobra.Command{
		Use:   "restore JOB",
		Short: "Restore a backup job, optionally selecting a historical version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := backup.LoadConfig(configPath)
			if err != nil {
				return err
			}
			if !dryRun && !yes {
				return fmt.Errorf("restore writes data; rerun with --dry-run to preview or --yes to execute")
			}
			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			manager := backup.NewBackupManager(cfg, backup.RunnerOptions{
				DryRun: dryRun,
				Logf: func(format string, args ...any) {
					fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
				},
			})
			return manager.Restore(ctx, args[0], version)
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	c.Flags().StringVar(&version, "version", "", "historical version to restore; defaults to latest for the job")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned restore without executing it")
	c.Flags().BoolVar(&yes, "yes", false, "execute restore without interactive confirmation")
	c.Flags().DurationVar(&timeout, "timeout", 0, "overall restore timeout, for example 30m")
	return c
}
