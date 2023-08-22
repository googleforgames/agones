package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes"
	durpb "github.com/golang/protobuf/ptypes/duration"
	"github.com/sirupsen/logrus"
	logtypepb "google.golang.org/genproto/googleapis/logging/type"
)

// StackdriverFormat maps values to be recognized by the Google Cloud Platform.
// https://cloud.google.com/logging/docs/agent/configuration#special-fields
func StackdriverFormat(f *Formatter) error {
	f.SeverityMap = map[string]string{
		"panic":   logtypepb.LogSeverity_CRITICAL.String(),
		"fatal":   logtypepb.LogSeverity_CRITICAL.String(),
		"warning": logtypepb.LogSeverity_WARNING.String(),
		"debug":   logtypepb.LogSeverity_DEBUG.String(),
		"error":   logtypepb.LogSeverity_ERROR.String(),
		"trace":   logtypepb.LogSeverity_DEBUG.String(),
		"info":    logtypepb.LogSeverity_INFO.String(),
	}
	f.TimestampFormat = func(fields logrus.Fields, now time.Time) error {
		// https://cloud.google.com/logging/docs/agent/configuration#timestamp-processing
		ts, err := ptypes.TimestampProto(now)
		if err != nil {
			return err
		}
		fields["timestamp"] = ts
		return nil
	}

	return nil
}

// HTTPRequest contains an http.Request as well as additional
// information about the request and its response.
// https://github.com/googleapis/google-cloud-go/blob/v0.39.0/logging/logging.go#L617
type HTTPRequest struct {
	// Request is the http.Request passed to the handler.
	Request *http.Request

	// RequestSize is the size of the HTTP request message in bytes, including
	// the request headers and the request body.
	RequestSize int64

	// Status is the response code indicating the status of the response.
	// Examples: 200, 404.
	Status int

	// ResponseSize is the size of the HTTP response message sent back to the client, in bytes,
	// including the response headers and the response body.
	ResponseSize int64

	// Latency is the request processing latency on the server, from the time the request was
	// received until the response was sent.
	Latency time.Duration

	// LocalIP is the IP address (IPv4 or IPv6) of the origin server that the request
	// was sent to.
	LocalIP string

	// RemoteIP is the IP address (IPv4 or IPv6) of the client that issued the
	// HTTP request. Examples: "192.168.1.1", "FE80::0202:B3FF:FE1E:8329".
	RemoteIP string

	// CacheHit reports whether an entity was served from cache (with or without
	// validation).
	CacheHit bool

	// CacheValidatedWithOriginServer reports whether the response was
	// validated with the origin server before being served from cache. This
	// field is only meaningful if CacheHit is true.
	CacheValidatedWithOriginServer bool
}

func (r HTTPRequest) MarshalJSON() ([]byte, error) {
	if r.Request == nil {
		return nil, nil
	}
	u := *r.Request.URL
	u.Fragment = ""
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
	e := &logEntry{
		RequestMethod:                  r.Request.Method,
		RequestURL:                     fixUTF8(u.String()),
		Status:                         r.Status,
		UserAgent:                      r.Request.UserAgent(),
		ServerIP:                       r.LocalIP,
		RemoteIP:                       r.RemoteIP,
		Referer:                        r.Request.Referer(),
		CacheHit:                       r.CacheHit,
		CacheValidatedWithOriginServer: r.CacheValidatedWithOriginServer,
	}
	if r.RequestSize > 0 {
		e.RequestSize = fmt.Sprintf("%d", r.RequestSize)
	}
	if r.ResponseSize > 0 {
		e.ResponseSize = fmt.Sprintf("%d", r.ResponseSize)
	}
	if r.Latency != 0 {
		e.Latency = ptypes.DurationProto(r.Latency)
	}

	return json.Marshal(e)
}

type logEntry struct {
	RequestMethod                  string          `json:"requestMethod,omitempty"`
	RequestURL                     string          `json:"requestUrl,omitempty"`
	RequestSize                    string          `json:"requestSize,omitempty"`
	Status                         int             `json:"status,omitempty"`
	ResponseSize                   string          `json:"responseSize,omitempty"`
	UserAgent                      string          `json:"userAgent,omitempty"`
	RemoteIP                       string          `json:"remoteIp,omitempty"`
	ServerIP                       string          `json:"serverIp,omitempty"`
	Referer                        string          `json:"referer,omitempty"`
	Latency                        *durpb.Duration `json:"latency,omitempty"`
	CacheLookup                    bool            `json:"cacheLookup,omitempty"`
	CacheHit                       bool            `json:"cacheHit,omitempty"`
	CacheValidatedWithOriginServer bool            `json:"cacheValidatedWithOriginServer,omitempty"`
	CacheFillBytes                 string          `json:"cacheFillBytes,omitempty"`
	Protocol                       string          `json:"protocol,omitempty"`
}

// fixUTF8 is a helper that fixes an invalid UTF-8 string by replacing
// invalid UTF-8 runes with the Unicode replacement character (U+FFFD).
// See Issue https://github.com/googleapis/google-cloud-go/issues/1383.
func fixUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Otherwise time to build the sequence.
	buf := new(bytes.Buffer)
	buf.Grow(len(s))
	for _, r := range s {
		if utf8.ValidRune(r) {
			buf.WriteRune(r)
		} else {
			buf.WriteRune('\uFFFD')
		}
	}
	return buf.String()
}
