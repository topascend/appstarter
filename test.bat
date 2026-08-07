@echo off
setlocal enabledelayedexpansion
title 计算机信息报告
color 0A
echo ============================================================
echo                   计算机信息报告
echo ============================================================
echo.

:: 基本系统信息
echo [系统概况]
echo 计算机名    : %COMPUTERNAME%
echo 用户名      : %USERNAME%
echo 当前日期    : %DATE% %TIME%
echo.

:: 操作系统信息（使用 systeminfo 过滤关键字段）
echo [操作系统]
systeminfo | findstr /B /C:"OS 名称" /C:"OS 版本" /C:"系统类型" /C:"系统启动时间" /C:"处理器"
echo.

:: CPU 详细信息
echo [处理器]
wmic cpu get name,numberofcores,numberoflogicalprocessors /format:list | findstr "="
echo.

:: 物理内存
echo [物理内存]
for /f "tokens=2 delims==" %%a in ('wmic computersystem get totalphysicalmemory /format:list ^| find "="') do set "mem=%%a"
set /a mem_mb=!mem!/1048576
set /a mem_gb=!mem!/1073741824
echo 总内存      : !mem_gb! GB (!mem_mb! MB)
echo.

:: 内存条详细信息（插槽、速度）
wmic memorychip get capacity,speed,manufacturer,partnumber /format:list | findstr "="
echo.

:: 磁盘分区（仅本地硬盘，类型=3）
echo [磁盘分区]
wmic logicaldisk where drivetype=3 get deviceid,volumename,size,freespace /format:table
echo.

:: 网络 IP 地址
echo [网络 IP 地址]
ipconfig | findstr "IPv4 地址"
echo.

:: 可选：MAC 地址
echo [物理地址]
ipconfig /all | findstr "物理地址" | findstr /v "隧道"
echo.

:: 系统运行时间（从 systeminfo 中提取）
for /f "tokens=2 delims=:" %%a in ('systeminfo ^| find "系统启动时间"') do set "boot=%%a"
echo 系统启动时间: %boot%
echo.

echo ============================================================
echo 信息收集完毕。
pause