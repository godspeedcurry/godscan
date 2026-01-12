package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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

// LogLevel represents different log levels (for compatibility with Logger interface)
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

// Logger interface defines logging operations
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warning(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})

	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// ExistingLogger implements Logger interface using existing logging functions
type ExistingLogger struct {
	mu     sync.RWMutex
	fields map[string]interface{}
}

// NewExistingLogger creates a new logger that uses existing logging functions
func NewExistingLogger() Logger {
	return &ExistingLogger{
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *ExistingLogger) Debug(msg string, args ...interface{}) {
	l.logWithFields("DEBUG", msg, args...)
}

// Info logs an info message
func (l *ExistingLogger) Info(msg string, args ...interface{}) {
	l.logWithFields("INFO", msg, args...)
}

// Warning logs a warning message
func (l *ExistingLogger) Warning(msg string, args ...interface{}) {
	l.logWithFields("WARN", msg, args...)
}

// Error logs an error message
func (l *ExistingLogger) Error(msg string, args ...interface{}) {
	l.logWithFields("ERROR", msg, args...)
}

// Fatal logs a fatal message and exits
func (l *ExistingLogger) Fatal(msg string, args ...interface{}) {
	l.logWithFields("FATAL", msg, args...)
}

// SetLevel sets the minimum log level (no-op for compatibility)
func (l *ExistingLogger) SetLevel(level LogLevel) {
	// This would require modifying the underlying logging system
	// For now, it's a no-op to maintain compatibility
}

// SetOutput sets the output destination (no-op for compatibility)
func (l *ExistingLogger) SetOutput(w io.Writer) {
	// This would require modifying the underlying logging system
	// For now, it's a no-op to maintain compatibility
}

// WithField adds a field to the logger context
func (l *ExistingLogger) WithField(key string, value interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &ExistingLogger{
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *ExistingLogger) WithFields(fields map[string]interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &ExistingLogger{
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// logWithFields adds fields to the message and logs it
func (l *ExistingLogger) logWithFields(level string, msg string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Add fields to message if any
	if len(l.fields) > 0 {
		var fieldStrs []string
		for k, v := range l.fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		msgWithFields := fmt.Sprintf("%s [%s]", msg, fmt.Sprintf("%v", fieldStrs))

		// Call existing logging functions
		switch level {
		case "DEBUG":
			Debug(msgWithFields, args...)
		case "INFO":
			Info(msgWithFields, args...)
		case "WARN":
			Warning(msgWithFields, args...)
		case "ERROR":
			Error(msgWithFields, args...)
		case "FATAL":
			Fatal(msgWithFields, args...)
		}
	} else {
		// Call existing logging functions without fields
		switch level {
		case "DEBUG":
			Debug(msg, args...)
		case "INFO":
			Info(msg, args...)
		case "WARN":
			Warning(msg, args...)
		case "ERROR":
			Error(msg, args...)
		case "FATAL":
			Fatal(msg, args...)
		}
	}
}

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

var (
	globalLogFile *os.File
	globalLogMu   sync.Mutex
)

// InitLog opens the log file for writing. It should be called once at startup.
func InitLog(path string) error {
	if path == "" {
		return nil
	}
	globalLogMu.Lock()
	defer globalLogMu.Unlock()

	if globalLogFile != nil {
		globalLogFile.Close()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log dir failed: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}
	globalLogFile = f
	return nil
}

// CloseLog closes the global log file.
func CloseLog() {
	globalLogMu.Lock()
	defer globalLogMu.Unlock()
	if globalLogFile != nil {
		globalLogFile.Close()
		globalLogFile = nil
	}
}

func logRecordOnly(levelText string, msg string) {
	line := formatLine(levelText, msg, false, color.Reset)

	globalLogMu.Lock()
	defer globalLogMu.Unlock()

	if globalLogFile != nil {
		// Use the persistent file handle
		if _, err := globalLogFile.WriteString(line + "\n"); err != nil {
			// Fallback or ignore? If we can't write to log, maybe stderr
			fmt.Fprintf(os.Stderr, "Error writing to log: %v\n", err)
		}
		return
	}

	// Fallback to slow path if InitLog wasn't called (e.g. tests or early errors)
	out := viper.GetString("output")
	if out == "" {
		return
	}
	// Note: We avoid calling FileWrite here to avoid circular dependencies if we move things,
	// but mostly to avoid the other mutex. But simpler to just duplicate the write logic or depend on FileWrite again?
	// FileWrite is in utils/useful.go. logs.go is in utils too.
	// We can't easily call FileWrite if we want to avoid its lock, but FileWrite HAS a lock.
	// So let's just use FileWrite for fallback.
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
		fmt.Printf("{\"time\":\"%s\",\"level\":\"%s\",\"message\":%q}\n", GetCurrentTime(), levelName(level), msg)
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
