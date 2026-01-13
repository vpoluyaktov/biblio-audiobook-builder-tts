package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents logging verbosity
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel = LevelInfo
	mu           sync.RWMutex
	output       io.Writer = os.Stderr
	stdLogger    *log.Logger
)

func init() {
	stdLogger = log.New(output, "", log.LstdFlags)
}

// SetLevel sets the global log level
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// SetLevelFromString sets the log level from a string
func SetLevelFromString(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		SetLevel(LevelDebug)
	case "INFO":
		SetLevel(LevelInfo)
	case "WARN", "WARNING":
		SetLevel(LevelWarn)
	case "ERROR":
		SetLevel(LevelError)
	default:
		SetLevel(LevelInfo)
	}
}

// GetLevel returns the current log level
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// SetOutput sets the output destination for the logger
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
	stdLogger = log.New(output, "", log.LstdFlags)
	log.SetOutput(w)
}

// levelPrefix returns the prefix for a given log level
func levelPrefix(level Level) string {
	switch level {
	case LevelDebug:
		return "[DEBUG] "
	case LevelInfo:
		return "[INFO]  "
	case LevelWarn:
		return "[WARN]  "
	case LevelError:
		return "[ERROR] "
	default:
		return ""
	}
}

// shouldLog returns true if the given level should be logged
func shouldLog(level Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= currentLevel
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if shouldLog(LevelDebug) {
		stdLogger.Printf(levelPrefix(LevelDebug)+format, v...)
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if shouldLog(LevelInfo) {
		stdLogger.Printf(levelPrefix(LevelInfo)+format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if shouldLog(LevelWarn) {
		stdLogger.Printf(levelPrefix(LevelWarn)+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if shouldLog(LevelError) {
		stdLogger.Printf(levelPrefix(LevelError)+format, v...)
	}
}

// Fatal logs an error message and exits
func Fatal(format string, v ...interface{}) {
	stdLogger.Printf(levelPrefix(LevelError)+format, v...)
	os.Exit(1)
}

// Fatalf is an alias for Fatal for compatibility
func Fatalf(format string, v ...interface{}) {
	Fatal(format, v...)
}

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("LEVEL(%d)", l)
	}
}
