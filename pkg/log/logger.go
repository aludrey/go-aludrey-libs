package log

type Level uint32
type Fields map[string]interface{}

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

type Logger interface {
	Panic(args ...interface{})
	Fatal(args ...interface{})
	Error(args ...interface{})
	Warn(args ...interface{})
	Info(args ...interface{})
	Debug(args ...interface{})
	Trace(args ...interface{})
	Print(args ...interface{})
	SetLevel(level Level)
	WithFields(keyValues Fields) Logger
	WithRequestId(requestId string) Logger
	WithStreamName(streamName string) Logger
}
