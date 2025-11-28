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

func GetCurrentTime() string {
	return time.Now().Format("15:04:05")
}

func shouldLog(level int) bool {
	if level == LevelFatal {
		return true
	}
	return level <= viper.GetInt("loglevel")
}

func logRecord(level int, detail string) {
	if !shouldLog(level) {
		return
	}
	FileWrite(viper.GetString("output"), detail+"\n")
}

func logPrint(level int, detail string) {
	if !shouldLog(level) {
		return
	}
	if viper.GetBool("quiet") {
		return
	}
	if viper.GetBool("json") {
		type jl struct {
			Time    string `json:"time"`
			Level   string `json:"level"`
			Message string `json:"message"`
		}
		fmt.Println(fmt.Sprintf(`{"time":"%s","level":"%s","message":%q}`, GetCurrentTime(), levelName(level), detail))
		return
	}
	fmt.Println(detail)
}

func LogBeautify(x string, colorAttr color.Attribute, y string, level int) {
	logPrint(level, fmt.Sprintf("[%s] [%s] %s", GetCurrentTime(), color.New(colorAttr).Sprintf("%s", x), y))
	logRecord(level, fmt.Sprintf("[%s] [%s] %s", GetCurrentTime(), x, y))
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

func levelName(level int) string {
	switch level {
	case LevelInfo:
		return "INFO"
	case LevelSuccess:
		return "SUCCESS"
	case LevelFailed:
		return "FAILED"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelDebug:
		return "DEBUG"
	default:
		return "INFO"
	}
}
