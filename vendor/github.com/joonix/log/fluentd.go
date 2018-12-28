package log

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// FluentdFormatter is similar to logrus.JSONFormatter but with log level that are recongnized
// by kubernetes fluentd.
type FluentdFormatter struct {
	TimestampFormat string
}

// Format the log entry. Implements logrus.Formatter.
func (f *FluentdFormatter) Format(entry *logrus.Entry) ([]byte, error) {
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
	prefixFieldClashes(data)

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339Nano
	}

	data["time"] = entry.Time.Format(timestampFormat)
	data["message"] = entry.Message
	data["severity"] = entry.Level.String()

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

func prefixFieldClashes(data logrus.Fields) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
	}
}
