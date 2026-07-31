@echo off
cd /d "%~dp0"
:: 静默后台启动（GUI 二进制，无控制台窗口），双击即运行，靠 stop.bat 停止
for %%f in (Ponitor_*.exe) do start "" "%%f" & goto :started
echo No build found. Run rebuild.bat first.
pause
exit /b 1
:started
