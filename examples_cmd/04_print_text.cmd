@echo off
chcp 65001 >nul
cd /d "%~dp0.."
niimbc-cli.exe --length 40 --density 2 text "Насос циркуляционный" "Артикул: НЦ-25/60"
if errorlevel 1 echo Print failed! & pause
