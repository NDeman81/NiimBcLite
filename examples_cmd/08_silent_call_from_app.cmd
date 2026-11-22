@echo off
chcp 65001 >nul
cd /d "%~dp0.."
rem Тихий вызов для 1С / скриптов: выводит только ошибки и возвращает код 0 при успехе
niimbc-cli.exe --quiet --copies 1 template "TestShablonFullNew" "%~1"
exit /b %ERRORLEVEL%
