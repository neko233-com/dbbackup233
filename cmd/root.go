package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "dbbackup233",
	Short: "Cross-platform database backup CLI and Go library",
	Long: `dbbackup233 backs up one or many databases to local storage or
S3-compatible object storage.

It is designed for game servers and small teams that colocate MySQL with
application servers for low latency and need predictable, low-pressure database
backups without depending on cloud RDS snapshots.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(cronCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(pruneCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(versionCmd())
}
