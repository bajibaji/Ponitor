@echo off
title System Monitor
cd /d "%~dp0"

echo ==========================================
echo   System Monitor - Go binary
echo ==========================================
echo.

start "SystemMonitor" /MIN monitor.exe
timeout /t 3 /nobreak >nul

for /f "tokens=2 delims=:" %%a in ('ipconfig ^| findstr /c:"IPv4" ^| findstr /v "169.254"') do (
    set IP=%%a
    goto :found
)
:found
set IP=%IP: =%

echo   Done. Open on phone (same WiFi):
echo.
echo   http://%IP%:8080
echo.
echo   Keep this window open. Close to stop.
pause >nul
taskkill /FI "WINDOWTITLE eq SystemMonitor" /F >nul 2>&1
