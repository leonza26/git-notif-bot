@echo off
title Build & Test Bot Git Notif (Golang)
cd /d "%~dp0"

echo [1/2] Meng-compile main.go menjadi git-notif.exe...
go build -o git-notif.exe main.go

if %ERRORLEVEL% NEQ 0 (
    echo [GAGAL] Pastikan Go sudah terinstall di laptop Anda.
    pause
    exit /b 1
)

echo [2/2] Mengirim pesan TEST ke grup Telegram Anda...
echo.
"%~dp0git-notif.exe" test

echo.
pause
