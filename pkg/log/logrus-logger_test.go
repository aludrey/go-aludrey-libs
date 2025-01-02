package log

import (
	"testing"

	loggerHooks "github.com/aludrey/go-aludrey-libs/pkg/log/hooks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var logger Logger = NewLogger(LogrusLoggerConfig{
	Level:        InfoLevel,
	ReportCaller: true,
	FirehoseHook: &loggerHooks.FirehoseHookConfig{
		DefaultStreamName: "aludrey-dev-us-e2-logs-apps",
		Env:               "dev",
		AppName:           "library-test",
		AwsRegion:         "us-east-2",
	},
})

func TestLogPrint(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)
	logger.Print("Test Log Print Level")
	assert.Equal(t, logrus.GetLevel(), logrus.InfoLevel)
}

func TestLogDebug(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logger.Debug("Test Log Debug Level")
	assert.Equal(t, logrus.GetLevel(), logrus.DebugLevel)
}

func TestLogInfo(t *testing.T) {
	logrus.SetLevel(logrus.InfoLevel)
	logger.Info("Test Log Info Level")
	assert.Equal(t, logrus.GetLevel(), logrus.InfoLevel)
}

func TestLogWarn(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger.Warn("Test log Warn Level")
	assert.Equal(t, logrus.GetLevel(), logrus.WarnLevel)
}

func TestLogError(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)
	logger.Error("Test Log Error Level")
	assert.Equal(t, logrus.GetLevel(), logrus.ErrorLevel)
}

func TestLogPanic(t *testing.T) {
	logrus.SetLevel(logrus.PanicLevel)
	assert.Panics(t, func() {
		logger.Panic("Test Log Panic Level")
	})
	assert.Equal(t, logrus.GetLevel(), logrus.PanicLevel)
}

func TestLogFatal(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	level := logrus.GetLevel()
	assert.True(t, level == logrus.FatalLevel)

}

func TestLogTrace(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	logger.Trace("Test Log Trace Level")
	assert.Equal(t, logrus.GetLevel(), logrus.TraceLevel)
}

func TestWithFields(t *testing.T) {
	fields := Fields{
		"testFieldKey": "very important value",
	}
	nl := logger.WithFields(fields)
	nl.Info("Test Log With Fields")
}

func TestLoggerSegregation(t *testing.T) {
	// I'm an http controller
	// I get a request and I generate a request id
	requestId := "123"
	logger1 := logger.WithRequestId(requestId)
	//I then receive another request and generate another request id
	requestId = "456"
	logger2 := logger.WithRequestId(requestId)
	// I then log some info
	logger1.Info("I'm an http controller loggin some info. The expected output should have a request id of 123")
	logger2.Info("I'm an http controller loggin some info. The expected output should have a request id of 456")
	//I then pass the logger to the service
	logger1.WithFields(Fields{
		"field": "calculatedfieldforrequest123",
	})
	logger1.Debug("I'm the sevice loggin some info. The expected output should have a request id of 123 and field: calculatedfieldforrequest123")
	logger2.Debug("I'm the sevice loggin some info. The expected output should have a request id of 456 and no field")

	logger.Info("I'm the main logger, i should not have a requestId nor custom fields")
	// I encountered a security issue, logging it with the corresponding stream name
	logger.WithStreamName("aludrey-dev-us-e2-logs-sec").Error("I'm logging a security issue with a custom stream name")
	//I could also pass logger.WithStreamName("aludrey-dev-us-e2-logs-sec") to a specific service, to isolate it from the responsability of picking the correct stream
	// the default log is not affected
	logger.Info("I'm an application log")

}
