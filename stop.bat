@echo off
echo Stopping System Monitor...

for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080.*LISTENING"') do taskkill /PID %%a /F >nul 2>&1

echo Done.
pause
