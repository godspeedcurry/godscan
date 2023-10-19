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

func LogBeautify(x string, colorAttr color.Attribute, y string) {
	fmt.Printf("[%s] [%s] %s\n", get_current_time(), color.New(colorAttr).Sprintf("%s", x), y)
}

func Debug(format string, args ...interface{}) {
	LogBeautify("DEBUG", color.BgYellow, fmt.Sprintf(format, args...))
}

func Info(format string, args ...interface{}) {
	LogBeautify("INFO", color.FgCyan, fmt.Sprintf(format, args...))
}

func Success(format string, args ...interface{}) {
	LogBeautify("SUCCESS", color.FgHiRed, fmt.Sprintf(format, args...))
}

func Failed(format string, args ...interface{}) {
	LogBeautify("FAILED", color.BgRed, fmt.Sprintf(format, args...))
}

func Warning(format string, args ...interface{}) {
	LogBeautify("WARN", color.FgYellow, fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	LogBeautify("ERROR", color.FgRed, fmt.Sprintf(format, args...))
}

func Fatal(format string, args ...interface{}) {
	LogBeautify("FATAL", color.FgHiRed, fmt.Sprintf(format, args...))
}
