@echo off
title Rebuild Monitor
cd /d "%~dp0"

echo Stopping old instance...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080.*LISTENING"') do taskkill /PID %%a /F >nul 2>&1
timeout /t 1 /nobreak >nul

echo Building...
windres -i icon.rc -O coff -o icon.syso 2>nul

for /f "delims=- tokens=1" %%v in ('git describe --tags 2^>nul') do set "VER=%%v"
if not defined VER set "VER=v0.0"
set "VER=%VER:v=%"
set "OUT=Ponitor_%VER%.exe"

go build -ldflags="-H=windowsgui -s -w -X main.version=%VER%" -o "%OUT%" .
if %ERRORLEVEL% NEQ 0 (
    echo BUILD FAILED!
    pause
    exit /b 1
)

echo Starting...
start "" "%OUT%"
timeout /t 2 /nobreak >nul

echo Done. Output: %OUT%
pause
