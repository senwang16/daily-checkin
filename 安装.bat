@echo off
cd /d "%~dp0"
daily-checkin.exe install
echo.
echo 已安装：每天 09:30 自动签到（Trae + WorkBuddy），关机再开机登录后自动补签。
echo 如需自定义时间/平台，请改用双击 daily-checkin.exe 打开安装向导。
echo.
echo 按任意键关闭...
pause >nul
