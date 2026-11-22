@echo off
chcp 65001 >nul
cd /d "%~dp0.."
rem Пример настройки шрифта, размера, жирности, отступов и зазоров для одиночного штрихкода:
niimbc-cli.exe --length 50 --format code128 --safe --font-name "Times New Roman" --font-size "22" --bold --margin 6 --side-margin 14 --gap 4 --text-gap 3 --offset 0 --top "Верхний заголовок" --bot "Нижняя подпись" barcode "123498756000099999"
if errorlevel 1 echo Print failed! & pause
