@echo off
chcp 65001 >nul
cd /d "%~dp0.."
echo === Available templates ===
niimbc-cli.exe template
echo.
echo === Printing single label using template ===
niimbc-cli.exe template "BlackLabel" "123498756000099999;Позиция 1;Склад №2"
if errorlevel 1 echo Print failed! & pause
