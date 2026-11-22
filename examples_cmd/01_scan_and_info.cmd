@echo off
chcp 65001 >nul
cd /d "%~dp0.."
echo === Searching for NIIMBOT N1 and saving config.json ===
niimbc-cli.exe scan
echo.
echo === Querying printer status and RFID ===
niimbc-cli.exe info
pause
