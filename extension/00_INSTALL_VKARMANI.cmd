@echo off
setlocal
cd /d "%~dp0"
chcp 65001 >nul 2>&1
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup\install.ps1"
set RC=%ERRORLEVEL%
echo.
if not "%RC%"=="0" (
  echo VKarmani setup failed. Error code: %RC%
  echo Run setup\diagnose.ps1 or send the full output to support.
) else (
  echo VKarmani setup completed.
)
echo.
pause
exit /b %RC%
