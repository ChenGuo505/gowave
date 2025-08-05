package log

import (
	"fmt"
	"time"
)

type TextFormatter struct {
}

func (t *TextFormatter) Format(opt *LoggingOptions) string {
	now := time.Now()
	msgPrompt := "| "
	if opt.Level == LoggerLevelError {
		msgPrompt = "\n Error:"
	}
	if opt.IsColored {
		return fmt.Sprintf("%s[gowave]%s |%s %v %s|%s %s %s%s %v %s",
			cyan, reset,
			blue, now.Format("2006-01-02 15:04:05"), reset,
			opt.Level.Color(), opt.Level.Level(), reset, msgPrompt, opt.Msg, opt.LoggerFields.String(),
		)
	}
	return fmt.Sprintf("[gowave] | %v | %s %s %v %s",
		now.Format("2006-01-02 15:04:05"),
		opt.Level.Level(), msgPrompt, opt.Msg, opt.LoggerFields.String())
}
