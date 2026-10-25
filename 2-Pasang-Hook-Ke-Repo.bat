@echo off
title Pasang Git Hook Otomatis ke Repository Lokal
cd /d "%~dp0"

if not exist "%~dp0git-notif.exe" (
    echo Meng-compile git-notif.exe terlebih dahulu...
    go build -o git-notif.exe main.go
)

echo ============================================================
echo   PEMASANG GIT HOOK TELEGRAM (COMMIT, PULL, MERGE, PUSH)
echo ============================================================
echo.
echo Pilih repository yang ingin dipasang notifikasi otomatis:
echo [1] RTMS_Monitoring (D:\Project Sistem\Project Web Apps\RTMS_Monitoring)
echo [2] Masukkan path folder repository lain secara manual
echo.
set /p PILIHAN="Masukkan pilihan (1 atau 2): "

if "%PILIHAN%"=="1" (
    set "REPO_PATH=D:\Project Sistem\Project Web Apps\RTMS_Monitoring"
) else (
    set /p REPO_PATH="Paste path lengkap folder repository Git Anda: "
)

if not exist "%REPO_PATH%\.git" (
    echo.
    echo [ERROR] Folder "%REPO_PATH%\.git" tidak ditemukan! Pastikan folder tersebut sudah di-git init.
    pause
    exit /b 1
)

set "EXE_PATH=%~dp0git-notif.exe"

powershell -NoProfile -ExecutionPolicy Bypass -Command "$exe = ($env:EXE_PATH -replace '\\', '/'); $hooksDir = Join-Path $env:REPO_PATH '.git\hooks'; [IO.File]::WriteAllText((Join-Path $hooksDir 'post-commit'), \"#!/bin/sh`n`\"$exe`\" commit`n\"); [IO.File]::WriteAllText((Join-Path $hooksDir 'post-merge'), \"#!/bin/sh`n`\"$exe`\" pull`n\"); [IO.File]::WriteAllText((Join-Path $hooksDir 'pre-push'), \"#!/bin/sh`n`\"$exe`\" push`n\");"

echo.
echo [BERHASIL] Git Hook berhasil dipasang di:
echo %REPO_PATH%\.git\hooks
echo.
echo Sekarang setiap kali Anda melakukan git commit, git pull, git merge, atau git push
echo di repository tersebut, notifikasi otomatis masuk ke grup Telegram Anda!
echo.
pause
