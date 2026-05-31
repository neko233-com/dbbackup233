package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

func backupCmd() *cobra.Command {
	var configPath string
	var dryRun bool
	var timeout time.Duration

	c := &cobra.Command{
		Use:   "backup",
		Short: "Run configured database backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := backup.LoadConfig(configPath)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			runner := backup.NewRunner(cfg, backup.RunnerOptions{
				DryRun: dryRun,
				Logf: func(format string, args ...any) {
					fmt.Fprintf(cmd.OutOrStdout(), format+"\n", args...)
				},
			})
			return runner.Run(ctx)
		},
	}

	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned work without running dump commands or uploads")
	c.Flags().DurationVar(&timeout, "timeout", 0, "overall backup timeout, for example 30m")
	return c
}
