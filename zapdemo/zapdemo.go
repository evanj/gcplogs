package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/evanj/gcplogs"
	"github.com/evanj/gcplogs/gcpzap"
	"go.uber.org/zap"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Write([]byte(rootHTML))
}

type server struct {
	tracer gcpzap.Tracer
}

func (s *server) logDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")

	reqLogger := s.tracer.FromRequest(r)
	fmt.Fprintf(w, "wrote some logs:\n")

	reqLogger.Debug("debug message", zap.Int("example_key", 100))
	reqLogger.Info("info message", zap.Int("example_key", 101))
	reqLogger.Warn("warning message", zap.Int("example_key", 102))
	reqLogger.Error("error message", zap.Int("example_key", 103))
}

func (s *server) fatalDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=utf-8")
	fmt.Fprintf(w, "writing fatal level\n")

	s.tracer.FromRequest(r).Fatal("fatal message")
}

func main() {
	logger, err := gcpzap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	projectID := gcplogs.DefaultProjectID()
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "Could not find Google Project ID; Set "+gcplogs.ProjectEnvVar)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	listenAddr := ":" + port

	logger.Info("zapdemo starting ...", zap.String("projectID", projectID), zap.String("addr", listenAddr))

	s := &server{gcpzap.Tracer{Tracer: gcplogs.Tracer{ProjectID: projectID}, Logger: logger}}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/log_demo", s.logDemo)
	http.HandleFunc("/fatal", s.fatalDemo)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

const rootHTML = `<!DOCTYPE html><html>
<head><title>Zap Demo</title></head>
<body>
<h1>Zap Demo</h1>
<p>This application tests logging with Uber's zap.</p>
<ul>
<li><a href="/log_demo">Writes some log messages with zap</a></li>
<li><a href="/fatal">Writes a message at fatal level</a></li>
</ul>
</body></html>
`
