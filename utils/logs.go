package utils

import (
	"fmt"
	"main/common"
	"time"

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
	return time.Now().Format("2006-01-02 15:04:05")
}

func log_print(level int, detail string) {
	if level > common.LogLevel {
		return
	}
	fmt.Println(detail)
}

func Debug(format string, args ...interface{}) {
	color.White("[%s] [%s] %s", get_current_time(), "Warning", fmt.Sprintf(format, args...))
}

func Info(format string, args ...interface{}) {
	color.Blue("[%s] [%s] %s", get_current_time(), "Info", fmt.Sprintf(format, args...))
}

func Success(format string, args ...interface{}) {
	color.Green("[%s] [%s] %s", get_current_time(), "Info", fmt.Sprintf(format, args...))
}

func Failed(format string, args ...interface{}) {
	color.Green("[%s] [%s] %s", get_current_time(), "Failed", fmt.Sprintf(format, args...))
}

func Warning(format string, args ...interface{}) {
	color.Yellow("[%s] [%s] %s", get_current_time(), "Warning", fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	color.Red("[%s] [%s] %s", get_current_time(), "Error", fmt.Sprintf(format, args...))
}

func Fatal(format string, args ...interface{}) {
	color.HiRed("[%s] [%s] %s", get_current_time(), "FATAL", fmt.Sprintf(format, args...))
}
