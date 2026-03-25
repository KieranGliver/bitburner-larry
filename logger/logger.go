package logger

import "time"

type Level uint

const (
	INFO Level = iota
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case WARN:
		return "WARN "
	case ERROR:
		return "ERROR"
	default:
		return "INFO "
	}
}

type LogEntry struct {
	Level   Level
	Message string
	Time    time.Time
}

// LogMsg is a tea message — send via p.Send(logger.Info("..."))
type LogMsg struct{ Entry LogEntry }

func NewLog(level Level, msg string) LogMsg {
	return LogMsg{Entry: LogEntry{Level: level, Message: msg, Time: time.Now()}}
}

func Info(msg string) LogMsg  { return NewLog(INFO, msg) }
func Warn(msg string) LogMsg  { return NewLog(WARN, msg) }
func Error(msg string) LogMsg { return NewLog(ERROR, msg) }
