package utils

import (
	"fmt"
	"time"

	"github.com/godspeedcurry/godscan/common"

	"github.com/fatih/color"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelSuccess
	LevelFailed
	LevelWarning
	LevelError
	LevelFatal
)

func get_current_time() string {
	return time.Now().Format("15:04:05")
}

func log_print(level int, detail string) {
	if level > common.LogLevel {
		return
	}
	fmt.Println(detail)
}

func Debug(format string, args ...interface{}) {
	color.Cyan("[%s] [%s] %s", get_current_time(), "DEBUG", fmt.Sprintf(format, args...))
}

func Info(format string, args ...interface{}) {
	color.Cyan("[%s] [%s] %s", get_current_time(), "INFO", fmt.Sprintf(format, args...))
}

func Success(format string, args ...interface{}) {
	color.Green("[%s] [%s] %s", get_current_time(), "SUCCESS", fmt.Sprintf(format, args...))
}

func Failed(format string, args ...interface{}) {
	color.Green("[%s] [%s] %s", get_current_time(), "FAILED", fmt.Sprintf(format, args...))
}

func Warning(format string, args ...interface{}) {
	color.Yellow("[%s] [%s] %s", get_current_time(), "WARN", fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	color.Red("[%s] [%s] %s", get_current_time(), "ERROR", fmt.Sprintf(format, args...))
}

func Fatal(format string, args ...interface{}) {
	color.HiRed("[%s] [%s] %s", get_current_time(), "FATAL", fmt.Sprintf(format, args...))
}
