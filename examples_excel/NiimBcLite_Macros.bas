Attribute VB_Name = "NiimBcLite_Macros"
Option Explicit

Private Const CLI_EXE_PATH As String = "C:\dev\NiimBcLite\niimbc-cli.exe"
Private Const DEFAULT_TEMPLATE As String = "TestShablonFullNew"
Private Const DEFAULT_LENGTH_MM As Double = 50

Private Function RunNiimCliUTF8(ByVal cliArgs As String, Optional ByRef outText As String = "") As Long
    Dim fso As Object, sh As Object, stm As Object
    Dim tmpCmd As String, tmpLog As String, binData() As Byte

    Set fso = CreateObject("Scripting.FileSystemObject")
    Set sh = CreateObject("WScript.Shell")

    tmpCmd = fso.GetSpecialFolder(2) & "\niim_run_" & Format(Now, "hhnnss") & "_" & Int(Rnd * 10000) & ".cmd"
    tmpLog = tmpCmd & ".log"

    Set stm = CreateObject("ADODB.Stream")
    stm.Type = 2
    stm.Charset = "utf-8"
    stm.Open
    stm.WriteText "@echo off" & vbCrLf
    stm.WriteText "chcp 65001 >nul" & vbCrLf
    stm.WriteText """" & CLI_EXE_PATH & """ " & cliArgs & " > """ & tmpLog & """ 2>&1" & vbCrLf
    stm.WriteText "exit /b %ERRORLEVEL%" & vbCrLf

    stm.Position = 0
    stm.Type = 1
    stm.Position = 3
    binData = stm.Read
    stm.Close

    Set stm = CreateObject("ADODB.Stream")
    stm.Type = 1
    stm.Open
    stm.Write binData
    stm.SaveToFile tmpCmd, 2
    stm.Close

    RunNiimCliUTF8 = sh.Run("cmd.exe /c """ & tmpCmd & """", 0, True)

    If fso.FileExists(tmpLog) Then
        Set stm = CreateObject("ADODB.Stream")
        stm.Type = 2
        stm.Charset = "utf-8"
        stm.Open
        stm.LoadFromFile tmpLog
        outText = stm.ReadText
        stm.Close
        On Error Resume Next
        fso.DeleteFile tmpLog, True
        On Error GoTo 0
    End If

    On Error Resume Next
    fso.DeleteFile tmpCmd, True
    On Error GoTo 0
End Function

Private Function EscArg(ByVal s As String) As String
    EscArg = """" & Replace(Trim(s), """", "'") & """"
End Function

Private Function ConvertPngToBmp(ByVal pngPath As String, ByVal bmpPath As String) As Boolean
    On Error GoTo ErrHandler
    Dim img As Object, ip As Object
    Set img = CreateObject("WIA.ImageFile")
    Set ip = CreateObject("WIA.ImageProcess")
    img.LoadFile pngPath
    ip.Filters.Add ip.FilterInfos("Convert").FilterID
    ip.Filters(1).Properties("FormatID").Value = "{B96B3CAB-0728-11D3-9D7B-0000F81EF32E}"
    Set img = ip.Apply(img)
    If Dir(bmpPath) <> "" Then Kill bmpPath
    img.SaveFile bmpPath
    ConvertPngToBmp = True
    Exit Function
ErrHandler:
    ConvertPngToBmp = False
End Function

Public Function Niim_GetTemplateNames() As Variant
    Dim outText As String, rc As Long
    Dim lines() As String, line As String, p1 As Long, p2 As Long, tName As String
    Dim res() As String, cnt As Long, i As Long

    rc = RunNiimCliUTF8("template", outText)
    If rc <> 0 Or Trim(outText) = "" Then
        Niim_GetTemplateNames = Array()
        Exit Function
    End If

    lines = Split(Replace(outText, vbCrLf, vbLf), vbLf)
    cnt = 0
    For i = LBound(lines) To UBound(lines)
        line = Trim(lines(i))
        p1 = InStr(line, ". ")
        p2 = InStrRev(line, " (length:")
        If p1 > 0 And p2 > p1 + 2 Then
            tName = Trim(Mid(line, p1 + 2, p2 - (p1 + 2)))
            If tName <> "" Then
                cnt = cnt + 1
                ReDim Preserve res(1 To cnt)
                res(cnt) = tName
            End If
        End If
    Next i

    If cnt = 0 Then
        Niim_GetTemplateNames = Array()
    Else
        Niim_GetTemplateNames = res
    End If
End Function

' 1. Заполнение указанного столбца таблицы списком шаблонов
Public Sub Niim_FillColumnWithTemplates(ByVal ws As Worksheet, ByVal colLetter As String, Optional ByVal startRow As Long = 2)
    Dim names As Variant, i As Long, lastRow As Long
    names = Niim_GetTemplateNames()
    lastRow = ws.Cells(ws.Rows.Count, colLetter).End(xlUp).Row
    If lastRow >= startRow Then
        ws.Range(ws.Cells(startRow, colLetter), ws.Cells(lastRow, colLetter)).ClearContents
    End If
    If IsEmpty(names) Or UBound(names) < LBound(names) Then Exit Sub
    For i = LBound(names) To UBound(names)
        ws.Cells(startRow + (i - LBound(names)), colLetter).Value = names(i)
    Next i
End Sub

Public Sub Demo_FillColumnE_WithTemplates()
    Niim_FillColumnWithTemplates ActiveSheet, "E", 2
End Sub

' 2. Заполнение раскрывающегося списка (ComboBox / ListBox) на UserForm
Public Sub Niim_FillComboWithTemplates(ByVal ctrl As Object)
    Dim names As Variant, i As Long
    ctrl.Clear
    names = Niim_GetTemplateNames()
    If IsEmpty(names) Or UBound(names) < LBound(names) Then Exit Sub
    For i = LBound(names) To UBound(names)
        ctrl.AddItem names(i)
    Next i
    If ctrl.ListCount > 0 Then ctrl.ListIndex = 0
End Sub

' 3. Формирование изображения штрихкода и вставка в указанную ячейку
Public Sub Niim_InsertBarcodeToCell(ByVal targetCell As Range, ByVal barcodeData As String, _
                                    Optional ByVal bcFormat As String = "auto", _
                                    Optional ByVal topCaption As String = "", _
                                    Optional ByVal lengthMM As Double = 50)
    Dim fso As Object, pngPath As String, cmd As String, logOut As String, rc As Long
    Dim shp As Shape, oldShp As Shape, shpName As String
    Dim scaleW As Double, scaleH As Double, ratio As Double

    If Trim(barcodeData) = "" Then Exit Sub

    Set fso = CreateObject("Scripting.FileSystemObject")
    pngPath = fso.GetSpecialFolder(2) & "\niim_cell_" & targetCell.Address(False, False) & ".png"

    cmd = "--safe --length " & Replace(CStr(lengthMM), ",", ".") & " --format " & EscArg(bcFormat)
    If Trim(topCaption) <> "" Then cmd = cmd & " --top " & EscArg(topCaption)
    cmd = cmd & " --out " & EscArg(pngPath) & " preview barcode " & EscArg(barcodeData)

    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Or Not fso.FileExists(pngPath) Then
        MsgBox "Ошибка генерации штрихкода:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
        Exit Sub
    End If

    shpName = "NiimBC_" & targetCell.Worksheet.Name & "_" & targetCell.Address(False, False)
    On Error Resume Next
    For Each oldShp In targetCell.Worksheet.Shapes
        If oldShp.Name = shpName Then oldShp.Delete
    Next oldShp
    On Error GoTo 0

    Set shp = targetCell.Worksheet.Shapes.AddPicture( _
        Filename:=pngPath, LinkToFile:=msoFalse, SaveWithDocument:=msoTrue, _
        Left:=targetCell.Left + 2, Top:=targetCell.Top + 2, Width:=-1, Height:=-1)
    shp.Name = shpName

    scaleW = (targetCell.Width - 4) / shp.Width
    scaleH = (targetCell.Height - 4) / shp.Height
    ratio = IIf(scaleW < scaleH, scaleW, scaleH)
    If ratio > 0 Then
        shp.Width = shp.Width * ratio
        shp.Height = shp.Height * ratio
    End If
    shp.Left = targetCell.Left + (targetCell.Width - shp.Width) / 2
    shp.Top = targetCell.Top + (targetCell.Height - shp.Height) / 2

    On Error Resume Next
    fso.DeleteFile pngPath, True
    On Error GoTo 0
End Sub

Public Sub Demo_InsertBarcodeNextCell()
    If Trim(CStr(ActiveCell.Value)) = "" Then
        MsgBox "Выделите ячейку с кодом!", vbExclamation, "NiimBcLite"
        Exit Sub
    End If
    Niim_InsertBarcodeToCell ActiveCell.Offset(0, 1), CStr(ActiveCell.Value), "auto", ""
End Sub

' 4. Формирование изображения штрихкода и вставка в элемент Image на UserForm
Public Sub Niim_ShowBarcodeOnForm(ByVal imgControl As Object, ByVal barcodeData As String, _
                                  Optional ByVal bcFormat As String = "auto", _
                                  Optional ByVal topCaption As String = "", _
                                  Optional ByVal lengthMM As Double = 50)
    Dim fso As Object, tmpDir As String, pngPath As String, bmpPath As String
    Dim cmd As String, logOut As String, rc As Long

    If Trim(barcodeData) = "" Then
        Set imgControl.Picture = Nothing
        Exit Sub
    End If

    Set fso = CreateObject("Scripting.FileSystemObject")
    tmpDir = fso.GetSpecialFolder(2)
    pngPath = tmpDir & "\niim_form_preview.png"
    bmpPath = tmpDir & "\niim_form_preview.bmp"

    cmd = "--safe --length " & Replace(CStr(lengthMM), ",", ".") & " --format " & EscArg(bcFormat)
    If Trim(topCaption) <> "" Then cmd = cmd & " --top " & EscArg(topCaption)
    cmd = cmd & " --out " & EscArg(pngPath) & " preview barcode " & EscArg(barcodeData)

    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Or Not fso.FileExists(pngPath) Then
        MsgBox "Ошибка формирования превью:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
        Exit Sub
    End If

    If ConvertPngToBmp(pngPath, bmpPath) Then
        imgControl.PictureSizeMode = 3
        Set imgControl.Picture = LoadPicture(bmpPath)
    Else
        MsgBox "Не удалось конвертировать PNG в BMP через WIA.", vbExclamation, "NiimBcLite"
    End If

    On Error Resume Next
    fso.DeleteFile pngPath, True
    fso.DeleteFile bmpPath, True
    On Error GoTo 0
End Sub

' 5. Передача данных для печати из полей UserForm
Public Function Niim_PrintFromForm(ByVal templateName As String, ByVal copies As Long, ParamArray fieldValues() As Variant) As Boolean
    Dim i As Long, cols() As String, rowData As String, cmd As String, logOut As String, rc As Long

    If Trim(templateName) = "" Then
        MsgBox "Выберите шаблон этикетки!", vbExclamation, "NiimBcLite"
        Exit Function
    End If
    If copies < 1 Then copies = 1

    If UBound(fieldValues) >= LBound(fieldValues) Then
        ReDim cols(LBound(fieldValues) To UBound(fieldValues))
        For i = LBound(fieldValues) To UBound(fieldValues)
            cols(i) = Replace(Trim(CStr(fieldValues(i))), ";", ",")
        Next i
        rowData = Join(cols, ";")
    End If

    cmd = "--quiet --copies " & CStr(copies) & " template " & EscArg(templateName)
    If rowData <> "" Then cmd = cmd & " " & EscArg(rowData)

    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then
        MsgBox "Ошибка печати с формы:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
        Niim_PrintFromForm = False
    Else
        Niim_PrintFromForm = True
    End If
End Function

' Базовые макросы для работы напрямую со строками и ячейками листа
Public Sub Niim_PrintActiveCellBarcode()
    Dim codeVal As String, topVal As String, cmd As String, logOut As String, rc As Long
    codeVal = Trim(CStr(ActiveCell.Value))
    If codeVal = "" Then
        MsgBox "Выделите ячейку со штрихкодом!", vbExclamation, "NiimBcLite"
        Exit Sub
    End If
    topVal = Trim(CStr(ActiveCell.Offset(0, 1).Value))

    cmd = "--quiet --safe --length " & Replace(CStr(DEFAULT_LENGTH_MM), ",", ".")
    If topVal <> "" Then cmd = cmd & " --top " & EscArg(topVal)
    cmd = cmd & " barcode " & EscArg(codeVal)

    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then MsgBox "Ошибка печати:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
End Sub

Public Sub Niim_PrintActiveRowTemplate()
    Dim r As Long, lastCol As Long, c As Long
    Dim cols() As String, rowData As String, cmd As String, logOut As String, rc As Long

    r = ActiveCell.Row
    lastCol = ActiveSheet.Cells(r, ActiveSheet.Columns.Count).End(xlToLeft).Column
    If lastCol < 1 Then Exit Sub

    ReDim cols(1 To lastCol)
    For c = 1 To lastCol
        cols(c) = Replace(Trim(CStr(ActiveSheet.Cells(r, c).Value)), ";", ",")
    Next c
    rowData = Join(cols, ";")
    If Replace(rowData, ";", "") = "" Then
        MsgBox "Текущая строка пуста!", vbExclamation, "NiimBcLite"
        Exit Sub
    End If

    cmd = "--quiet template " & EscArg(DEFAULT_TEMPLATE) & " " & EscArg(rowData)
    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then MsgBox "Ошибка печати по шаблону:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
End Sub

Public Sub Niim_CheckPrinterInfo()
    Dim logOut As String, rc As Long
    rc = RunNiimCliUTF8("info", logOut)
    MsgBox logOut, IIf(rc = 0, vbInformation, vbCritical), "NIIMBOT N1 Info"
End Sub
