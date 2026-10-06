@echo off
setlocal
cd /d "%~dp0"
chcp 65001 >nul 2>&1
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup\uninstall.ps1"
set RC=%ERRORLEVEL%
echo.
pause
exit /b %RC%
