package log

import (
	"encoding/json"
	"time"
)

type JsonFormatter struct {
}

func (j *JsonFormatter) Format(opt *LoggingOptions) string {
	now := time.Now()
	jsonLog := JsonLog{
		Level:        opt.Level.Level(),
		Timestamp:    now.Format("2006-01-02 15:04:05"),
		Message:      opt.Msg,
		LoggerFields: opt.LoggerFields,
	}
	jsonLogStr, err := json.Marshal(jsonLog)
	if err != nil {
		return `{"error": "failed to marshal log message"}`
	}
	return string(jsonLogStr)
}

type JsonLog struct {
	Level        string       `json:"level"`
	Timestamp    string       `json:"timestamp"`
	Message      any          `json:"message"`
	LoggerFields LoggerFields `json:"fields,omitempty"`
}
