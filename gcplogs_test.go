package gcplogs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProjectID(t *testing.T) {
	// create a sandbox directory and sanitize the environment
	tempDir := t.TempDir()

	// save the state of special environment variables and remove them so the test works
	specialEnvVars := []string{ProjectEnvVar, "GOOGLE_APPLICATION_CREDENTIALS", "PATH"}
	origValues := map[string]string{}
	for _, key := range specialEnvVars {
		origValues[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, value := range origValues {
			os.Setenv(key, value)
		}
	}()
	os.Setenv("PATH", tempDir)

	projectID := DefaultProjectID()
	if projectID != "" {
		t.Fatal("Initial project ID must be empty; some environment must be wrong?")
	}

	// point this to application default credentials
	keyPath := filepath.Join(tempDir, "key.json")
	err := os.WriteFile(keyPath, []byte(invalidServiceAccountKey), 0600)
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyPath)

	projectID = DefaultProjectID()
	if projectID != "bigquery-tools" {
		t.Error("project ID must be bigquery-tools with key:", projectID)
	}

	// override with GOOGLE_CLOUD_PROJECT: Used with Cloud Shell, new App Engine
	os.Setenv(ProjectEnvVar, "env-project")
	projectID = DefaultProjectID()
	if projectID != "env-project" {
		t.Error("Project environment variable must take priority:", projectID)
	}

	os.Unsetenv(ProjectEnvVar)
	os.Remove(keyPath)
	if DefaultProjectID() != "" {
		t.Error("need to reset environment to not find project ID")
	}

	// set up a fake gcloud
	gcloudPath := filepath.Join(tempDir, "gcloud")
	err = os.WriteFile(gcloudPath, []byte(fakeGcloud), 0700)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println("WTF", tempdir)
	// time.Sleep(time.Minute)
	projectID = DefaultProjectID()
	if projectID != "gcloud-project-id" {
		t.Error("incorrect gcloud project:", projectID)
	}
	args, err := os.ReadFile(gcloudPath + ".args")
	if err != nil {
		t.Fatal(err)
	}
	const expectedArgs = "config get-value core/project\n"
	if string(args) != expectedArgs {
		t.Error("wrong gcloud args:", string(args))
	}
}

const fakeGcloud = `#!/bin/sh
echo $@ > $0.args
echo "gcloud-project-id"`

// This is a revoked service account.
const invalidServiceAccountKey = `{
  "type": "service_account",
  "project_id": "bigquery-tools",
  "private_key_id": "55ed102c272e1aa954893d6cefeec82dddba0bf5",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCDWEgEeauzMUq+\n8i8MzbMBc2txKsnrlHuTaw+usaWD0wrr8ApuOXyii1q8u3EcvA1KFHI4kX9fzLKw\nVDNc8LhX/yhwd9EyJLuihb7D3URJyNksbf8IsR1/KH0y6VmeaJVR9F/amN6uFy6s\nw8C3IpTgQjzrR7pFXAK0A3MaSSCUxRSioVq5PnXoE67zyR7XC57BaUas6WGVm9RT\ncsgrw4g53cDmUkFcmjgxgEFoFF7B08FBitjc3X8S50k2heCVIC5F+tCQcdPiyfax\nIk/LLUEq3Nw4u99BSD36Deb7h9NUAjVYnLUF+qH0nlClP5EepSUvCJti537C1JfT\nAzXru46HAgMBAAECggEACIJxuAiB/UwWQaTDM5soG9H0hhJ1npOyJezryS+tP4su\ny/ZzVozW7FkG+e9S9r+gRMpqVAvpKrXCZfYulbjq2JipcA/zN8J1faQYpevx/q3K\nlDxUJ6YB+TkQU3oW2lKASh0BENKSqsjJt1u/Yp4U8yqXc87j3JaHfk4y6OMP/1Nl\nrgiHuYmWyHgb/rGaviHUoZf/tVCNv2jngfCCpGWhjmy+xFxD6dzmln3+5nzMM6DY\nfamC0RbCqT1IQDrwCiIVPAgH5gk2I1fhQKZOEiWkP9kLCrLrMxlFvyURGy0gASdN\nj5A3maXZlmuQ7DFvMcwqFku0i+DzBDhMwjCNpdJFUQKBgQC4ihkEvmEhLQFm4aqn\ncqe4c6N5+TS0ymbGuCGeS1wfoBKHIOjhYZcI5HoMVNSfQic0x4PsXAnJ54iLMqNw\nkhcaGyVamN0pB4MHRV80PTsf1k73P3Zz94sKKs82z6cYITCqtZ9cmLZrlDps8L/W\nxwjtaS1WMIIlhmoUXfi0kxJF3QKBgQC2NNtu0PFUH9Nt4sDtVtmqC+VQa9JlK0YA\nDOf6rGptDxqeZGZYA6U4+/ThPXi+JYjZQhNvir2JCSCsE9m9Pd+D8R5MPbGY2yfJ\n4yrEOW8ICdW1eqkHFOUQkudewzJqgs65nLuOk8QT34luRUyEPwy60e1tfz0E/3EW\nLbJeO9i5swKBgCcHpjT3owlmQGanEfXqbQi5BHlWuMwIBua+qPWW0Lwrmd+UmUyZ\n3FzYHewfwPyR/ELQc9l4ueVHH/z4z9KOQ26VEThxHk2ANjlCddlRngCkzfzDImVy\nlKio1zyrfJbA5k8krLjj36kvJ5BE9v4RCJVV6m3RQqV3IVZ/bYubk4DNAoGARq5k\nfS2CoH6kFxmCe89YKpXow/S/rk1GH1jiWKSvuFTGn7EU3omze1KKISImh6Sp3JW8\nUmXAtrsauIYOzlGFNnq/pRW9oi1J1xBPk8Uv5C9kfrzxevTJE0/ZfzI7iYPqy6gY\nPevmgUsS1fr9/sMynfo3n2Vfd2PcK51YdyPCI+8CgYEArAkSydX/3kyXjOOrZtds\nl4bE5QS0wcKoTPsmAlpTITD2/vTEwR73GLdYDyNj0f3r+KlKpnG4kO/UV7CGiAs5\nOtcPEcJnIIFv7tEBYCaHDbAxVPNno7smikDx/AFpvobZQVeULdWu0LjYICjILQ76\nEB+4ReLd/AmOCtOFiFzbsYg=\n-----END PRIVATE KEY-----\n",
  "client_email": "example@bigquery-tools.iam.gserviceaccount.com",
  "client_id": "103262392421472942815",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/example%40bigquery-tools.iam.gserviceaccount.com"
}`

func TestTracerFromRequest(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"invalid", ""},
		{"105445aa7843bc8bf206b120001000/0;o=1", "projects/test_id/traces/105445aa7843bc8bf206b120001000"},
	}

	tracer := &Tracer{"test_id"}
	zeroTracer := &Tracer{}

	for i, test := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(TraceHeader, test.input)

		output := tracer.FromRequest(req)
		if output != test.expected {
			t.Errorf("%d: FromRequest(%#v)=%#v; expected %#v", i, test.input, output, test.expected)
		}

		zeroOutput := zeroTracer.FromRequest(req)
		if zeroOutput != "" {
			t.Errorf("%d: FromRequest() must return the empty string if ProjectID is not set: %#v",
				i, zeroOutput)
		}
	}
}
