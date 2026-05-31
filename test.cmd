@echo off
setlocal
cd /d "%~dp0"

echo == gofmt check ==
for /f "delims=" %%f in ('gofmt -l .') do (
  echo gofmt needed: %%f
  exit /b 1
)

echo == go test ./... ==
go test ./...
if errorlevel 1 exit /b 1

echo == go build ./... ==
go build ./...
if errorlevel 1 exit /b 1

echo == docker mysql integration ==
go test -tags docker_integration ./backup -run TestDockerMySQLBackupRestoreAndHistory -count=1 -v
if errorlevel 1 exit /b 1

echo == docker postgres integration ==
go test -tags docker_integration ./backup -run TestDockerPostgresBackupRestoreAndHistory -count=1 -v
if errorlevel 1 exit /b 1

echo All tests passed.
