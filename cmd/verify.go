package cmd

import (
	"fmt"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

func verifyCmd() *cobra.Command {
	var configPath string
	var strict bool
	c := &cobra.Command{
		Use:   "verify",
		Short: "Verify config, tools, and writable backup paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := backup.LoadConfig(configPath)
			if err != nil {
				return err
			}
			result := backup.VerifyConfig(cfg)
			for _, warning := range result.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "WARN %s\n", warning)
			}
			for _, problem := range result.Errors {
				fmt.Fprintf(cmd.OutOrStdout(), "ERROR %s\n", problem)
			}
			if !result.OK {
				return fmt.Errorf("verify failed")
			}
			if strict && len(result.Warnings) > 0 {
				return fmt.Errorf("verify strict failed with warnings")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "OK")
			return nil
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	c.Flags().BoolVar(&strict, "strict", false, "treat warnings as errors")
	return c
}
