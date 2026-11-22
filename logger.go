package main

import (
"fmt"
"os"
"strings"
"sync"
"time"
)

// LogLevel задаёт уровень детализации вывода в консоль.
type LogLevel int

const (
LogError LogLevel = iota // только ошибки, предупреждения и финальный итог
LogInfo                  // основные этапы работы с метками времени (как в C#-проекте)
LogDebug                 // полный протокольный дамп всех байтов BLE с миллисекундами
)

var (
logMu           sync.Mutex
currentLogLevel = LogInfo
)

// ParseLogLevel преобразует текстовое имя уровня в LogLevel.
func ParseLogLevel(s string) LogLevel {
switch strings.ToLower(strings.TrimSpace(s)) {
case "error", "err", "quiet", "min", "minimal":
return LogError
case "debug", "trace", "full", "verbose":
return LogDebug
default:
return LogInfo
}
}

// SetLogLevel устанавливает текущий уровень логирования.
func SetLogLevel(s string) {
logMu.Lock()
defer logMu.Unlock()
currentLogLevel = ParseLogLevel(s)
}

// GetLogLevel возвращает текущий уровень логирования.
func GetLogLevel() LogLevel {
logMu.Lock()
defer logMu.Unlock()
return currentLogLevel
}

func logStamp() string {
return time.Now().Format("2006-01-02 15:04:05.000")
}

// LogKey выводит важные итоговые события (отображаются на всех уровнях, включая error).
func LogKey(format string, args ...any) {
logMu.Lock()
defer logMu.Unlock()
fmt.Printf("[%s] [OK]  %s\n", logStamp(), fmt.Sprintf(format, args...))
}

// LogInfoMsg выводит основные шаги работы (уровни info и debug).
func LogInfoMsg(format string, args ...any) {
if GetLogLevel() < LogInfo {
return
}
logMu.Lock()
defer logMu.Unlock()
fmt.Printf("[%s] [INF] %s\n", logStamp(), fmt.Sprintf(format, args...))
}

// LogDebugMsg выводит подробный протокольный обмен BLE (только уровень debug).
func LogDebugMsg(format string, args ...any) {
if GetLogLevel() < LogDebug {
return
}
logMu.Lock()
defer logMu.Unlock()
fmt.Printf("[%s] [DBG] %s\n", logStamp(), fmt.Sprintf(format, args...))
}

// LogWarn выводит предупреждения в stderr (отображаются на всех уровнях).
func LogWarn(format string, args ...any) {
logMu.Lock()
defer logMu.Unlock()
fmt.Fprintf(os.Stderr, "[%s] [WRN] %s\n", logStamp(), fmt.Sprintf(format, args...))
}

// LogErr выводит ошибки в stderr (отображаются на всех уровнях).
func LogErr(format string, args ...any) {
logMu.Lock()
defer logMu.Unlock()
fmt.Fprintf(os.Stderr, "[%s] [ERR] %s\n", logStamp(), fmt.Sprintf(format, args...))
}