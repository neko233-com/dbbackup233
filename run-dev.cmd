@echo off
setlocal
cd /d "%~dp0"

if "%~1"=="" (
  go run . --help
) else (
  go run . %*
)
