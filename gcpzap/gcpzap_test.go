package gcpzap

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/evanj/gcplogs"

	"testing"

	"go.uber.org/zap"
)

type stderrInterceptor struct {
	origStderr *os.File
	tempfile   *os.File
}

func interceptStderr() *stderrInterceptor {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	interceptor := &stderrInterceptor{os.Stderr, tempfile}
	os.Stderr = tempfile
	return interceptor
}

func (s *stderrInterceptor) readAll() string {
	_, err := s.tempfile.Seek(0, os.SEEK_SET)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(s.tempfile)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (s *stderrInterceptor) Close() {
	os.Stderr = s.origStderr
	err := s.tempfile.Close()
	if err != nil {
		panic(err)
	}
	err = os.Remove(s.tempfile.Name())
	if err != nil {
		panic(err)
	}
}

func TestNewProduction(t *testing.T) {
	// replace stderr with a temporary file
	interceptor := interceptStderr()
	defer interceptor.Close()

	// calling NewProduction twice must succeed (ignoring duplicate errors)
	_, err := NewProduction()
	if err != nil {
		t.Fatal(err)
	}
	logger, err := NewProduction()
	if err != nil {
		t.Fatal(err)
	}
	logger.Error("message", zap.Int("example", 42))
	logger.Sync()

	loggedString := interceptor.readAll()
	if !strings.Contains(loggedString, `"severity":"ERROR"`) {
		t.Error("wrong severity:", loggedString)
	}
	if !strings.Contains(loggedString, ".TestNewProduction(...)\\n") {
		t.Error("incorrect stack trace?", loggedString)
	}
}

func TestWithTrace(t *testing.T) {
	// replace stderr with a temporary file
	interceptor := interceptStderr()
	defer interceptor.Close()

	rootLogger, err := NewProduction()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(gcplogs.TraceHeader, "traceid/spanid")
	tracer := &Tracer{gcplogs.Tracer{ProjectID: "projectid"}, rootLogger}
	logger := tracer.FromRequest(r)
	logger.Info("message")

	logString := interceptor.readAll()

	const expected = `"logging.googleapis.com/trace":"projects/projectid/traces/traceid"`
	if !strings.Contains(logString, expected) {
		t.Errorf("log should contain %#v; %#v", expected, logString)
	}
}
