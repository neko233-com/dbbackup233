package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func cronCmd() *cobra.Command {
	var configPath string
	var install bool
	var name string
	var dryRun bool

	c := &cobra.Command{
		Use:   "cron SCHEDULE",
		Short: "Create a cross-platform scheduled backup command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronInstall(cmd, args[0], configPath, name, install, dryRun)
		},
	}
	c.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file")
	c.Flags().BoolVar(&install, "install", false, "install the schedule on this machine")
	c.Flags().StringVar(&name, "name", "dbbackup233", "schedule name")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print commands without installing")

	c.AddCommand(cronListCmd())
	c.AddCommand(cronRemoveCmd())
	return c
}

func cronListCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "list",
		Short: "List the installed dbbackup233 schedule command",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronList(cmd, name)
		},
	}
	c.Flags().StringVar(&name, "name", "dbbackup233", "schedule name")
	return c
}

func cronRemoveCmd() *cobra.Command {
	var name string
	var dryRun bool
	c := &cobra.Command{
		Use:   "remove",
		Short: "Remove the installed dbbackup233 schedule command",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronRemove(cmd, name, dryRun)
		},
	}
	c.Flags().StringVar(&name, "name", "dbbackup233", "schedule name")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print commands without removing")
	return c
}

func runCronInstall(cmd *cobra.Command, schedule, configPath, name string, install, dryRun bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	backupCmd := fmt.Sprintf("%q backup -c %q", exe, configPath)
	if runtime.GOOS == "windows" {
		command := windowsScheduleCommand(schedule, name, backupCmd)
		fmt.Fprintln(cmd.OutOrStdout(), strings.Join(command, " "))
		if install && !dryRun {
			return exec.Command(command[0], command[1:]...).Run()
		}
		return nil
	}
	line := schedule + " " + backupCmd
	fmt.Fprintln(cmd.OutOrStdout(), line)
	if install && !dryRun {
		script := fmt.Sprintf("(crontab -l 2>/dev/null | grep -v '%s'; echo '%s') | crontab -", name, line+" # "+name)
		return exec.Command("sh", "-c", script).Run()
	}
	return nil
}

func runCronList(cmd *cobra.Command, name string) error {
	command := cronListCommand(name)
	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(command, " "))
	return exec.Command(command[0], command[1:]...).Run()
}

func runCronRemove(cmd *cobra.Command, name string, dryRun bool) error {
	command := cronRemoveCommand(name)
	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(command, " "))
	if dryRun {
		return nil
	}
	return exec.Command(command[0], command[1:]...).Run()
}

func windowsScheduleCommand(schedule, name, backupCmd string) []string {
	parts := strings.Fields(schedule)
	sc := "DAILY"
	st := "02:00"
	if len(parts) >= 2 {
		st = parts[1]
	}
	if strings.HasPrefix(schedule, "@hourly") {
		sc = "HOURLY"
		st = "00:00"
	}
	return []string{"schtasks", "/Create", "/TN", name, "/SC", sc, "/ST", st, "/TR", backupCmd, "/F"}
}

func cronListCommand(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{"schtasks", "/Query", "/TN", name}
	}
	return []string{"sh", "-c", fmt.Sprintf("crontab -l 2>/dev/null | grep '%s'", name)}
}

func cronRemoveCommand(name string) []string {
	if runtime.GOOS == "windows" {
		return []string{"schtasks", "/Delete", "/TN", name, "/F"}
	}
	return []string{"sh", "-c", fmt.Sprintf("crontab -l 2>/dev/null | grep -v '%s' | crontab -", name)}
}
