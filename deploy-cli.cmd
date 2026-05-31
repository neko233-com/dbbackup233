@echo off
setlocal
cd /d "%~dp0"

set VERSION=%~1
if "%VERSION%"=="" (
  echo Usage: deploy-cli.cmd v0.1.0
  exit /b 1
)

go test ./...
if errorlevel 1 exit /b 1

git tag %VERSION%
if errorlevel 1 exit /b 1

git push origin %VERSION%
if errorlevel 1 exit /b 1

echo Pushed release tag %VERSION%.
