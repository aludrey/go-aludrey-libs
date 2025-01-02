package loggerHooks

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/sirupsen/logrus"
)

const (
	firehoseKeyForEnv        = "env"
	firehoseKeyForAppName    = "app"
	firehoseKeyForTime       = "time"
	firehoseKeyForLevel      = "level"
	firehoseKeyForMessage    = "message"
	firehoseKeyForCaller     = "caller"
	firehoseKeyForStreamName = "stream_name"
	firehoseRequestId        = "request_id"
)

var defaultLevels = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
}

type FirehoseHookConfig struct {
	DefaultStreamName  string
	Env                string
	AppName            string
	AwsRegion          string
	IgnoreFields       map[string]bool
	Filters            map[string]func(interface{}) interface{}
	RequestIdFiledKey  string
	StreamNameFieldKey string
}

// FirehoseHook is logrus hook for AWS Firehose.
// Amazon Kinesis Firehose is a fully-managed service that delivers real-time
// streaming data to destinations such as Amazon Simple Storage Service (Amazon
// S3), Amazon Elasticsearch Service (Amazon ES), and Amazon Redshift.
type FirehoseHook struct {
	client     *firehose.Firehose
	async      bool
	levels     []logrus.Level
	addNewline bool
	config     FirehoseHookConfig
}

// New returns initialized logrus hook for Firehose with persistent Firehose logger.
func NewFireHoseHook(config FirehoseHookConfig) (*FirehoseHook, error) {
	sess := session.Must(session.NewSession())
	svc := firehose.New(sess, aws.NewConfig().WithRegion(config.AwsRegion))
	if config.IgnoreFields == nil {
		config.IgnoreFields = make(map[string]bool)
	}
	if config.Filters == nil {
		config.Filters = make(map[string]func(interface{}) interface{})
	}
	return &FirehoseHook{
		config: config,
		client: svc,
		levels: defaultLevels,
	}, nil
}

// Levels returns logging level to fire this hook.
func (h *FirehoseHook) Levels() []logrus.Level {
	return h.levels
}

// SetLevels sets logging level to fire this hook.
func (h *FirehoseHook) SetLevels(levels []logrus.Level) {
	h.levels = levels
}

// Async sets async flag and send log asynchroniously.
// If use this option, Fire() does not return error.
func (h *FirehoseHook) Async() {
	h.async = true
}

// AddIgnore adds field name to ignore.
func (h *FirehoseHook) AddIgnore(name string) {
	h.config.IgnoreFields[name] = true
}

// AddFilter adds a custom filter function.
func (h *FirehoseHook) AddFilter(name string, fn func(interface{}) interface{}) {
	h.config.Filters[name] = fn
}

// AddNewline sets if a newline is added to each message.
func (h *FirehoseHook) AddNewLine(b bool) {
	h.addNewline = b
}

// Fire is invoked by logrus and sends log to Firehose.
func (h *FirehoseHook) Fire(entry *logrus.Entry) error {
	if !h.async {
		return h.fire(entry)
	}

	// send log asynchroniously and return no error.
	go h.fire(entry)
	return nil
}

// Fire is invoked by logrus and sends log to Firehose.
func (h *FirehoseHook) fire(entry *logrus.Entry) error {
	in := &firehose.PutRecordInput{
		DeliveryStreamName: stringPtr(h.getStreamName(entry)),
		Record: &firehose.Record{
			Data: h.getData(entry),
		},
	}
	_, err := h.client.PutRecord(in)
	return err
}

func (h *FirehoseHook) getStreamName(entry *logrus.Entry) string {
	if name, ok := entry.Data[firehoseKeyForStreamName].(string); ok {
		return name
	}
	return h.config.DefaultStreamName
}

func (h *FirehoseHook) getData(entry *logrus.Entry) []byte {

	data := make(logrus.Fields)

	entry.Data[firehoseKeyForAppName] = h.config.AppName
	entry.Data[firehoseKeyForEnv] = h.config.Env
	entry.Data[firehoseKeyForMessage] = entry.Message
	entry.Data[firehoseKeyForTime] = entry.Time.Local().Format("2006-01-02T15:04:05.000Z")
	entry.Data[firehoseKeyForLevel] = entry.Level.String()

	stack := make([]uintptr, 10)
	length := runtime.Callers(6, stack[:])
	frames := runtime.CallersFrames(stack[:length])

	var stackTrace []string
	for {
		frame, more := frames.Next()
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	//entry.Data["stack"] = stackTrace

	entry.Data[firehoseKeyForCaller] = stackTrace //entry.Caller.Func.Name() + ":" + strconv.Itoa(entry.Caller.Line)

	for k, v := range entry.Data {
		if _, ok := h.config.IgnoreFields[k]; ok {
			continue
		}
		if fn, ok := h.config.Filters[k]; ok {
			v = fn(v) // apply custom filter
		} else {
			v = formatData(v) // use default formatter
		}
		data[k] = v
	}
	data[firehoseRequestId] = entry.Data[h.config.RequestIdFiledKey]

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	if h.addNewline {
		n := []byte("\n")
		bytes = append(bytes, n...)
	}
	return bytes
}

// formatData returns value as a suitable format.
func formatData(value interface{}) (formatted interface{}) {
	switch value := value.(type) {
	case json.Marshaler:
		return value
	case error:
		return value.Error()
	case fmt.Stringer:
		return value.String()
	default:
		return value
	}
}

func stringPtr(str string) *string {
	return &str
}
