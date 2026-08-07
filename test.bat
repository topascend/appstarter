@echo off
setlocal enabledelayedexpansion
title 计算机信息报�?color 0A
echo ============================================================
echo                   计算机信息报�?echo ============================================================
echo.

:: 基本系统信息
echo [系统概况]
echo 计算机名    : %COMPUTERNAME%
echo 用户�?     : %USERNAME%
echo 当前日期    : %DATE% %TIME%
echo 工作目录    : %CD%
echo.

:: 操作系统信息
echo [操作系统]
systeminfo 2>/dev/null | findstr /B /C:"OS 名称" /C:"OS 版本" /C:"系统类型" /C:"系统启动时间" /C:"处理�?
echo.

:: CPU 信息
echo [处理器]
powershell -NoProfile -Command "Get-CimInstance Win32_Processor | Select-Object -First 1 Name,NumberOfCores,NumberOfLogicalProcessors | Format-List"
echo.

:: 物理内存
echo [物理内存]
powershell -NoProfile -Command "$total=(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory; Write-Host ('总内�? {0:N2} GB' -f ($total/1GB))"
echo.

:: 内存条信�?echo [内存条信息]
powershell -NoProfile -Command "Get-CimInstance Win32_PhysicalMemory | Select-Object Capacity,Speed,Manufacturer,PartNumber | Format-List"
echo.

:: 磁盘分区
echo [磁盘分区]
powershell -NoProfile -Command "Get-CimInstance Win32_LogicalDisk | Where-Object {$_.DriveType -eq 3} | Select-Object DeviceID,VolumeName,@{N='Size(GB)';E={[math]::Round($_.Size/1GB,2)}},@{N='FreeSpace(GB)';E={[math]::Round($_.FreeSpace/1GB,2)}} | Format-Table -AutoSize"
echo.

:: 网络 IP 地址
echo [网络 IP 地址]
ipconfig | findstr "IPv4 地址"
echo.

:: 物理地址
echo [物理地址]
ipconfig /all | findstr "物理地址" | findstr /v "隧道"
echo.

:: 系统运行时间
echo [系统运行时间]
powershell -NoProfile -Command "$boot=(Get-CimInstance Win32_OperatingSystem).LastBootUpTime; Write-Host ('系统启动时间: ' + $boot)"
echo.

echo ============================================================
echo 信息收集完毕�?pause
