package d2util

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
)

// LogLevel determines how verbose the logging is (higher is more verbose)
type LogLevel = int

// Log levels
const (
	LogLevelNone LogLevel = iota
	LogLevelFatal
	LogLevelError
	LogLevelWarning
	LogLevelInfo
	LogLevelDebug
	LogLevelUnspecified
)

// LogLevelDefault is the default log level
const LogLevelDefault = LogLevelInfo

const (
	red     = 1
	green   = 2
	yellow  = 3
	magenta = 5
	cyan    = 6
)

const fmtColorEscape = "\033[3%dm"
const colorEscapeReset = "\033[0m"

// Log format strings for log levels
const (
	fmtPrefix     = "[%s]"
	LogFmtDebug   = "[DEBUG]" + colorEscapeReset + " %s\r\n"
	LogFmtInfo    = "[INFO]" + colorEscapeReset + " %s\r\n"
	LogFmtWarning = "[WARNING]" + colorEscapeReset + " %s\r\n"
	LogFmtError   = "[ERROR]" + colorEscapeReset + " %s\r\n"
	LogFmtFatal   = "[FATAL]" + colorEscapeReset + " %s\r\n"
)

// NewLogger creates a new logger with a default
func NewLogger() *Logger {
	l := &Logger{
		level:        LogLevelDefault,
		colorEnabled: true,
		mutex:        sync.Mutex{},
	}

	l.Writer = log.Writer()

	return l
}

// Logger is used to write log messages, and can have a log level to determine verbosity
type Logger struct {
	prefix string
	io.Writer
	level        LogLevel
	colorEnabled bool
	mutex        sync.Mutex
}

// SetPrefix sets a prefix for the message.
// example:
// 		logger.SetPrefix("XYZ")
// 		logger.Debug("ABC") will print "[XYZ] [DEBUG] ABC"
func (l *Logger) SetPrefix(s string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.prefix = s
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if level == LogLevelUnspecified {
		level = LogLevelDefault
	}

	l.level = level
}

// SetColorEnabled adds color escape-sequences to the logging output
func (l *Logger) SetColorEnabled(b bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if runtime.GOOS == "windows" {
		b = false
	}

	l.colorEnabled = b
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	go l.print(LogLevelInfo, msg)
}

// Infof formats and then logs an info message
func (l *Logger) Infof(fmtMsg string, args ...interface{}) {
	l.Info(fmt.Sprintf(fmtMsg, args...))
}

// Warning logs a warning message
func (l *Logger) Warning(msg string) {
	go l.print(LogLevelWarning, msg)
}

// Warningf formats and then logs a warning message
func (l *Logger) Warningf(fmtMsg string, args ...interface{}) {
	l.Warning(fmt.Sprintf(fmtMsg, args...))
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	go l.print(LogLevelError, msg)
}

// Errorf formats and then logs a error message
func (l *Logger) Errorf(fmtMsg string, args ...interface{}) {
	l.Error(fmt.Sprintf(fmtMsg, args...))
}

// Fatal logs an fatal error message and exits programm
func (l *Logger) Fatal(msg string) {
	go l.print(LogLevelFatal, msg)
	os.Exit(1)
}

// Fatalf formats and then logs a error message
func (l *Logger) Fatalf(fmtMsg string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(fmtMsg, args...))
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	go l.print(LogLevelDebug, msg)
}

// Debugf formats and then logs a debug message
func (l *Logger) Debugf(fmtMsg string, args ...interface{}) {
	l.Debug(fmt.Sprintf(fmtMsg, args...))
}

func (l *Logger) print(level LogLevel, msg string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.level < level {
		return
	}

	colors := map[LogLevel]int{
		LogLevelDebug:   cyan,
		LogLevelInfo:    green,
		LogLevelWarning: yellow,
		LogLevelError:   red,
		LogLevelFatal:   red,
	}

	fmtString := ""

	if l.prefix != "" {
		if l.colorEnabled {
			fmtString = fmt.Sprintf(fmtColorEscape, magenta)
		}

		fmtString += fmt.Sprintf(fmtPrefix, l.prefix)
	}

	if l.colorEnabled {
		fmtString += fmt.Sprintf(fmtColorEscape, colors[level])
	}

	switch level {
	case LogLevelDebug:
		fmtString += LogFmtDebug
	case LogLevelInfo:
		fmtString += LogFmtInfo
	case LogLevelWarning:
		fmtString += LogFmtWarning
	case LogLevelError:
		fmtString += LogFmtError
	case LogLevelFatal:
		fmtString += LogFmtFatal
	case LogLevelNone:
	default:
		return
	}

	_, err := l.Write(format(fmtString, []byte(msg)))
	if err != nil {
		log.Print(err)
	}
}

func format(fmtStr string, fmtInput []byte) []byte {
	return []byte(fmt.Sprintf(fmtStr, string(fmtInput)))
}

func (l *Logger) Write(p []byte) (n int, err error) {
	return l.Writer.Write(p)
}
