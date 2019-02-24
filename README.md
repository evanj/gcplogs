# Google Cloud Platform Logs

This is a set of experiments for how logs are automatically collected by Google Cloud Platform products. If you write a log line in the correct magic format, it will get sucked up by Stackdriver Logging, and will preserve the "special" fields such as the log level/severity and timestamp. It will also record the log with "structured" payloads, which are then easily searchable in BigQurey. If you write exceptions with stack traces in the right format, they will automatically be recorded by the Error Reporting. I spent a bit of time figuring out how these work, so we can configure our applications to "just work" on Google Cloud.

These examples are in Go, because I happen to like Go, but some of the formatting things are language independent.

Google provides a fluentd output plugin that scans for exception patterns to combine the errors into a single entry that can then get picked up by error reporting. The ones used by Stackdriver in production, at least on App Engine are different, but the concept is very similar. See: https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions


## App Engine (New go111 universe)

This is not about the old "legacy" App Engine, or the App Engine "flexible": One is dead and the other seems useless. The old legacy App Engine used to collect all logs from a single request together. This was incredibly useful. In the new world, you can approximate that by parsing the trace ID from the `X-Cloud-Trace-Context` header. The format is `X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE`. You can then include this trace ID in future log lines, and the tool will collapse the messages. For details on the other parts, see the [Stackdriver Trace reference for the X-Cloud-Trace-Context header](https://cloud.google.com/trace/docs/troubleshooting#force-trace). 

The tricky part is the trace log is in the format `projects/PROJECT_ID/traces/TRACE_ID`. Not only do you need to parse the trace ID, but you also need to parse the project. The good news is in many cases we can auto-detect the project ID. On App Engine, it is available in the .  


To report errors to the error report, you need to write out a record that looks like a panic, either as reported by the default Go panic handler, or by the net/http server handler. You can make some small edits:

### Things you can remove from the stack trace:

* Function addresses (`+0x2f5`)
* Exit status line
* Function arguments between `()`; `(..)` is also okay

### Things you cannot remove:

* Blank line after panic
* `goroutine` line
* Change the goroutine number to a string
* The parenthesis after the function name


## Cloud Functions

*Good news*: You don't need to do anything to get sensible logs with Cloud Functions!

*Bad news*: You can't customize how it handles logs at all.

Since currently Cloud Functions only allow a single request to execute a time, it automatically tags all log lines with both `trace` and `labels.execution_id`, so you can easily correlate log lines for a single request. All you need to do is write out your logs. Unfortunately, it does not appear that Cloud Functions will automatically parse JSON. Despite the Stackdriver documentation mentioning that you might be able to use the "log" field: https://cloud.google.com/logging/docs/agent/configuration#process-payload

Logging the following payload did not work: `{"log": "json warning", "severity": "WARNING"}`

Similarly, Cloud Functions will not automatically report errors for things that "look like" panics. It installs some sort of panic handler itself which reports directly to Stackdriver error reporting for real panics. It then writes the panic to the HTTP response with something that looks like the following:

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

# Random bonus: formatting time benchmarks

I compared the following time formats:

* Unix seconds `.` nanoseconds (e.g. 1550949427.724366000): Using `fmt.Sprintf`, `strconv.Itoa`, and `strconv.AppendBytes`
* RFC3389 with nanoseconds (e.g. 2019-02-23T14:17:07.724366Z): Using `time.Format`, and a not thread safe cached implementation.

The results in order of fastest to slowest:

* Unix strconv.Append to []byte instead of string: 71.0 ns/op
* RFC3389 cached: 93.1 ns/op
* Unix strconv.Append: 94.4 ns/op
* Unix strconv.Itoa: 133 ns/op
* Unix Sprintf: 204 ns/op
* RFC3389: 260 ns/op
