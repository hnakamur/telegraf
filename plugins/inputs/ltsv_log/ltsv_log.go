package ltsv_log

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type DupPointModifier interface {
	Modify(t *time.Time, tags map[string]string)
}

type AddTagDupPointModifier struct {
	UniqTagName string
	prevTime    time.Time
	dupCount    int64
}

func (m *AddTagDupPointModifier) Modify(t *time.Time, tags map[string]string) {
	if *t == m.prevTime {
		m.dupCount++
		tags[m.UniqTagName] = strconv.FormatInt(m.dupCount, 10)
	} else {
		m.dupCount = 0
		m.prevTime = *t
	}
}

type IncTimeDupPointModifier struct {
	prevTime time.Time
}

func (m *IncTimeDupPointModifier) Modify(t *time.Time, _ map[string]string) {
	if !t.After(m.prevTime) {
		*t = m.prevTime.Add(time.Nanosecond)
	}
	m.prevTime = *t
}

type NoOpDupPointModifier struct{}

func (n *NoOpDupPointModifier) Modify(_ *time.Time, _ map[string]string) {
}

type ltsvLogReader struct {
	Measurement string
	Path        string
	TimeLabel   string
	TimeFormat  string
	StrFields   []string
	IntFields   []string
	FloatFields []string
	BoolFields  []string
	// NOTE: I tried Tags, but values were not set, so I changed the name to LogTags.
	LogTags                          []string
	WaitMilliseconds                 int
	DuplicatePointsWorkaroundMethod  string
	DuplicatePointsWorkaroundUniqTag string

	sync.Mutex
	done chan struct{}
	acc  telegraf.Accumulator

	fieldSet map[string]string
	tagSet   map[string]bool

	// NOTE: We keep the file open to read the rest after a log rotate.
	file             *os.File
	prevFileSize     int64
	dupPointModifier DupPointModifier
}

var sampleConfig = `
  ## The measurement name
  measurement = "nginx_access"
  ## A LTSV formatted log file path
  ## See http://ltsv.org/ for Labeled Tab-separated Values (LTSV)
  ## Here is an example config for nginx.
  ##
  ##  log_format  ltsv  'time:$time_iso8601\t'
  ##                    'host:$host\t'
  ##                    'http_host:$http_host\t'
  ##                    'scheme:$scheme\t'
  ##                    'remote_addr:$remote_addr\t'
  ##                    'remote_user:$remote_user\t'
  ##                    'time_local:$time_local\t'
  ##                    'request:$request\t'
  ##                    'status:$status\t'
  ##                    'body_bytes_sent:$body_bytes_sent\t'
  ##                    'http_referer:$http_referer\t'
  ##                    'http_user_agent:$http_user_agent\t'
  ##                    'http_x_forwarded_for:$http_x_forwarded_for\t'
  ##                    'request_time:$request_time\t'
  ##                    'upstream_response_time:$upstream_response_time\t'
  ##                    'upstream_http_content_type:$upstream_http_content_type\t'
  ##                    'upstream_status:$upstream_status\t'
  ##                    'upstream_cache_status:$upstream_cache_status';
  ##  access_log  /var/log/nginx/access.ltsv.log  ltsv;
  ##
  path = "/var/log/nginx/access.ltsv.log"
  ## time field label
  time_label = "time"
  ## time value format (See https://golang.org/pkg/time/#Parse)
  time_format = "2006-01-02T15:04:05-07:00"
  ## integer field names
  int_fields = ["body_bytes_sent"]
  ## float field names
  float_fields = ["request_time"]
  ## boolean field names
  bool_fields = []
  ## string field names
  str_fields = []
  ## tag names
  log_tags = ["host", "http_host", "scheme", "remote_addr", "remote_user", "request", "status", "http_referer", "http_user_agent"]
  ## wait milliseconds when there is no log to read
  ## CAUTION: settings this value to 0 leads to high CPU usage
  wait_milliseconds = 10
  ## duplicate points workaround method: add_uniq_tag, increment_time, or none.
  ## See https://docs.influxdata.com/influxdb/v0.10/troubleshooting/frequently_encountered_issues/#writing-duplicate-points
  duplicate_points_workaround_method = "increment_time"
  ## tag name used for ensure uniquness of points
  duplicate_points_workaround_uniq_tag = "uniq"
`

func (r *ltsvLogReader) SampleConfig() string {
	return sampleConfig
}

func (r *ltsvLogReader) Description() string {
	return "Read a log file in LTSV (Labeled Tab-separated Values) format"
}

// Start the ltsv log reader. Caller must call *ltsvLogReader.Stop() to clean up.
func (r *ltsvLogReader) Start(acc telegraf.Accumulator) error {
	r.Lock()
	defer r.Unlock()

	r.acc = acc
	r.done = make(chan struct{})
	r.fieldSet = newFieldSet(r.StrFields, r.IntFields, r.FloatFields, r.BoolFields)
	r.tagSet = newTagSet(r.LogTags)
	switch r.DuplicatePointsWorkaroundMethod {
	case "add_uniq_tag":
		r.dupPointModifier = &AddTagDupPointModifier{UniqTagName: r.DuplicatePointsWorkaroundUniqTag}
	case "increment_time":
		r.dupPointModifier = &IncTimeDupPointModifier{}
	default:
		r.dupPointModifier = &NoOpDupPointModifier{}
	}
	err := r.openLog()
	if err != nil {
		return err
	}

	// Start the log file reader
	go r.receiver()
	log.Printf("Started a LTSV log reader, path: %s\n", r.Path)

	return nil
}

func newFieldSet(strFields, intFields, floatFields, boolFields []string) map[string]string {
	s := make(map[string]string)
	for _, field := range strFields {
		s[field] = "string"
	}
	for _, field := range intFields {
		s[field] = "int"
	}
	for _, field := range floatFields {
		s[field] = "float"
	}
	for _, field := range boolFields {
		s[field] = "boolean"
	}
	return s
}

func newTagSet(names []string) map[string]bool {
	s := make(map[string]bool)
	for _, name := range names {
		s[name] = true
	}
	return s
}

func (r *ltsvLogReader) receiver() {
	defer r.clean()
	for {
		select {
		case <-r.done:
			return
		default:
			n, err := r.read()
			if err != nil {
				log.Printf("error while reading from %s, error: %s\n", r.Path, err.Error())
			}
			if n == 0 {
				time.Sleep(time.Duration(r.WaitMilliseconds) * time.Millisecond)
			}
		}
	}
}

func (r *ltsvLogReader) read() (n int64, err error) {
	// NOTE: We want to check the size of the specified file name, not
	// the opened file. So we use os.Stat(t.path), not t.file.Stat() here.
	info, err := os.Stat(r.Path)
	if err != nil {
		return
	}

	size := info.Size()
	defer r.setPrevFileSize(size)
	var n2 int64
	if size < r.prevFileSize && size > 0 {
		// NOTE: If the log file size is smaller than the previous size and is greater
		// than zero, we assume the log file was rotated and logs are written to the
		// new file.

		// First we read logs from the rotated file.
		n2, err = r.readCurrentFile()
		if err != nil {
			return
		}
		n += n2

		err = r.reopenLog()
		if err != nil {
			return
		}
	}
	n2, err = r.readCurrentFile()
	n += n2
	return
}

func (r *ltsvLogReader) setPrevFileSize(size int64) {
	r.prevFileSize = size
}

func (r *ltsvLogReader) readCurrentFile() (n int64, err error) {
	scanner := bufio.NewScanner(r.file)
	for scanner.Scan() {
		text := scanner.Text()
		n += int64(len(text))
		err = r.processLine(text)
		if err != nil {
			return
		}
	}
	err = scanner.Err()
	return
}

func (r *ltsvLogReader) processLine(line string) error {
	var t time.Time
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	terms := strings.Split(line, "\t")
	for _, term := range terms {
		kv := strings.SplitN(term, ":", 2)
		k := kv[0]
		if k == r.TimeLabel {
			var err error
			t, err = time.Parse(r.TimeFormat, kv[1])
			if err != nil {
				return err
			}
		} else if typ, ok := r.fieldSet[k]; ok {
			switch typ {
			case "string":
				fields[k] = kv[1]
			case "int":
				val, err := strconv.ParseInt(kv[1], 10, 64)
				if err != nil {
					return err
				}
				fields[k] = val
			case "float":
				val, err := strconv.ParseFloat(kv[1], 64)
				if err != nil {
					return err
				}
				fields[k] = val
			case "boolean":
				val, err := strconv.ParseBool(kv[1])
				if err != nil {
					return err
				}
				fields[k] = val
			}
		} else if _, ok := r.tagSet[k]; ok {
			tags[k] = kv[1]
		}
	}
	r.dupPointModifier.Modify(&t, tags)
	r.acc.AddFields(r.Measurement, fields, tags, t)
	return nil
}

func (r *ltsvLogReader) clean() {
	r.Lock()
	defer r.Unlock()
	r.closeLog()
}

func (r *ltsvLogReader) Stop() {
	r.Lock()
	close(r.done)
	r.Unlock()
}

func (r *ltsvLogReader) openLog() error {
	file, err := os.Open(r.Path)
	if err != nil {
		return err
	}
	r.file = file
	return nil
}

func (r *ltsvLogReader) closeLog() error {
	return r.file.Close()
}

func (r *ltsvLogReader) reopenLog() error {
	err := r.closeLog()
	if err != nil {
		return err
	}
	return r.openLog()
}

func (n *ltsvLogReader) Gather(_ telegraf.Accumulator) error {
	return nil
}

func init() {
	inputs.Add("ltsv_log", func() telegraf.Input {
		return &ltsvLogReader{}
	})
}
