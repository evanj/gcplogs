package gcpzap

import (
	"bytes"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestEncodeLevel(t *testing.T) {
	// set up an encoder to write the levels
	cfg := zapcore.EncoderConfig{}
	cfg.LevelKey = "severity"
	cfg.EncodeLevel = encodeLevel
	enc, err := newEncoder(cfg)
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), nil)
	entry := zapcore.Entry{Level: zapcore.DebugLevel}
	err = core.Write(entry, nil)
	if err != nil {
		t.Fatal(err)
	}

	// manually check debug and fatal
	const expected = `{"severity":"DEBUG"}` + "\n"
	out := buf.String()
	if out != expected {
		t.Errorf("expected:%#v; got %#v", expected, out)
	}

	// ensure we can write all levels without errors: verifies our static array mapping levels
	for level := minLevel; level <= maxLevel; level++ {
		buf.Reset()
		entry.Level = level
		err = core.Write(entry, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestEncodeTime(t *testing.T) {
	// set up an encoder to write the levels
	cfg := zapcore.EncoderConfig{}
	cfg.TimeKey = "time"
	cfg.EncodeTime = encodeTime
	enc, err := newEncoder(cfg)
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), nil)
	entry := zapcore.Entry{Time: time.Unix(1551033753, 929117000)}
	err = core.Write(entry, nil)
	if err != nil {
		t.Fatal(err)
	}

	const expected = `{"time":"2019-02-24T18:42:33.929117Z"}` + "\n"
	out := buf.String()
	if out != expected {
		t.Errorf("expected:%#v; got %#v", expected, out)
	}
}
