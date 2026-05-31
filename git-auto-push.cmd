@echo off
setlocal
cd /d "%~dp0"

set MSG=%*
if "%MSG%"=="" set MSG=chore: update dbbackup233

git status --short
git add .
if errorlevel 1 exit /b 1

git commit -m "%MSG%"
if errorlevel 1 exit /b 1

git push
if errorlevel 1 exit /b 1

echo Pushed.
