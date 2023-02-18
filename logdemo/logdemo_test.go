package main

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func fixedNanos(nanoseconds int) string {
	const zeros = "000000000"
	nanosString := strconv.Itoa(nanoseconds)
	return zeros[0:len(zeros)-len(nanosString)] + nanosString
}

func strconvFormat(t time.Time) string {
	secondsString := strconv.Itoa(int(t.Unix()))
	return secondsString + "." + fixedNanos(t.Nanosecond())
}

func appendFixedNanos(dst []byte, nanoseconds int) []byte {
	const nsFixedSize = 9
	nsBuffer := make([]byte, 0, nsFixedSize)
	nsBuffer = strconv.AppendInt(nsBuffer, int64(nanoseconds), 10)
	for i := len(nsBuffer); i < nsFixedSize; i++ {
		dst = append(dst, '0')
	}
	dst = append(dst, nsBuffer...)
	return dst
}

func strconvAppendFormatBytes(t time.Time) []byte {
	// 10 for unix seconds, 1 for ., 9 for nanos
	const maxSize = 20

	buffer := make([]byte, 0, maxSize)
	buffer = strconv.AppendInt(buffer, t.Unix(), 10)
	buffer = append(buffer, '.')

	buffer = appendFixedNanos(buffer, t.Nanosecond())
	return buffer
}

func strconvAppendFormat(t time.Time) string {
	return string(strconvAppendFormatBytes(t))
}

func fnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func TestTimeFormats(t *testing.T) {
	fns := []func(t time.Time) string{
		formatUnixWithNanos, strconvFormat, strconvAppendFormat,
	}

	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Unix(1550948234, 358873000), "1550948234.358873000"},
		{time.Unix(1550948234, 0), "1550948234.000000000"},
		{time.Unix(1550948234, 1), "1550948234.000000001"},
		{time.Unix(1550948234, 999999999), "1550948234.999999999"},
		{time.Unix(1550948234, 99999999), "1550948234.099999999"},
	}

	for i, test := range tests {
		for j, fn := range fns {
			out := fn(test.input)
			if out != test.expected {
				t.Errorf("test %d fn %d %s(%#v)=%s; expected %s",
					i, j, fnName(fn), test.input.Format(time.RFC3339Nano), out, test.expected)
			}

			outUTC := fn(test.input.UTC())
			if outUTC != test.expected {
				t.Errorf("test %d fn %d %s(%#v)=%s; expected %s",
					i, j, fnName(fn), test.input.Format(time.RFC3339Nano), outUTC, test.expected)
			}
		}
	}
}

// this is RFC3339Nano with fixed-width nanoseconds
const rfc3339NanoLeadingZeros = "2006-01-02T15:04:05.000000000Z07:00"

func rfcFormat(t time.Time) string {
	return t.UTC().Format(rfc3339NanoLeadingZeros)
}

var lastMinute time.Time
var lastFormattedMinute []byte

// This is a not thread safe implementation where we cache the last formatted time until the next
// minute. In the case of logging, this is the extremely common case. The benchmark here takes this
// from 260 ns/op -> 93 ns/op
func rfcFormatInsanity(t time.Time) string {
	const minutePrefixBytes = 17

	minute := t.Truncate(time.Minute)
	if minute.Equal(lastMinute) && t.After(minute) {
		// cached case! Create the output buffer
		const afterMinuteBytes = 2 + 1 + 9 + 1
		output := make([]byte, 0, minutePrefixBytes+afterMinuteBytes)
		output = append(output, lastFormattedMinute...)

		seconds := t.Unix() - minute.Unix()
		output = strconv.AppendInt(output, seconds, 10)

		output = append(output, '.')
		output = appendFixedNanos(output, t.Nanosecond())
		output = append(output, 'Z')
		return string(output)
	}
	out := t.UTC().Format(rfc3339NanoLeadingZeros)
	lastMinute = minute
	lastFormattedMinute = []byte(out[:minutePrefixBytes])

	return out
}

func TestRFCFormats(t *testing.T) {
	fns := []func(t time.Time) string{
		rfcFormat, rfcFormatInsanity,
	}

	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Unix(1550948234, 358873000), "2019-02-23T18:57:14.358873000Z"},
		{time.Unix(1550948234, 0), "2019-02-23T18:57:14.000000000Z"},
		{time.Unix(1550948234, 1), "2019-02-23T18:57:14.000000001Z"},
		{time.Unix(1550948234, 999999999), "2019-02-23T18:57:14.999999999Z"},
		{time.Unix(1550948234, 99999999), "2019-02-23T18:57:14.099999999Z"},
	}

	for i, test := range tests {
		for j, fn := range fns {
			out := fn(test.input)
			if out != test.expected {
				t.Errorf("test %d fn %d %s(%#v)=%s; expected %s",
					i, j, fnName(fn), test.input.Format(time.RFC3339Nano), out, test.expected)
			}

			outUTC := fn(test.input.UTC())
			if outUTC != test.expected {
				t.Errorf("test %d fn %d %s(%#v)=%s; expected %s",
					i, j, fnName(fn), test.input.Format(time.RFC3339Nano), outUTC, test.expected)
			}
		}
	}
}

var doNotOptimize int

var benchTime = time.Unix(1550948234, 358873000)

func benchFn(b *testing.B, fn func(time.Time) string) {
	total := 0
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		formatted := fn(benchTime)
		total += len(formatted)
	}
	doNotOptimize = total
}

func BenchmarkTimeFormat(b *testing.B) {
	fns := []func(time.Time) string{
		formatUnixWithNanos, strconvFormat, strconvAppendFormat, rfcFormat, rfcFormatInsanity,
	}

	for _, fn := range fns {
		f := fn
		b.Run(fnName(fn), func(b *testing.B) {
			benchFn(b, f)
		})
	}
}

func BenchmarkAppendBytes(b *testing.B) {
	total := 0
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		formatted := strconvAppendFormatBytes(benchTime)
		total += len(formatted)
	}
	doNotOptimize = total
}
