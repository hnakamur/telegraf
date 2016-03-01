# ltsv_log Service Input Plugin

The ltsv_log plugin gathers metrics by reading a [LTSV (Labeled Tab-separated Values)](http://ltsv.org/) formatted log file.
It works like the `tail` command and keep reading when more logs are added.
And when the log file is rotated, this plugin read logs until the end from the old file and then read logs from the new file.

### Configuration:

```toml
[[inputs.ltsv_log]]
  # SampleConfig
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
```

### Measurements & Fields:

- measurement of the name specified in the config `measurement` value
- fields specified in the config `int_fields`, `float_fields`, `bool_fields`, and `str_fields` values.

### Tags:

- tags specified in the config `log_tags` values.

### Example Output:

```
[root@localhost telegraf]# sudo -u telegraf ./telegraf -config /etc/telegraf/telegraf.conf -input-filter ltsv_log -debug
2016/03/04 00:56:47 Attempting connection to output: influxdb
2016/03/04 00:56:47 Successfully connected to output: influxdb
2016/03/04 00:56:47 Attempting connection to output: file
2016/03/04 00:56:47 Successfully connected to output: file
2016/03/04 00:56:47 Starting Telegraf (version 0.10.4.1-25-g6cd85e1)
2016/03/04 00:56:47 Loaded outputs: influxdb file
2016/03/04 00:56:47 Loaded inputs: ltsv_log
2016/03/04 00:56:47 Tags enabled: host=localhost.localdomain
2016/03/04 00:56:47 Agent Config: Interval:10s, Debug:true, Quiet:false, Hostname:"localhost.localdomain", Flush Interval:10s
2016/03/04 00:56:47 Started a LTSV log reader, path: /var/log/nginx/access.ltsv.log
> nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200 body_bytes_sent=612i,request_time=0 1457020445000000000
^C2016/03/04 00:56:50 Gathered metrics, (10s interval), from 1 inputs in 12.065µs
2016/03/04 00:56:50 Hang on, flushing any cached metrics before shutdown
nginx_access,host=localhost,http_host=localhost,http_referer=-,http_user_agent=curl/7.29.0,remote_addr=127.0.0.1,remote_user=-,request=GET\ /\ HTTP/1.1,scheme=http,status=200 body_bytes_sent=612i,request_time=0 1457020445000000000
2016/03/04 00:56:50 Wrote 1 metrics to output file in 69.122µs
2016/03/04 00:56:50 Wrote 1 metrics to output influxdb in 2.806848ms
```
