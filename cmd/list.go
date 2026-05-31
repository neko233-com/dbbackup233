package cmd

import (
	"fmt"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var configPath string
	c := &cobra.Command{
		Use:   "list",
		Short: "List backup history from the local manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := backup.LoadConfig(configPath)
			if err != nil {
				return err
			}
			arts, err := backup.ListArtifacts(cfg.Defaults.ManifestPath)
			if err != nil {
				return err
			}
			for _, art := range arts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%d\t%s\n", art.Version, art.JobName, art.SourceType, art.FilePath, art.Size, art.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	return c
}
