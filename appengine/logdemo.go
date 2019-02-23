package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const cloudTraceHeader = "X-Cloud-Trace-Context"

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	w.Write([]byte(rootHTML))
}

// Returns the trace ID from a cloud trace header, or the empty string if it does not exist. See:
// https://cloud.google.com/trace/docs/troubleshooting#force-trace
func parseTraceID(r *http.Request) string {
	headerValue := r.Header.Get(cloudTraceHeader)
	slashIndex := strings.IndexByte(headerValue, '/')
	if slashIndex < 0 {
		return ""
	}
	return headerValue[:slashIndex]
}

func logDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")

	traceID := parseTraceID(r)

}

func realPanic(w http.ResponseWriter, r *http.Request) {
	panic("hello this is a panic")
}

func panicPrinter(w http.ResponseWriter, panicMessage string) {
	os.Stderr.WriteString(panicMessage)
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	w.Write([]byte("panic written to stderr:\n\n"))
	w.Write([]byte(panicMessage))
}

func replayDefaultPanic(w http.ResponseWriter, r *http.Request) {
	panicPrinter(w, defaultPanic)
}

func replayHTTPPanic(w http.ResponseWriter, r *http.Request) {
	panicPrinter(w, httpPanic)
}

func writeStderr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "must use post", http.StatusMethodNotAllowed)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "ok; writing %d bytes to stderr\n", len(data))
	os.Stderr.Write(data)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/log_demo", logDemo)
	http.HandleFunc("/panic", realPanic)
	http.HandleFunc("/replay_default_panic", replayDefaultPanic)
	http.HandleFunc("/replay_http_panic", replayHTTPPanic)
	http.HandleFunc("/write_stderr", writeStderr)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

const rootHTML = `<!DOCTYPE html><html>
<head><title>App Engine Logging Demo</title></head>
<body>
<h1>App Engine Logging Demo</h1>
<p>This application tests logging errors and messages that should get detected by Google Stackdriver when deployed on App Engine.</p>
<ul>
<li><a href="/panic">A real panic, caught by the http server</a></li>
<li><a href="/replay_default_panic">Replay "standard" panic</a></li>
<li><a href="/replay_http_panic">Replay http server panic</a></li>
<li><a href="/write_stderr">POST ONLY: Use <code>curl --data 'something' URL</code> to write to stderr</a></li>
</ul>
</body></html>
`

const defaultPanic = `panic: not a real panic (default)

goroutine 1 [running]:
main.panicNormally(...)
  /gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:28
main.funcWithArgs(0x1)
  /gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:32 +0x39
main.main()
  /gopath/src/github.com/evanj/gcplogs/appengine/logdemo.go:45 +0xb1
exit status 2
`

const httpPanic = `2019/02/23 07:25:58 http: panic serving [::1]:62811: not a real panic (http)
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
`
