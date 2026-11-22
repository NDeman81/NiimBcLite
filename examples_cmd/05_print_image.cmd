@echo off
chcp 65001 >nul
cd /d "%~dp0.."
if not exist "sample_logo.png" niimbc-cli.exe --safe --top "LOGO" --out "sample_logo.png" preview barcode "12345678"
niimbc-cli.exe --length 50 image "sample_logo.png"
if errorlevel 1 echo Print failed! & pause
