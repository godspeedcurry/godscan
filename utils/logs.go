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

func formatLine(levelText string, msg string, colorize bool, colorAttr color.Attribute) string {
	if colorize {
		return fmt.Sprintf("[%s] %s", color.New(colorAttr).Sprintf("%s", levelText), msg)
	}
	return fmt.Sprintf("[%s] %s", levelText, msg)
}

func logRecordOnly(levelText string, msg string) {
	line := formatLine(levelText, msg, false, color.Reset)
	out := viper.GetString("output")
	if out == "" {
		return
	}
	FileWrite(out, "%s\n", line)
}

func logRecord(level int, levelText string, msg string) {
	if !shouldLog(level) {
		return
	}
	logRecordOnly(levelText, msg)
}

func logPrint(level int, levelText string, msg string, colorAttr color.Attribute) {
	if !shouldLog(level) {
		return
	}
	if viper.GetBool("quiet") {
		return
	}
	if viper.GetBool("json") {
		fmt.Println(fmt.Sprintf(`{"time":"%s","level":"%s","message":%q}`, GetCurrentTime(), levelName(level), msg))
		return
	}
	fmt.Println(formatLine(levelText, msg, true, colorAttr))
}

func LogBeautify(levelText string, colorAttr color.Attribute, msg string, level int) {
	logPrint(level, levelText, msg, colorAttr)
	logRecord(level, levelText, msg)
}

// InfoFile writes INFO lines to file only (no console), respecting loglevel.
func InfoFile(format string, args ...interface{}) {
	if !shouldLog(LevelInfo) {
		return
	}
	logRecordOnly("INFO", fmt.Sprintf(format, args...))
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
