package gcf

import (
	"log"
	"net/http"
	"os"
)

func LogExample(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	w.Write([]byte("ok path:" + r.URL.Path))
	log.Printf("hello log url=%s remoteAddr=%s", r.URL.String(), r.RemoteAddr)

	if r.URL.Path == "/real_panic" {
		w.Write([]byte("\n\nreal panic"))
		panic("real panic")
	} else if r.URL.Path == "/default_panic" {
		w.Write([]byte("\n\nreplayed default panic"))
		os.Stderr.Write([]byte(defaultPanic))
	} else if r.URL.Path == "/http_panic" {
		w.Write([]byte("\n\nreplayed http panic"))
		os.Stderr.Write([]byte(httpPanic))
	} else {
		w.Write([]byte("\n\nlog test"))
		// documentation suggests we can use "log" with Google Cloud functions; doesn't seem to work?
		// https://cloud.google.com/logging/docs/agent/configuration#process-payload
		os.Stdout.Write([]byte(`{"log": "json warning", "severity": "WARNING"}` + "\n"))
		os.Stdout.Write([]byte(`{"log": "json warning with extra key", "severity": "WARNING", "key": 42}` + "\n"))
	}
}

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
