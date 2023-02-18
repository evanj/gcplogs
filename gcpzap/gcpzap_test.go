package gcpzap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/evanj/gcplogs"
	"go.uber.org/zap"
)

type stderrInterceptor struct {
	origStderr *os.File
	tempFile   *os.File
}

func interceptStderr(t *testing.T) *stderrInterceptor {
	tempDir := t.TempDir()
	f, err := os.CreateTemp(tempDir, "test_stderr")
	if err != nil {
		panic(err)
	}

	interceptor := &stderrInterceptor{os.Stderr, f}
	os.Stderr = f
	return interceptor
}

func (s *stderrInterceptor) readAll() string {
	_, err := s.tempFile.Seek(0, io.SeekStart)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(s.tempFile)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (s *stderrInterceptor) Close() {
	os.Stderr = s.origStderr
	err := s.tempFile.Close()
	if err != nil {
		panic(err)
	}
	err = os.Remove(s.tempFile.Name())
	if err != nil {
		panic(err)
	}
}

func TestNewProduction(t *testing.T) {
	// replace stderr with a temporary file
	interceptor := interceptStderr(t)
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
	interceptor := interceptStderr(t)
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
