@echo off
chcp 65001 >nul
cd /d "%~dp0.."
niimbc-cli.exe --safe --top "Предпросмотр" --out "preview_check.png" preview barcode "123498756000099999"
if exist "preview_check.png" start "" "preview_check.png"
