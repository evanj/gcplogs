package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTraceID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"invalid", ""},
		{"105445aa7843bc8bf206b120001000/0;o=1", "105445aa7843bc8bf206b120001000"},
	}

	for i, test := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(cloudTraceHeader, test.input)

		output := parseTraceID(req)
		if output != test.expected {
			t.Errorf("%d: parseTraceID(%#v)=%#v; expected %#v", i, test.input, output, test.expected)
		}
	}
}
