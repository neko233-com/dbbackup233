package cmd

import (
	"fmt"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

func pruneCmd() *cobra.Command {
	var configPath string
	var dryRun bool
	c := &cobra.Command{
		Use:   "prune",
		Short: "Prune local backup history using retention config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := backup.LoadConfig(configPath)
			if err != nil {
				return err
			}
			result, err := backup.PruneManifest(cfg.Defaults.ManifestPath, cfg.Retention, backup.PruneOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			for _, art := range result.Deleted {
				fmt.Fprintf(cmd.OutOrStdout(), "delete %s %s %s\n", art.JobName, art.Version, art.FilePath)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "kept=%d deleted=%d\n", len(result.Kept), len(result.Deleted))
			return nil
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print deletions without removing files or rewriting manifest")
	return c
}
