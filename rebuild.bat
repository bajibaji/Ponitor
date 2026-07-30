@echo off
title Rebuild Monitor
cd /d "%~dp0"

echo Stopping old instance...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080.*LISTENING"') do taskkill /PID %%a /F >nul 2>&1
timeout /t 1 /nobreak >nul

echo Building...
windres -i icon.rc -O coff -o icon_windows_amd64.syso 2>nul
go build -ldflags="-H=windowsgui -s -w" -o monitor.exe .
if %ERRORLEVEL% NEQ 0 (
    echo BUILD FAILED!
    pause
    exit /b 1
)

echo Starting...
start "SystemMonitor" /MIN monitor.exe
timeout /t 2 /nobreak >nul

echo Done. Refresh your phone.
pause
