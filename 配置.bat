@echo off
title 每日签到 - 配置向导

echo ===================================
echo   每日签到 - 命令行配置向导
echo   若双击 daily-checkin.exe 报错，可用本脚本兜底
echo ===================================
echo.

set "time=09:30"
set /p time="请输入每天签到时间（HH:MM，回车默认 09:30）："
if "%time%"=="" set "time=09:30"

echo.
echo 请选择要签到的平台（Y=是，N=否）：
choice /C YN /M "是否启用 Trae（每日 +200 积分）"
if %errorlevel%==1 set "trae=yes"
if %errorlevel%==2 set "trae=no"

choice /C YN /M "是否启用 WorkBuddy（每日 +100 积分）"
if %errorlevel%==1 set "wb=yes"
if %errorlevel%==2 set "wb=no"

set "platforms="
if "%trae%"=="yes" set "platforms=trae"
if "%wb%"=="yes" (
  if "%platforms%"=="" (
    set "platforms=workbuddy"
  ) else (
    set "platforms=%platforms%,workbuddy"
  )
)

if "%platforms%"=="" (
  echo 错误：至少选择一个平台
echo 配置未完成。
pause
exit /b 1
)

echo.
echo 即将安装：每天 %time% 自动签到，平台：%platforms%
daily-checkin.exe install --time=%time% --platforms=%platforms%
echo.
pause
