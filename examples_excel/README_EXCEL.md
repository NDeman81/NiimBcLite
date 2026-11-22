# Интеграция `NiimBcLite` (`niimbc-cli.exe`) с Microsoft Excel (VBA и UserForm)

Модуль **`NiimBcLite_Macros.bas`** поддерживает работу как с ячейками листа Excel, так и с пользовательскими формами (`UserForm`).

## Подключение модуля
1. Откройте книгу Excel и нажмите **`Alt + F11`**.
2. Выберите **`File` -> `Import File...`** и укажите **`NiimBcLite_Macros.bas`**.
3. Проверьте константу `CLI_EXE_PATH` в начале модуля (`C:\dev\NiimBcLite\niimbc-cli.exe`).

---

## Примеры использования 5 функций

### 1. Заполнение столбца таблицы списком шаблонов
```vb
' Заполнить столбец E начиная со 2-й строки на активном листе:
Niim_FillColumnWithTemplates ActiveSheet, "E", 2
```
*(Также доступна готовая кнопка-макрос `Demo_FillColumnE_WithTemplates`.)*

### 2. Заполнение раскрывающегося списка (`ComboBox`) на форме (`UserForm`)
```vb
Private Sub UserForm_Initialize()
    Niim_FillComboWithTemplates Me.cmbTemplate
End Sub
```

### 3. Вставка изображения штрихкода в указанную ячейку таблицы
```vb
' Вставить штрихкод в ячейку D2 (с пропорциональным вписыванием в границы ячейки):
Niim_InsertBarcodeToCell ActiveSheet.Range("D2"), ActiveSheet.Range("A2").Value, "code128", ActiveSheet.Range("B2").Value
```
*(Также доступна готовая кнопка-макрос `Demo_InsertBarcodeNextCell`, вставляющая штрихкод из активной ячейки в ячейку справа.)*

### 4. Вывод изображения штрихкода в элемент `Image` на форме (`UserForm`)
```vb
Private Sub txtBarcode_Change()
    Niim_ShowBarcodeOnForm Me.imgPreview, Me.txtBarcode.Value, "auto", Me.txtCaption.Value
End Sub
```
*(PNG прозрачно конвертируется в BMP через встроенный компонент Windows WIA и масштабируется внутри `imgPreview`.)*

### 5. Печать этикетки по нажатию кнопки на форме (`UserForm`)
```vb
Private Sub btnPrint_Click()
    If Niim_PrintFromForm(Me.cmbTemplate.Value, CLng(Me.txtCopies.Value), Me.txtBarcode.Value, Me.txtTitle.Value, Me.txtLocation.Value) Then
        MsgBox "Этикетка отправлена на печать!", vbInformation
    End If
End Sub
```

### 4б. Предпросмотр полного шаблона в элемент `Image` на форме (`UserForm`)
```vb
Private Sub btnPreview_Click()
    Niim_ShowTemplateOnForm Me.imgPreview, Me.cmbTemplate.Value, Me.txtField1.Value, Me.txtField2.Value, Me.txtField3.Value
End Sub
```

### 6. Печать одиночного штрихкода из Excel с настройкой шрифта, отступов и зазоров
```vb
Public Sub PrintCustomStyledBarcode()
    Dim cmd As String, logOut As String, rc As Long
    cmd = "--quiet --safe --length 50 --format code128 " & _
          "--font-name ""Times New Roman"" --font-size ""22"" --bold " & _
          "--margin 6 --side-margin 14 --gap 4 --text-gap 3 " & _
          "--top " & EscArg(ActiveCell.Offset(0, 1).Value) & " " & _
          "--bot " & EscArg(ActiveCell.Offset(0, 2).Value) & " " & _
          "barcode " & EscArg(ActiveCell.Value)
    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then MsgBox "Ошибка печати: " & logOut, vbCritical, "NiimBcLite"
End Sub
```
