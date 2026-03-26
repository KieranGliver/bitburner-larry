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
	Summary string
	Detail  string
	Time    time.Time
}

// LogMsg is a tea message — send via p.Send(logger.Info("..."))
type LogMsg struct{ Entry LogEntry }

func NewLog(level Level, summary string) LogMsg {
	return LogMsg{Entry: LogEntry{Level: level, Summary: summary, Time: time.Now()}}
}

func NewLogDetail(level Level, summary, detail string) LogMsg {
	return LogMsg{Entry: LogEntry{Level: level, Summary: summary, Detail: detail, Time: time.Now()}}
}

func Info(summary string) LogMsg  { return NewLog(INFO, summary) }
func Warn(summary string) LogMsg  { return NewLog(WARN, summary) }
func Error(summary string) LogMsg { return NewLog(ERROR, summary) }

func InfoDetail(summary, detail string) LogMsg  { return NewLogDetail(INFO, summary, detail) }
func WarnDetail(summary, detail string) LogMsg  { return NewLogDetail(WARN, summary, detail) }
func ErrorDetail(summary, detail string) LogMsg { return NewLogDetail(ERROR, summary, detail) }
