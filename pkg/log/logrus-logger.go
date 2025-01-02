package log

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	loggerHooks "github.com/aludrey/go-aludrey-libs/pkg/log/hooks"
	"github.com/sirupsen/logrus"
)

const (
	requestIdKey       = "request_id"
	streamNameFieldKey = "stream_name"
)

type LogrusLoggerConfig struct {
	Level        Level
	LogFile      string
	ReportCaller bool
	FirehoseHook *loggerHooks.FirehoseHookConfig
}

type LogrusLogger struct {
	logrusInstance *logrus.Logger
	fields         Fields
	requestId      string
	streamName     string
}

func setFileOutput(logger *logrus.Logger, logFile string) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}
}

func NewLogger(config LogrusLoggerConfig) Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.Level(config.Level))
	logger.SetReportCaller(config.ReportCaller)
	logger.SetFormatter(&logrus.JSONFormatter{})
	setFileOutput(logger, config.LogFile)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:        "02-01-2006 15:04:05", // the "time" field configuration
		FullTimestamp:          true,
		DisableLevelTruncation: true, // log level field configuration
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// this function is required when you want to introduce your custom format.
			// In my case I wanted file and line to look like this `file="engine.go:141`
			// but f.File provides a full path along with the file name.
			// So in `formatFilePath()` function I just trimmed everything before the file name
			// and added a line number in the end
			return "", formatFilePath(f.File) + fmt.Sprint(f.Line)
		},
	})
	if config.FirehoseHook != nil {
		config.FirehoseHook.RequestIdFiledKey = requestIdKey
		hook, err := loggerHooks.NewFireHoseHook(*config.FirehoseHook)
		if err != nil {
			logger.Info("Failed to create firehose hook")
		} else {
			logger.AddHook(hook)
		}
	}

	return &LogrusLogger{
		fields:         nil,
		logrusInstance: logger,
	}
}
func (p *LogrusLogger) AddHook(hook logrus.Hook) {
	p.logrusInstance.AddHook(hook)
}
func (p *LogrusLogger) getEntry() *logrus.Entry {
	entry := p.logrusInstance.WithContext(context.Background())
	if p.fields != nil {
		entry = entry.WithFields(logrus.Fields(p.fields))
		p.fields = nil
	}
	entry = entry.WithField(requestIdKey, p.requestId)
	if p.streamName != "" {
		entry = entry.WithField(streamNameFieldKey, p.streamName)
	}
	return entry
}
func (p *LogrusLogger) Panic(args ...interface{}) {
	p.getEntry().Panic(args...)
}
func (p *LogrusLogger) Fatal(args ...interface{}) {
	p.getEntry().Fatal(args...)
}
func (p *LogrusLogger) Error(args ...interface{}) {
	p.getEntry().Error(args...)
}
func (p *LogrusLogger) Warn(args ...interface{}) {
	p.getEntry().Warn(args...)
}
func (p *LogrusLogger) Info(args ...interface{}) {
	p.getEntry().Info(args...)
}
func (p *LogrusLogger) Debug(args ...interface{}) {
	p.getEntry().Debug(args...)
}
func (p *LogrusLogger) Trace(args ...interface{}) {
	p.getEntry().Trace(args...)
}
func (p *LogrusLogger) Print(args ...interface{}) {
	p.getEntry().Print(args...)
}
func (p *LogrusLogger) SetLevel(level Level) {
	logrus.SetLevel(logrus.Level(level))
}
func (p *LogrusLogger) WithFields(fields Fields) Logger {
	return &LogrusLogger{
		fields:         fields,
		logrusInstance: p.logrusInstance,
		requestId:      p.requestId,
		streamName:     p.streamName,
	}
}
func (p *LogrusLogger) AddField(key string, value interface{}) Logger {
	fields := p.fields
	if fields == nil {
		fields = Fields{}
	}
	fields[key] = value
	return p.WithFields(fields)
}
func (p *LogrusLogger) WithRequestId(requestId string) Logger {
	return &LogrusLogger{
		logrusInstance: p.logrusInstance,
		fields:         p.fields,
		requestId:      requestId,
		streamName:     p.streamName,
	}
}
func (p *LogrusLogger) WithStreamName(streamName string) Logger {
	return &LogrusLogger{
		logrusInstance: p.logrusInstance,
		fields:         p.fields,
		requestId:      p.requestId,
		streamName:     streamName,
	}
}
func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}
