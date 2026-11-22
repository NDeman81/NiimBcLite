# NiimBcLite (`niimbc-cli.exe`)

Быстрая автономная CLI-утилита для одиночной печати этикеток на принтерах **NIIMBOT N1** (без GUI, без веб-сервера и без лишних опросов RFID перед печатью).

## Приоритет типа этикетки
1. Явный флаг `--label <тип>` в командной строке
2. Тип этикетки, указанный в шаблоне (`template.label`)
3. Значение `label` в `config.json`
4. По умолчанию: `withgaps` (без опроса RFID; принудительный опрос RFID — через `--label auto` или команду `info`)

## Основные команды
```bat
niimbc-cli.exe scan                                   :: найти принтер и сохранить адрес в config.json
niimbc-cli.exe info                                   :: принудительно запросить статус, заряд и RFID
niimbc-cli.exe barcode "123498756000099999"           :: напечатать одиночный штрихкод / 2D-код
niimbc-cli.exe text "Строка 1" "Строка 2"             :: напечатать текстовую этикетку
niimbc-cli.exe image logo.png                         :: напечатать изображение (PNG/JPEG/GIF)
niimbc-cli.exe template                               :: вывести список сохранённых шаблонов
niimbc-cli.exe template "ИмяШаблона" "кол1;кол2"      :: напечатать этикетку по шаблону из Конструктора
niimbc-cli.exe preview barcode "12345" --out out.png  :: сохранить PNG-макет без подключения к принтеру
```

Готовые `.cmd`-скрипты находятся в папке `examples_cmd\`.