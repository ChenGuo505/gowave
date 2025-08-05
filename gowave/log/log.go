package log

import (
	"fmt"
	"github.com/ChenGuo505/gowave/internal/gwstrings"
	"io"
	"log"
	"os"
	"path"
	"strings"
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

const defaultLogFileSize = 100 << 20 // 100MB

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
		return ""
	}
	result := "| {"
	for k, v := range f {
		result += fmt.Sprintf("%s: %v, ", k, v)
	}
	result = result[:len(result)-2] + "}" // Remove the last comma and space
	return result
}

type Logger struct {
	Level        LoggerLevel
	Outs         []*logWriter
	Formatter    LoggingFormatter
	LoggerFields LoggerFields
	LogPath      string
	LogFileSize  int64 // Size in bytes, used for log rotation
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

type logWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

func NewLogger() *Logger {
	return &Logger{}
}

func DefaultLogger() *Logger {
	logger := NewLogger()
	logger.Level = LoggerLevelDebug
	w := &logWriter{
		Level: LoggerLevelDebug,
		Out:   os.Stdout,
	}
	logger.Outs = []*logWriter{w}
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

func (l *Logger) SetLogPath(logPath string) {
	l.LogPath = logPath
	l.Outs = append(l.Outs, &logWriter{
		Level: -1,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("all.", time.Now().UnixMilli(), ".log"))),
	})
	l.Outs = append(l.Outs, &logWriter{
		Level: LoggerLevelDebug,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("debug.", time.Now().UnixMilli(), ".log"))),
	})
	l.Outs = append(l.Outs, &logWriter{
		Level: LoggerLevelInfo,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("info.", time.Now().UnixMilli(), ".log"))),
	})
	l.Outs = append(l.Outs, &logWriter{
		Level: LoggerLevelWarn,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("warn.", time.Now().UnixMilli(), ".log"))),
	})
	l.Outs = append(l.Outs, &logWriter{
		Level: LoggerLevelError,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("error.", time.Now().UnixMilli(), ".log"))),
	})
	l.Outs = append(l.Outs, &logWriter{
		Level: LoggerLevelFatal,
		Out:   fileWriter(path.Join(l.LogPath, gwstrings.JoinStrings("fatal.", time.Now().UnixMilli(), ".log"))),
	})
}

func (l *Logger) checkFileSize(w *logWriter) {
	logFile := w.Out.(*os.File)
	if logFile != nil {
		stat, err := logFile.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		size := stat.Size()
		if l.LogFileSize <= 0 {
			l.LogFileSize = defaultLogFileSize
		}
		if size >= l.LogFileSize {
			_, oldName := path.Split(stat.Name())
			newName := gwstrings.JoinStrings(oldName[:strings.Index(oldName, ".")], ".", time.Now().UnixMilli(), ".log")
			out := fileWriter(path.Join(l.LogPath, newName))
			w.Out = out
		}
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
		if out.Out == os.Stdout {
			opt.IsColored = true
			msgStr := l.Formatter.Format(opt)
			_, _ = fmt.Fprintln(out.Out, msgStr)
		} else {
			opt.IsColored = false
			msgStr := l.Formatter.Format(opt)
			if out.Level == -1 || out.Level == level {
				l.checkFileSize(out)
				_, _ = fmt.Fprintln(out.Out, msgStr)
			}
		}
	}
}

func fileWriter(name string) io.Writer {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return file
}
