package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Format configuration of the logrus formatter output.
type Format func(*Formatter) error

// DefaultFormat is Stackdriver.
var DefaultFormat = StackdriverFormat

// Formatter that is called on by logrus.
type Formatter struct {
	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat func(logrus.Fields, time.Time) error

	// SeverityMap allows for customizing the names for keys of the log level field.
	SeverityMap map[string]string

	// PrettyPrint will indent all json logs
	PrettyPrint bool
}

// NewFormatter with optional options. Defaults to the Stackdriver option.
func NewFormatter(opts ...Format) *Formatter {
	f := Formatter{}
	if len(opts) == 0 {
		opts = append(opts, DefaultFormat)
	}
	for _, apply := range opts {
		if err := apply(&f); err != nil {
			panic(err)
		}
	}

	return &f
}

// Format the log entry. Implements logrus.Formatter.
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(logrus.Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/Sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}
	prefixFieldClashes(data, entry.HasCaller())

	if !f.DisableTimestamp && f.TimestampFormat != nil {
		// https://cloud.google.com/logging/docs/agent/configuration#timestamp-processing
		if err := f.TimestampFormat(data, entry.Time); err != nil {
			return nil, err
		}
	}

	if entry.Message != "" {
		data["message"] = entry.Message
	}

	if s, ok := f.SeverityMap[entry.Level.String()]; ok {
		data["severity"] = s
	} else {
		data["severity"] = f.SeverityMap["debug"]
	}

	if entry.HasCaller() {
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		if funcVal != "" {
			data[logrus.FieldKeyFunc] = funcVal
		}
		if fileVal != "" {
			data[logrus.FieldKeyFile] = fileVal
		}
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return b.Bytes(), nil
}

func prefixFieldClashes(data logrus.Fields, reportCaller bool) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
		delete(data, "time")
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
		delete(data, "msg")
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
		delete(data, "level")
	}

	if m, ok := data["message"]; ok {
		data["fields.message"] = m
		delete(data, "message")
	}

	if l, ok := data["timestamp"]; ok {
		data["fields.timestamp"] = l
		delete(data, "timestamp")
	}

	if l, ok := data["severity"]; ok {
		data["fields.severity"] = l
		delete(data, "severity")
	}

	if reportCaller {
		if l, ok := data[logrus.FieldKeyFunc]; ok {
			data["fields."+logrus.FieldKeyFunc] = l
		}
		if l, ok := data[logrus.FieldKeyFile]; ok {
			data["fields."+logrus.FieldKeyFile] = l
		}
	}
}

func PrettyPrintFormat(f *Formatter) error {
	f.PrettyPrint = true
	return nil
}

func DisableTimestampFormat(f *Formatter) error {
	f.DisableTimestamp = true
	return nil
}
