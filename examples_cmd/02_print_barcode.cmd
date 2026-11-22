@echo off
chcp 65001 >nul
cd /d "%~dp0.."
niimbc-cli.exe --length 50 --safe --top "Склад №1" --bot "Инв. номер" barcode "123498756000099999"
if errorlevel 1 echo Print failed! & pause
