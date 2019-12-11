# Google Cloud Platform Logs

If you write logs in the correct format, Google Cloud's [Stackdriver Logging](https://cloud.google.com/logging/docs/basic-concepts) will understand the timestamps, severity levels, collect structured logs, and report stack traces in the error report. This package contains some code to make this easier, as well as some experiments I used to reverse engineer it. The code is in Go, but the formatting things are mostly language independent.

This also contains `gcpzap`, a wrapper for the zap logging library which configures it so Stackdriver understands the logs.


## Logging tips

* Write JSON logs, one payload per line, to stdout or stderr.
* Use `message` for the message, `severity` for the severity level, and one of the time formats. See the [Stackdriver documentation about message formats](https://cloud.google.com/logging/docs/agent/configuration#process-payload) for details.
* Export logs to BigQuery to be able to search.


## Timestamps

The [Stackdriver documentation about time fields](https://cloud.google.com/logging/docs/agent/configuration#timestamp-processing) used to be incorrect, but has been updated. App Engine, Cloud Functions, Cloud Run and Kubernetes seems to support the following formats:

* `timestamp` as a struct: `"timestamp":{"seconds":1551023890,"nanos":858987654}}`
* `timestampSeconds` and `timestampNanos`: `"timestampSeconds":1551023890,"timestampNanos":862987654`
* `time` as a RFC3339/ISO8601 string: `"time":"2019-02-24T15:58:10.864987654Z"}`

The documentation used to state it supported time as Unix seconds dot nanoseconds ("SSSS.NNNNNNNNN"). That format did not work, either in a JSON string or a JSON float. The demo still includes these formats to verify that they do not work.


## Collapsed Logs and Trace IDs

The Google Cloud HTTP load balancer attaches `X-Cloud-Trace-Context` headers to incoming requests. [The format is `X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE`](https://cloud.googler.com/trace/docs/toubleshooting#force-trace). If you include the trace ID in the right format, Stackdriver will parse it. For now, this seems to only useful for querying logs, and for collecting logs together in App Engine (see below).


## Stack Traces/Errors

If you write out a panic, it will get reported in the Stackdriver error reporter. It must either look like a "default" panic, or the panic caught by the HTTP server. See examples below. You can make some small edits. [Google publishes a fluentd output plugin that scans for exception patterns](https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions). The ones used by Stackdriver in production are different, but the concept is very similar.

### Things you can remove from the stack trace:

* Function addresses (`+0x2f5`)
* Exit status line
* Function arguments between `()`; `(..)` is also okay

### Things you cannot remove:

* If writing out to stderr: can't remove the "panic:" leading message. If the stack trace is in a structured log, it seems unnecessary
* Blank line after panic
* `goroutine` line
* Change the goroutine number to a string
* The parenthesis after the function name


## Kubernetes Engine

Traces are not as useful as you might hope, but it does let you query across the HTTP load balancer logs and the container logs. The error reporter does not capture panics from the HTTP server, but does capture the default formatted panics.


### Example query to combine HTTP and container logs

```sql
SELECT log.* EXCEPT (trace), http.* EXCEPT (timestamp, trace) FROM (
	SELECT trace, timestamp, severity, COALESCE(textPayload, jsonPayload.message) AS text
	FROM `bigquery-tools.logs.stderr_20190224`
) AS log
LEFT OUTER JOIN (
	SELECT trace, timestamp, httpRequest.status, httpRequest.requestMethod, httpRequest.requestUrl, jsonpayload_type_loadbalancerlogentry.statusdetails
	FROM `bigquery-tools.logs.requests_20190224`
) AS http ON log.trace = http.trace

WHERE log.trace = 'projects/bigquery-tools/traces/9572036a5aff8aae3cd8f1f053d348b1'
ORDER BY COALESCE(log.timestamp, http.timestamp)
```


## Cloud Run / App Engine (New Version) Collapsed Logs

In the "new" App Engine Standard (Java8, Python3, Go111), and in Cloud Run: if you include a trace ID in the correct format, the log viewer will collect all logs that came from one HTTP request. It shows you when you expand the HTTP request entry in the combined log:

![Log viewer screenshot](/appengine-collected-logs.png?raw=true "Log viewer screenshot")



## Cloud Functions

*Good news*: You don't need to do anything to get sensible logs with Cloud Functions!

*Bad news*: You can't customize how it handles logs at all.

Since currently Cloud Functions only allow a single request to execute a time, it automatically tags all log lines with both `trace` and `labels.execution_id`, so you can easily correlate log lines for a single request. All you need to do is write out your logs. Unfortunately, it does not appear that Cloud Functions will parse JSON, so you can't use structured logs.

Similarly, Cloud Functions will not automatically report errors for things that "look like" panics. Instead, it installs some sort of panic handler itself, which reports it to Stackdriver. It then writes the panic to the HTTP response with something that looks like the following:

```
Function panic: real panic

goroutine 6 [running]:
runtime/debug.Stack(0xc000153688, 0x6d5b20, 0x79c830)
	/go/src/runtime/debug/stack.go:24 +0xa7
main.executeFunction.func1(0x7a0180, 0xc0001380e0)
	/tmp/sgb/gopath/src/serverlessapp/worker.go:116 +0x6e
panic(0x6d5b20, 0x79c830)
	/go/src/runtime/panic.go:513 +0x1b9
serverlessapp/vendor/gcf.LogExample(0x7a0180, 0xc0001380e0, 0xc000130200)
	/tmp/sgb/gopath/src/serverlessapp/vendor/gcf/gcf.go:16 +0x5a8
main.executeFunction(0x7a0180, 0xc0001380e0, 0xc000130200)
	/tmp/sgb/gopath/src/serverlessapp/worker.go:123 +0x148c
net/http.HandlerFunc.ServeHTTP(0x75e970, 0x7a0180, 0xc0001380e0, 0xc000130200)
	/go/src/net/http/server.go:1964 +0x44
net/http.(*ServeMux).ServeHTTP(0x9b8540, 0x7a0180, 0xc0001380e0, 0xc000130200)
	/go/src/net/http/server.go:2361 +0x127
net/http.serverHandler.ServeHTTP(0xc000087040, 0x7a0180, 0xc0001380e0, 0xc000130200)
	/go/src/net/http/server.go:2741 +0xab
net/http.(*conn).serve(0xc00012e0a0, 0x7a0400, 0xc00001c200)
	/go/src/net/http/server.go:1847 +0x646
created by net/http.(*Server).Serve
	/go/src/net/http/server.go:2851 +0x2f5
```


## Example panic messages

Some example panic messages created by Go programs, which are useful for testing how Stackdriver catches and reports errors. These are logged to stderr, and contain tab characters for indenting.


### Default panic

The "standard" panic report printed before the process crashes:

```
panic: hello panic

goroutine 1 [running]:
main.panicNormally(...)
	/gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:28
main.funcWithArgs(0x1)
	/gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:32 +0x39
main.main()
	/gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:45 +0xb1
exit status 2
```


### HTTP server

Logged with log.Printf in the bowels of the http server:

```
2019/02/23 07:25:58 http: panic serving [::1]:62811: hello this is a panic
goroutine 37 [running]:
net/http.(*conn).serve.func1(0xc00013a1e0)
	/go/src/net/http/server.go:1746 +0xd0
panic(0x1246000, 0x12ebcd0)
	/go/src/runtime/panic.go:513 +0x1b9
main.realPanic(0x12efd60, 0xc00014c1c0, 0xc000176100)
	/gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:23 +0x39
net/http.HandlerFunc.ServeHTTP(0x12bd070, 0x12efd60, 0xc00014c1c0, 0xc000176100)
	/go/src/net/http/server.go:1964 +0x44
net/http.(*ServeMux).ServeHTTP(0x14a07a0, 0x12efd60, 0xc00014c1c0, 0xc000176100)
	/go/src/net/http/server.go:2361 +0x127
net/http.serverHandler.ServeHTTP(0xc000080f70, 0x12efd60, 0xc00014c1c0, 0xc000176100)
	/go/src/net/http/server.go:2741 +0xab
net/http.(*conn).serve(0xc00013a1e0, 0x12eff60, 0xc0000f8200)
	/go/src/net/http/server.go:1847 +0x646
created by net/http.(*Server).Serve
	/go/src/net/http/server.go:2851 +0x2f5
```


## Random bonus: formatting time benchmarks

I compared the following time formats:

* Unix seconds dot nanoseconds (e.g. `1550949427.724366000`): Using `fmt.Sprintf`, `strconv.Itoa`, and `strconv.AppendBytes`
* RFC3389 with nanoseconds (e.g. `2019-02-23T14:17:07.724366Z`): Using `time.Format`, and an unsafe cached implementation.

The results in order of fastest to slowest:

| Implementation | Time (ns/op) |
| --- | ---: |
| Unix strconv.Append to []byte | 71.0 |
| RFC3389 cached | 93.1 |
| Unix strconv.Append | 94.4 |
| Unix strconv.Itoa | 133.0 |
| Unix Sprintf | 204.0 |
| RFC3389 | 260.0 |
