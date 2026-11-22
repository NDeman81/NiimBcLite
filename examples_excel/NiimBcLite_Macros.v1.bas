Attribute VB_Name = "NiimBcLite_Macros"
Option Explicit

' ============================================================================
' Настройки подключения к niimbc-cli.exe
' ============================================================================
Private Const CLI_EXE_PATH As String = "C:\dev\NiimBcLite\niimbc-cli.exe"
Private Const DEFAULT_TEMPLATE As String = "TestShablonFullNew"
Private Const DEFAULT_LENGTH_MM As Double = 50

' Вспомогательная функция: запуск niimbc-cli.exe через временный UTF-8 .cmd
' (гарантирует 100% сохранность кириллицы и ожидание завершения печати)
Private Function RunNiimCliUTF8(ByVal cliArgs As String, Optional ByRef outText As String = "") As Long
    Dim fso As Object, sh As Object, stm As Object
    Dim tmpCmd As String, tmpLog As String, binData() As Byte

    Set fso = CreateObject("Scripting.FileSystemObject")
    Set sh = CreateObject("WScript.Shell")

    tmpCmd = fso.GetSpecialFolder(2) & "\niim_run_" & Format(Now, "hhnnss") & "_" & Int(Rnd * 10000) & ".cmd"
    tmpLog = tmpCmd & ".log"

    ' Записываем .cmd в чистом UTF-8 без BOM с переключением кодовой страницы 65001
    Set stm = CreateObject("ADODB.Stream")
    stm.Type = 2
    stm.Charset = "utf-8"
    stm.Open
    stm.WriteText "@echo off" & vbCrLf
    stm.WriteText "chcp 65001 >nul" & vbCrLf
    stm.WriteText """" & CLI_EXE_PATH & """ " & cliArgs & " > """ & tmpLog & """ 2>&1" & vbCrLf
    stm.WriteText "exit /b %ERRORLEVEL%" & vbCrLf

    stm.Position = 3 ' пропускаем 3 байта UTF-8 BOM
    stm.Type = 1
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

' ============================================================================
' 1. Печать штрихкода из активной ячейки (с подписью из соседней ячейки справа)
' ============================================================================
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
    If rc <> 0 Then
        MsgBox "Ошибка печати:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
    End If
End Sub

' ============================================================================
' 2. Печать активной строки по шаблону из Конструктора (столбцы A;B;C...)
' ============================================================================
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
    If rc <> 0 Then
        MsgBox "Ошибка печати по шаблону:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
    End If
End Sub

' ============================================================================
' 3. Печать 2–3 ячеек активной строки как текстовой этикетки
' ============================================================================
Public Sub Niim_PrintActiveRowText()
    Dim r As Long, s1 As String, s2 As String, cmd As String, logOut As String, rc As Long
    r = ActiveCell.Row
    s1 = Trim(CStr(ActiveSheet.Cells(r, 1).Value))
    s2 = Trim(CStr(ActiveSheet.Cells(r, 2).Value))
    If s1 = "" And s2 = "" Then
        MsgBox "Заполните ячейки A и B в текущей строке!", vbExclamation, "NiimBcLite"
        Exit Sub
    End If

    cmd = "--quiet --length " & Replace(CStr(DEFAULT_LENGTH_MM), ",", ".") & " text " & EscArg(s1)
    If s2 <> "" Then cmd = cmd & " " & EscArg(s2)

    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then
        MsgBox "Ошибка текстовой печати:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
    End If
End Sub

' ============================================================================
' 4. Предпросмотр активной строки в PNG без расхода ленты
' ============================================================================
Public Sub Niim_PreviewActiveRow()
    Dim r As Long, lastCol As Long, c As Long
    Dim cols() As String, rowData As String, pngPath As String, cmd As String, logOut As String, rc As Long

    r = ActiveCell.Row
    lastCol = ActiveSheet.Cells(r, ActiveSheet.Columns.Count).End(xlToLeft).Column
    If lastCol < 1 Then Exit Sub

    ReDim cols(1 To lastCol)
    For c = 1 To lastCol
        cols(c) = Replace(Trim(CStr(ActiveSheet.Cells(r, c).Value)), ";", ",")
    Next c
    rowData = Join(cols, ";")

    pngPath = CreateObject("Scripting.FileSystemObject").GetSpecialFolder(2) & "\niim_preview.png"
    cmd = "--out " & EscArg(pngPath) & " preview template " & EscArg(DEFAULT_TEMPLATE) & " " & EscArg(rowData)
    rc = RunNiimCliUTF8(cmd, logOut)
    If rc <> 0 Then
        MsgBox "Ошибка предпросмотра:" & vbCrLf & logOut, vbCritical, "NiimBcLite"
    Else
        CreateObject("WScript.Shell").Run """" & pngPath & """", 1, False
    End If
End Sub

' ============================================================================
' 5. Принудительный запрос информации о принтере и остатке RFID
' ============================================================================
Public Sub Niim_CheckPrinterInfo()
    Dim logOut As String, rc As Long
    rc = RunNiimCliUTF8("info", logOut)
    MsgBox logOut, IIf(rc = 0, vbInformation, vbCritical), "Состояние NIIMBOT N1"
End Sub