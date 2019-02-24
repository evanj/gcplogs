package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/evanj/gcplogs"
)

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
	log.Printf("projectID: %s", gcplogs.DefaultProjectID())
}

// Stackdriver's nested timestamp JSON.
type logTimestamp struct {
	Seconds int64 `json:"seconds,omitempty"`
	Nanos   int   `json:"nanos,omitempty"`
}

// Contains data to be logged so Stackdriver parses it correctly. This is made for experimentation
// so it contains multiple timestamp types. See:
// https://cloud.google.com/logging/docs/agent/configuration#special-fields
// Even though this documents a format with JSON key "time" as a unix seconds "." nanos field,
// that does not work. Formatting "time" as RFC3389 with nanoseconds does work.
type stackdriverLine struct {
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
	TraceID  string `json:"logging.googleapis.com/trace,omitempty"`
	// SpanID is documented but doesn't seem useful for anything?
	SpanID           string        `json:"logging.googleapis.com/spanId,omitempty"`
	Timestamp        *logTimestamp `json:"timestamp,omitempty"`
	Time             string        `json:"time,omitempty"`
	TimestampSeconds int64         `json:"timestampSeconds,omitempty"`
	TimestampNanos   int           `json:"timestampNanos,omitempty"`
}

func formatUnixWithNanos(t time.Time) string {
	return fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
}

// Another version with a different timestamp format
type altStackdriverLine struct {
	Severity        string `json:"severity,omitempty"`
	Message         string `json:"message,omitempty"`
	TraceID         string `json:"logging.googleapis.com/trace,omitempty"`
	TimestampString string `json:"timestamp,omitempty"`
}

const traceKey = "logging.googleapis.com/trace"

func mustLogLine(w io.Writer, line interface{}) {
	serialized, err := json.Marshal(line)
	if err != nil {
		panic(err)
	}
	serialized = append(serialized, '\n')
	_, err = w.Write(serialized)
	if err != nil {
		panic(err)
	}
}

type server struct {
	tracer gcplogs.Tracer
}

func (s *server) logDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	fmt.Fprintf(w, "wrote some log lines to stderr:\n\n")

	output := io.MultiWriter(w, os.Stderr)
	now := time.Now().UTC().Truncate(time.Millisecond).Add(987654)
	traceID := s.tracer.FromRequest(r)

	line := &stackdriverLine{
		Severity:  "DEBUG",
		Message:   "debug with timestamp struct field (works)",
		TraceID:   traceID,
		Timestamp: &logTimestamp{now.Unix(), now.Nanosecond()},
	}
	mustLogLine(output, line)

	line.Severity = "INFO"
	line.Message = "info with time string (DOES NOT WORK)"
	line.Timestamp = nil
	now = now.Add(2 * time.Millisecond)
	line.Time = formatUnixWithNanos(now)
	mustLogLine(output, line)

	line.Severity = "WARNING"
	line.Message = "warning with timestampNano/timestampSecond (works)"
	line.Time = ""
	now = now.Add(2 * time.Millisecond)
	line.TimestampSeconds = now.Unix()
	line.TimestampNanos = now.Nanosecond()
	mustLogLine(output, line)

	line.Severity = "ERROR"
	line.Message = "error with time in RFC3339Nano (works)"
	line.TimestampSeconds = 0
	line.TimestampNanos = 0
	now = now.Add(2 * time.Millisecond)
	line.Time = now.Format(time.RFC3339Nano)
	mustLogLine(output, line)

	now = now.Add(2 * time.Millisecond)
	altLine := altStackdriverLine{"CRITICAL", "critical with timestamp in RFC3339Nano (DOES NOT WORK)", traceID,
		now.Format(time.RFC3339Nano)}
	mustLogLine(output, altLine)

	// wait long enough that the logged times are in the past
	time.Sleep(20 * time.Millisecond)
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

	projectID := gcplogs.DefaultProjectID()
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "Could not find Google Project ID; Set "+gcplogs.ProjectEnvVar)
		os.Exit(1)
	}
	log.Printf("detected projectID:%s", projectID)

	s := &server{gcplogs.Tracer{ProjectID: projectID}}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/log_demo", s.logDemo)
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
<li><a href="/log_demo">Demo/test of JSON logging formats</a></li>
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
