param(
    [string]$HostName = "127.0.0.1",
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$Password = "root",
    [string]$Database = "dbbackup233_it"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Required command not found: go"
}

$env:DBBACKUP233_MYSQL_HOST = $HostName
$env:DBBACKUP233_MYSQL_PORT = "$Port"
$env:DBBACKUP233_MYSQL_USER = $User
$env:DBBACKUP233_MYSQL_PASSWORD = $Password
$env:DBBACKUP233_MYSQL_DATABASE = $Database

Write-Host "Running official mysqldump integration test against ${HostName}:${Port}"
go test -tags mysql_integration ./backup -run TestLocalMySQL80Backup -count=1 -v

$configPath = Join-Path $PWD "config.local.yaml"
$backupDir = Join-Path $PWD "backups-local"
@"
defaults:
  compress: gzip
  timestamp_format: "20060102-150405"
  concurrency: 1
  work_dir: "$($backupDir.Replace('\', '/'))"
  manifest_path: "$($backupDir.Replace('\', '/'))/manifest.jsonl"

sources:
  - name: "mysql80-local"
    type: mysql
    mysql:
      host: "$HostName"
      port: $Port
      user: "$User"
      password: "$Password"
      database: "$Database"
      dump_tool: "mysqldump"
      restore_tool: "mysql"
      mode: "mysqldump"
      single_transaction: true
      quick: true
      routines: true
      triggers: true
      events: true
      set_gtid_purged: "OFF"

targets:
  - name: "local"
    type: local
    local:
      path: "$($backupDir.Replace('\', '/'))"

jobs:
  - name: "mysql80-local"
    source: "mysql80-local"
    targets: ["local"]
"@ | Set-Content -LiteralPath $configPath -Encoding UTF8

Write-Host "Running dbbackup233 CLI backup with config.local.yaml"
go run . backup -c $configPath --timeout 2m
go run . list -c $configPath
Write-Host "Local MySQL integration test passed"
