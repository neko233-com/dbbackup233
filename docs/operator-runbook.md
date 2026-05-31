# Operator Runbook

This runbook is for installing and operating `dbbackup233` on a game server.

## 1. Install

Windows PowerShell:

```powershell
iwr https://raw.githubusercontent.com/neko233-com/dbbackup233/main/scripts/install.ps1 -UseB | iex
```

Linux/macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/dbbackup233/main/scripts/install.sh | sh
```

Verify:

```bash
dbbackup233 version
```

Published release binaries support:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64
- windows/arm64

Local platform matrix check:

```powershell
.\scripts\check-platform-matrix.ps1
```

## 2. Install Dump Tools

MySQL:

```bash
dbbackup233 install mysqldump
```

PostgreSQL:

```bash
dbbackup233 install pg_dump
```

## 3. Configure

Copy `config.yaml` to `config.local.yaml` and edit local secrets and paths.

Do not commit `config.local.yaml`.

## 4. Verify

```bash
dbbackup233 verify -c config.local.yaml
```

Warnings mean the machine may still run some jobs, but missing tools or paths
should be fixed before scheduling.

For strict mode:

```bash
dbbackup233 verify -c config.local.yaml --strict
```

## 5. Dry Run

```bash
dbbackup233 backup -c config.local.yaml --dry-run
```

## 6. Run Backup

```bash
dbbackup233 backup -c config.local.yaml
```

Run selected jobs only:

```bash
dbbackup233 backup -c config.local.yaml --job mysql-game --job server-files
```

## 7. List History

```bash
dbbackup233 list -c config.local.yaml
```

## 8. Restore

Preview:

```bash
dbbackup233 restore mysql-game --version 20260531-120000 -c config.local.yaml --dry-run
```

Execute:

```bash
dbbackup233 restore mysql-game --version 20260531-120000 -c config.local.yaml --yes
```

Restore intentionally requires `--yes` unless `--dry-run` is used.
Before writing data, restore verifies the artifact SHA-256 recorded in the
manifest. If the local file changed or is incomplete, restore stops.

## 9. Schedule

Preview:

```bash
dbbackup233 cron "0 2 * * *" -c config.local.yaml --dry-run
```

Install:

```bash
dbbackup233 cron "0 2 * * *" -c config.local.yaml --install
```

Windows uses `schtasks`. Linux/macOS use `crontab`.

List or remove the schedule:

```bash
dbbackup233 cron list
dbbackup233 cron remove --dry-run
dbbackup233 cron remove
```

## 10. Update

Latest release:

```bash
dbbackup233 update
```

Check only:

```bash
dbbackup233 update --check
```

Pinned release:

```bash
dbbackup233 update --tag v0.1.0
```

Update is hot-update-next-run: the running backup process is not interrupted.
The installed binary is replaced so the next cron or manual run uses the new
version.

## 11. Emergency Checklist

When a backup fails:

1. Run `dbbackup233 verify -c config.local.yaml --strict`.
2. Check free disk space for `work_dir`.
3. Check database dump tool availability.
4. Check database credentials.
5. Check target directory or object storage credentials.
6. Check HTTP report endpoint logs if reporting is enabled.
