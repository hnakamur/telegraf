package ltsv_log

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/influxdata/telegraf/testutil"
)

const sampleLogs = "time:2016-03-03T13:58:57+00:00\thost:localhost\thttp_host:localhost\tscheme:http\tremote_addr:127.0.0.1\tremote_user:-\ttime_local:03/Mar/2016:13:58:57\t+0000\trequest:GET / HTTP/1.1\tstatus:200\tbody_bytes_sent:612\thttp_referer:-\thttp_user_agent:curl/7.29.0\thttp_x_forwarded_for:-\trequest_time:0.000\tupstream_response_time:-\tupstream_http_content_type:-\tupstream_status:-\tupstream_cache_status:-\n"

func TestLtsvLogGeneratesMetrics(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "access.ltsv.log")
	if err != nil {
		t.Fatal("failed to create a temporary file", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(sampleLogs); err != nil {
		t.Fatal("failed to write logs a temporary file", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal("failed to close the temporary log file", err)
	}

	measurement := "nginx_access"
	reader := &ltsvLogReader{
		Measurement:      measurement,
		Path:             tmpfile.Name(),
		TimeLabel:        "time",
		TimeFormat:       "2006-01-02T15:04:05-07:00",
		IntFields:        []string{"body_bytes_sent"},
		FloatFields:      []string{"request_time"},
		BoolFields:       []string{},
		StrFields:        []string{},
		LogTags:          []string{"host", "http_host", "scheme", "remote_addr", "remote_user", "request", "status", "http_referer", "http_user_agent"},
		WaitMilliseconds: 10,
	}
	var acc testutil.Accumulator
	reader.Start(&acc)
	time.Sleep(time.Second)
	reader.Stop()

	fields := map[string]interface{}{
		"body_bytes_sent": int64(612),
		"request_time":    0.0,
	}
	tags := map[string]string{
		"host":            "localhost",
		"http_host":       "localhost",
		"scheme":          "http",
		"remote_addr":     "127.0.0.1",
		"remote_user":     "-",
		"request":         "GET / HTTP/1.1",
		"status":          "200",
		"http_referer":    "-",
		"http_user_agent": "curl/7.29.0",
	}
	acc.AssertContainsTaggedFields(t, measurement, fields, tags)
}
