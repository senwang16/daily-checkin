@echo off
cd /d "%~dp0"
echo ==============================
echo   每日自动签到 一键安装
echo ==============================
echo.
set /p T=每天签到时间 (回车默认 09:30): 
if "%T%"=="" set T=09:30
echo.
echo 可选平台: trae (每日+200)  /  workbuddy (每日+100)
set /p P=要启用的平台 (回车默认 trae,workbuddy): 
if "%P%"=="" set P=trae,workbuddy
echo.
echo 正在安装：时间 %T% ，平台 %P%
echo.
daily-checkin.exe install --time=%T% --platforms=%P%
echo.
echo 按任意键关闭...
pause >nul
