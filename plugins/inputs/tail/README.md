# Service Input Plugin: Tail

The tail plugin gathers metrics by reading a log file.
It works like the BSD `tail` command and can keep reading when more logs are added.

### Configuration:

```toml
# Read a log file like the BSD tail command
[[inputs.ltsv_log]]
  ## The measurement name
  name_override = "nginx_access"

  ## A LTSV formatted log file path.
  ## See http://ltsv.org/ for Labeled Tab-separated Values (LTSV)
  ## Here is an example config for nginx (http://nginx.org/en/).
  ##
  ##  log_format  ltsv  'time:$time_iso8601\t'
  ##                    'host:$host\t'
  ##                    'http_host:$http_host\t'
  ##                    'scheme:$scheme\t'
  ##                    'remote_addr:$remote_addr\t'
  ##                    'remote_user:$remote_user\t'
  ##                    'request:$request\t'
  ##                    'status:$status\t'
  ##                    'body_bytes_sent:$body_bytes_sent\t'
  ##                    'http_referer:$http_referer\t'
  ##                    'http_user_agent:$http_user_agent\t'
  ##                    'http_x_forwarded_for:$http_x_forwarded_for\t'
  ##                    'request_time:$request_time';
  ##  access_log  /var/log/nginx/access.ltsv.log  ltsv;
  ##
  filename = "/var/log/nginx/access.ltsv.log"

  ## Seek to this location before tailing
  seek_offset = 0

  ## Seek from whence. See https://golang.org/pkg/os/#File.Seek
  seek_whence = 0

  ## Reopen recreated files (tail -F)
  re_open = true

  ## Fail early if the file does not exist
  must_exist = false

  ## Poll for file changes instead of using inotify
  poll = false

  ## Set this to true if the file is a named pipe (mkfifo)
  pipe = false

  ## Continue looking for new lines (tail -f)
  follow = true

  ## If non-zero, split longer lines into multiple lines
  max_line_size = 0

  ## Set this false to enable logging to stderr, true to disable logging
  disable_logging = false

  ## Data format to consume. Currently only "ltsv" is supported.
  ## Each data format has it's own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "ltsv"

  ## Time label to be used to create a timestamp for a measurement.
  time_label = "time"

  ## Time format for parsing timestamps.
  ## Please see https://golang.org/pkg/time/#Parse for the format string.
  time_format = "2006-01-02T15:04:05Z07:00"

  ## Labels for string fields.
  str_field_labels = []

  ## Labels for integer (64bit signed decimal integer) fields.
  ## For acceptable integer values, please refer to:
  ## https://golang.org/pkg/strconv/#ParseInt
  int_field_labels = ["body_bytes_sent"]

  ## Labels for float (64bit float) fields.
  ## For acceptable float values, please refer to:
  ## https://golang.org/pkg/strconv/#ParseFloat
  float_field_labels = ["request_time"]

  ## Labels for boolean fields.
  ## For acceptable boolean values, please refer to:
  ## https://golang.org/pkg/strconv/#ParseBool
  bool_field_labels = []

  ## Labels for tags to be added
  tag_labels = ["host", "http_host", "scheme", "remote_addr", "remote_user", "request", "status", "http_referer", "http_user_agent", "http_x_forwarded_for"]

  ## Method to modify duplicated measurement points.
  ## Must be one of "add_uniq_tag", "increment_time", "no_op".
  ## This will be used to modify duplicated points.
  ## For detail, please see https://docs.influxdata.com/influxdb/v0.10/troubleshooting/frequently_encountered_issues/#writing-duplicate-points
  ## NOTE: For modifier methods other than "no_op" to work correctly, the log lines
  ## MUST be sorted by timestamps in ascending order.
  duplicate_points_modifier_method = "add_uniq_tag"

  ## When duplicate_points_modifier_method is "increment_time",
  ## this will be added to the time of the previous measurement
  ## if the time of current time is equal to or less than the
  ## time of the previous measurement.
  ##
  ## NOTE: You need to set this value equal to or greater than
  ## precisions of your output plugins. Otherwise the times will
  ## become the same value!
  ## For the precision of the InfluxDB plugin, please see
  ## https://github.com/influxdata/telegraf/blob/v0.10.1/plugins/outputs/influxdb/influxdb.go#L40-L42
  ## For the duration string format, please see
  ## https://golang.org/pkg/time/#ParseDuration
  duplicate_points_increment_duration = "1us"

  ## When duplicate_points_modifier_method is "add_uniq_tag",
  ## this will be the label of the tag to be added to ensure uniqueness of points.
  ## NOTE: The uniq tag will be only added to the successive points of duplicated
  ## points, it will not be added to the first point of duplicated points.
  ## If you want to always add the uniq tag, add a tag with the same name as
  ## duplicate_points_modifier_uniq_tag and the string value "0" to [inputs.tail.tags].
  duplicate_points_modifier_uniq_tag = "uniq"

  ## Defaults tags to be added to measurements.
  [inputs.tail.tags]
    log_host = "log.example.com"
```

### Tail plugin with LTSV parser

#### Measurements & Fields:

- measurement of the name specified in the config `measurement` value
- fields specified in the config `int_field_labels`, `float_field_labels`, `bool_field_labels`, and `str_field_labels` values.

#### Tags:

- tags specified in the config `default_tags`, `duplicate_points_modifier_uniq_tag`, `tag_labels` values.

#### Example Output:

```
[root@localhost bin]# sudo -u telegraf ./telegraf -config /etc/telegraf/telegraf.conf -input-filter tail -debug & for i in `seq 1 3`; do curl -s -o /dev/null localhost; done && sleep 1 && for i in `seq 1 2`; do curl -s -o /dv/null localhost; done
[1] 17652
2016/03/05 17:27:52 Attempting connection to output: influxdb
2016/03/05 17:27:52 Successfully connected to output: influxdb
2016/03/05 17:27:52 Starting Telegraf (version 0.10.4.1-40-gd9189da)
2016/03/05 17:27:52 Loaded outputs: influxdb
2016/03/05 17:27:52 Loaded inputs: tail
2016/03/05 17:27:52 Tags enabled: host=localhost.localdomain
2016/03/05 17:27:52 Agent Config: Interval:5s, Debug:true, Quiet:false, Hostname:"localhost.localdomain", Flush Interval:10s
2016/03/05 17:27:52 Started a tail log reader, filename: /var/log/nginx/access.ltsv.log
2016/03/05 17:27:52 Seeked /var/log/nginx/access.ltsv.log - &{Offset:0 Whence:0}
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,http_x_forwarded_for=-,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200 body_bytes_sent=612i,request_time=0 1457166472000000000
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,http_x_forwarded_for=-,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200,uniq=1 body_bytes_sent=612i,request_time=0 1457166472000000000
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,http_x_forwarded_for=-,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200,uniq=2 body_bytes_sent=612i,request_time=0 1457166472000000000
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,http_x_forwarded_for=-,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200 body_bytes_sent=612i,request_time=0 1457166473000000000
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,http_x_forwarded_for=-,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200,uniq=1 body_bytes_sent=612i,request_time=0 1457166473000000000
```
