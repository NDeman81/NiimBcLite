@echo off
chcp 65001 >nul
cd /d "%~dp0.."
rem Раскомментируйте нужный формат для печати:
niimbc-cli.exe --format ean13 --safe --top "Товар EAN-13" barcode "460700465001"
rem niimbc-cli.exe --format code39 --safe --top "Code 39" barcode "INV-2026"
rem niimbc-cli.exe --format pdf417 --top "PDF417" barcode "NIIMBOT-N1-2026"
rem niimbc-cli.exe --format qrl --top "QR слева" --bot "Склад 2" barcode "https://example.local"
rem niimbc-cli.exe --format dml --top "DataMatrix" barcode "DM-00112233"
if errorlevel 1 echo Print failed! & pause
