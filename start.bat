@echo off
cd /d "%~dp0"
:: 静默后台启动（GUI 二进制，无控制台窗口），双击即运行，靠 stop.bat 停止
start "" monitor.exe
