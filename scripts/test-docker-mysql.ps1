$ErrorActionPreference = "Stop"

docker --version | Out-Host
if ($env:DBBACKUP233_DOCKER_MYSQL_PORT) {
    Write-Host "Using Docker MySQL host port $env:DBBACKUP233_DOCKER_MYSQL_PORT"
} else {
    Write-Host "Using an automatically selected non-3306 Docker MySQL host port"
}
go test -tags docker_integration ./backup -run TestDockerMySQLBackupRestoreAndHistory -count=1 -v
