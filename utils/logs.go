package utils

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

const (
	LevelInfo = iota
	LevelSuccess
	LevelFailed
	LevelWarning
	LevelError
	LevelFatal
	LevelDebug
)

func get_current_time() string {
	return time.Now().Format("15:04:05")
}

func log_record(level int, detail string) {
	if level > viper.GetInt("loglevel") {
		return
	}
	FileWrite("result.log", detail+"\n")
}

func log_print(level int, detail string) {
	if level > viper.GetInt("loglevel") {
		return
	}
	fmt.Println(detail)
}

func LogBeautify(x string, colorAttr color.Attribute, y string, level int) {
	log_print(level, fmt.Sprintf("[%s] [%s] %s", get_current_time(), color.New(colorAttr).Sprintf("%s", x), y))
	log_record(level, fmt.Sprintf("[%s] [%s] %s", get_current_time(), x, y))
}

func Debug(format string, args ...interface{}) {
	LogBeautify("DEBUG", color.BgYellow, fmt.Sprintf(format, args...), LevelDebug)
}

func Info(format string, args ...interface{}) {
	LogBeautify("INFO", color.FgCyan, fmt.Sprintf(format, args...), LevelInfo)
}

func Success(format string, args ...interface{}) {
	LogBeautify("SUCCESS", color.FgHiGreen, fmt.Sprintf(format, args...), LevelSuccess)
}

func Failed(format string, args ...interface{}) {
	LogBeautify("FAILED", color.BgRed, fmt.Sprintf(format, args...), LevelFailed)
}

func Warning(format string, args ...interface{}) {
	LogBeautify("WARN", color.FgYellow, fmt.Sprintf(format, args...), LevelWarning)
}

func Error(format string, args ...interface{}) {
	LogBeautify("ERROR", color.FgRed, fmt.Sprintf(format, args...), LevelError)
}

func Fatal(format string, args ...interface{}) {
	LogBeautify("FATAL", color.FgHiRed, fmt.Sprintf(format, args...), LevelFatal)
}
