package rtmpclient

type LogLevel int

const (
	LOG_LEVEL_TRACE LogLevel = iota + 1
	LOG_LEVEL_DEBUG
	LOG_LEVEL_WARNING
	LOG_LEVEL_FATAL
)

// Logging interface that can be implemented by users of this package.
type Logger interface {
	ModulePrintf(level LogLevel, format string, v ...interface{})
	ModulePrintln(level LogLevel, v ...interface{})
}

var logger Logger = nullLogger{}

// SetLogger sets the logger to be used by this package.
func SetLogger(l Logger) {
	logger = l
}

type nullLogger struct {
}

func (l nullLogger) ModulePrintf(level LogLevel, format string, v ...interface{}) {
}

func (l nullLogger) ModulePrintln(level LogLevel, v ...interface{}) {
}
