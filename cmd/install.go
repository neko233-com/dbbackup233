package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/neko233-com/dbbackup233/backup"
	"github.com/spf13/cobra"
)

type installPlan struct {
	Manager string
	Args    []string
}

func installCmd() *cobra.Command {
	var dryRun bool
	var force bool

	c := &cobra.Command{
		Use:   "install [mysqldump|xtrabackup|pg_dump|pg_basebackup|mongodump|redis-cli|clickhouse-client|all]",
		Short: "Install or verify official database dump tools",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tools, err := expandInstallTools(args[0])
			if err != nil {
				return err
			}
			for _, tool := range tools {
				if err := installTool(cmd.Context(), cmd, tool, dryRun, force); err != nil {
					return err
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print install commands without executing them")
	c.Flags().BoolVar(&force, "force", false, "run installer even if the tool already exists in PATH")
	return c
}

func expandInstallTools(value string) ([]string, error) {
	switch strings.ToLower(value) {
	case "mysql", "mysqldump":
		return []string{"mysqldump"}, nil
	case "xtrabackup", "percona-xtrabackup":
		return []string{"xtrabackup"}, nil
	case "postgres", "postgresql", "pg_dump":
		return []string{"pg_dump"}, nil
	case "pg_basebackup":
		return []string{"pg_basebackup"}, nil
	case "mongo", "mongodb", "mongodump":
		return []string{"mongodump"}, nil
	case "redis", "redis-cli":
		return []string{"redis-cli"}, nil
	case "clickhouse", "clickhouse-client":
		return []string{"clickhouse-client"}, nil
	case "all":
		return []string{"mysqldump", "xtrabackup", "pg_dump", "mongodump", "redis-cli", "clickhouse-client"}, nil
	default:
		return nil, fmt.Errorf("unsupported tool %q", value)
	}
}

func installTool(ctx context.Context, cmd *cobra.Command, tool string, dryRun, force bool) error {
	exe := backup.DefaultDumpToolName(tool)
	if path, err := exec.LookPath(exe); err == nil && !force {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already installed: %s\n", tool, path)
		return nil
	}

	plan, err := detectInstallPlan(tool)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "install %s using: %s %s\n", tool, plan.Manager, strings.Join(plan.Args, " "))
	if dryRun {
		return nil
	}

	install := exec.CommandContext(ctx, plan.Manager, plan.Args...)
	install.Stdout = cmd.OutOrStdout()
	install.Stderr = cmd.ErrOrStderr()
	install.Stdin = os.Stdin
	if err := install.Run(); err != nil {
		return err
	}
	if path, err := exec.LookPath(exe); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s installed: %s\n", tool, path)
		return nil
	}
	return fmt.Errorf("%s installer finished but %s was not found in PATH", tool, exe)
}

func detectInstallPlan(tool string) (installPlan, error) {
	switch runtime.GOOS {
	case "windows":
		return detectWindowsPlan(tool)
	case "darwin":
		return detectDarwinPlan(tool)
	case "linux":
		return detectLinuxPlan(tool)
	default:
		return installPlan{}, fmt.Errorf("automatic install is not supported on %s", runtime.GOOS)
	}
}

func detectWindowsPlan(tool string) (installPlan, error) {
	pkg := "Oracle.MySQL"
	if tool == "pg_dump" || tool == "pg_basebackup" {
		pkg = "PostgreSQL.PostgreSQL"
	} else if tool == "mongodump" {
		pkg = "MongoDB.DatabaseTools"
	} else if tool == "redis-cli" {
		pkg = "Redis.Redis"
	} else if tool == "xtrabackup" {
		pkg = "Percona.PerconaXtraBackup"
	} else if tool == "clickhouse-client" {
		pkg = "ClickHouse.ClickHouse"
	}
	if _, err := exec.LookPath("winget.exe"); err == nil {
		return installPlan{Manager: "winget.exe", Args: []string{"install", "--id", pkg, "-e"}}, nil
	}
	if _, err := exec.LookPath("choco.exe"); err == nil {
		chocoPkg := "mysql"
		if tool == "pg_dump" || tool == "pg_basebackup" {
			chocoPkg = "postgresql"
		} else if tool == "mongodump" {
			chocoPkg = "mongodb-database-tools"
		} else if tool == "redis-cli" {
			chocoPkg = "redis-64"
		} else if tool == "xtrabackup" {
			chocoPkg = "percona-xtrabackup"
		} else if tool == "clickhouse-client" {
			chocoPkg = "clickhouse"
		}
		return installPlan{Manager: "choco.exe", Args: []string{"install", chocoPkg, "-y"}}, nil
	}
	return installPlan{}, fmt.Errorf("no supported Windows package manager found; install winget or choco")
}

func detectDarwinPlan(tool string) (installPlan, error) {
	if _, err := exec.LookPath("brew"); err != nil {
		return installPlan{}, fmt.Errorf("brew is required for automatic install on macOS")
	}
	pkg := "mysql-client"
	if tool == "pg_dump" || tool == "pg_basebackup" {
		pkg = "libpq"
	} else if tool == "mongodump" {
		pkg = "mongodb/brew/mongodb-database-tools"
	} else if tool == "redis-cli" {
		pkg = "redis"
	} else if tool == "xtrabackup" {
		pkg = "percona-xtrabackup"
	} else if tool == "clickhouse-client" {
		pkg = "clickhouse"
	}
	return installPlan{Manager: "brew", Args: []string{"install", pkg}}, nil
}

func detectLinuxPlan(tool string) (installPlan, error) {
	packages := linuxPackages(tool)
	if _, err := exec.LookPath("apt-get"); err == nil {
		return installPlan{Manager: "sudo", Args: append([]string{"apt-get", "install", "-y"}, packages.apt...)}, nil
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return installPlan{Manager: "sudo", Args: append([]string{"dnf", "install", "-y"}, packages.rpm...)}, nil
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return installPlan{Manager: "sudo", Args: append([]string{"yum", "install", "-y"}, packages.rpm...)}, nil
	}
	if _, err := exec.LookPath("apk"); err == nil {
		return installPlan{Manager: "sudo", Args: append([]string{"apk", "add"}, packages.apk...)}, nil
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return installPlan{Manager: "sudo", Args: append([]string{"pacman", "-S", "--noconfirm"}, packages.pacman...)}, nil
	}
	return installPlan{}, fmt.Errorf("no supported Linux package manager found")
}

type linuxPackageNames struct {
	apt    []string
	rpm    []string
	apk    []string
	pacman []string
}

func linuxPackages(tool string) linuxPackageNames {
	if tool == "pg_dump" || tool == "pg_basebackup" {
		return linuxPackageNames{
			apt:    []string{"postgresql-client"},
			rpm:    []string{"postgresql"},
			apk:    []string{"postgresql-client"},
			pacman: []string{"postgresql"},
		}
	}
	if tool == "mongodump" {
		return linuxPackageNames{
			apt:    []string{"mongodb-database-tools"},
			rpm:    []string{"mongodb-database-tools"},
			apk:    []string{"mongodb-tools"},
			pacman: []string{"mongodb-tools"},
		}
	}
	if tool == "redis-cli" {
		return linuxPackageNames{
			apt:    []string{"redis-tools"},
			rpm:    []string{"redis"},
			apk:    []string{"redis"},
			pacman: []string{"redis"},
		}
	}
	if tool == "xtrabackup" {
		return linuxPackageNames{
			apt:    []string{"percona-xtrabackup-80"},
			rpm:    []string{"percona-xtrabackup-80"},
			apk:    []string{"percona-xtrabackup"},
			pacman: []string{"percona-xtrabackup"},
		}
	}
	if tool == "clickhouse-client" {
		return linuxPackageNames{
			apt:    []string{"clickhouse-client"},
			rpm:    []string{"clickhouse-client"},
			apk:    []string{"clickhouse-client"},
			pacman: []string{"clickhouse-client"},
		}
	}
	return linuxPackageNames{
		apt:    []string{"mysql-client"},
		rpm:    []string{"mysql"},
		apk:    []string{"mysql-client"},
		pacman: []string{"mysql-clients"},
	}
}
