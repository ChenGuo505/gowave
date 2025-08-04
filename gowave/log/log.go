package log

import (
	"fmt"
	"io"
	"os"
	"time"
)

const (
	redBg   = "\033[97;41m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	reset   = "\033[0m"
)

const (
	LoggerLevelDebug LoggerLevel = iota
	LoggerLevelInfo
	LoggerLevelWarn
	LoggerLevelError
	LoggerLevelFatal
)

type LoggerLevel int

type LoggerFields map[string]any

func (l LoggerLevel) Level() string {
	switch l {
	case LoggerLevelDebug:
		return "DEBUG"
	case LoggerLevelInfo:
		return "INFO"
	case LoggerLevelWarn:
		return "WARN"
	case LoggerLevelError:
		return "ERROR"
	case LoggerLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (l LoggerLevel) Color() string {
	switch l {
	case LoggerLevelDebug:
		return magenta
	case LoggerLevelInfo:
		return green
	case LoggerLevelWarn:
		return yellow
	case LoggerLevelError:
		return red
	case LoggerLevelFatal:
		return redBg
	default:
		return ""
	}
}

func (f LoggerFields) String() string {
	if len(f) == 0 {
		return "{}"
	}
	result := "{"
	for k, v := range f {
		result += fmt.Sprintf("%s: %v, ", k, v)
	}
	result = result[:len(result)-2] + "}" // Remove the last comma and space
	return result
}

type Logger struct {
	Level        LoggerLevel
	Outs         []io.Writer
	Formatter    LoggingFormatter
	LoggerFields LoggerFields
}

type LoggingFormatter interface {
	Format(opt *LoggingOptions) string
}

type LoggingOptions struct {
	Level        LoggerLevel
	IsColored    bool
	LoggerFields LoggerFields

	Msg any
}

type LoggerFormatter struct {
	Level        LoggerLevel
	IsColored    bool
	LoggerFields LoggerFields
}

func NewLogger() *Logger {
	return &Logger{}
}

func DefaultLogger() *Logger {
	logger := NewLogger()
	logger.Level = LoggerLevelDebug
	logger.Outs = []io.Writer{os.Stdout}
	logger.Formatter = &TextFormatter{}
	return logger
}

func (l *Logger) Debug(msg any) {
	l.print(LoggerLevelDebug, msg)
}

func (l *Logger) Info(msg any) {
	l.print(LoggerLevelInfo, msg)
}

func (l *Logger) Warn(msg any) {
	l.print(LoggerLevelWarn, msg)
}

func (l *Logger) Error(msg any) {
	l.print(LoggerLevelError, msg)
}

func (l *Logger) Fatal(msg any) {
	l.print(LoggerLevelFatal, msg)
	os.Exit(1)
}

func (l *Logger) WithFields(fields LoggerFields) *Logger {
	return &Logger{
		Level:        l.Level,
		Outs:         l.Outs,
		Formatter:    l.Formatter,
		LoggerFields: fields,
	}
}

func (l *Logger) print(level LoggerLevel, msg any) {
	if l.Level > level {
		return
	}
	fields := l.LoggerFields
	if fields == nil {
		fields = make(LoggerFields)
	}
	opt := &LoggingOptions{
		Level:        level,
		LoggerFields: fields,
		Msg:          msg,
	}
	for _, out := range l.Outs {
		if out == os.Stdout {
			opt.IsColored = true
		}
		msgStr := l.Formatter.Format(opt)
		_, _ = fmt.Fprintln(out, msgStr)
	}
}

func (f *LoggerFormatter) Format(msg any) string {
	now := time.Now()
	if f.IsColored {
		return fmt.Sprintf("%s[gowave]%s |%s %v %s|%s %s %s| msg: %#v | fields: %v",
			cyan, reset,
			blue, now.Format("2006-01-02 15:04:05"), reset,
			f.Level.Color(), f.Level.Level(), reset, msg, f.LoggerFields,
		)
	}
	return fmt.Sprintf("[gowave] | %v | %s | msg: %#v | fields: %v",
		now.Format("2006-01-02 15:04:05"),
		f.Level.Level(), msg, f.LoggerFields)
}
