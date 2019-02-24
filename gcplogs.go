package gcplogs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/oauth2/google"
)

const ProjectEnvVar = "GOOGLE_CLOUD_PROJECT"

// DefaultProjectID detects the current Google Cloud project ID, or return the empty string if it
// fails. This function reads files, makes HTTP requests, and might execute binaries. An
// application should not call it often. It is possible for the result to change while the
// application is running, such as if a key file is updated or the user changes the default project
// with gcloud. This is an implementation of the Java ServiceOptions.getDefaultProjectId:
// https://github.com/googleapis/google-cloud-java/blob/master/google-cloud-clients/google-cloud-core/src/main/java/com/google/cloud/ServiceOptions.java
//
// Python uses google.auth.default which has similar logic:
// https://github.com/googleapis/google-auth-library-python/blob/master/google/auth/_default.py
//
// This should eventually be replaced with whatever is implemented in the official GCP Go API:
// https://github.com/googleapis/google-cloud-go/issues/1294
//
// The approaches it uses are:
// * GOOGLE_CLOUD_PROJECT environment variable (manual, App Engine, Cloud Shell)
// * Application default credentials (Compute Engine, service account key)
// * Gcloud default project
func DefaultProjectID() string {
	// environment variable: manually configured, new App Engine, Cloud Shell
	projectID := os.Getenv(ProjectEnvVar)
	if projectID != "" {
		return projectID
	}

	// contains a project ID on compute engine or when using a key file
	// does NOT contain a project ID with personal gcloud credentials
	projectID, _ = defaultCredentialsProjectID()
	if projectID != "" {
		return projectID
	}

	projectID, _ = gcloudConfigProjectID()
	return projectID
}

// Return the project ID from application default credentials. This works on compute engine
// or when using a service account key.
func defaultCredentialsProjectID() (string, error) {
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx)

	if err != nil {
		return "", err
	}
	return creds.ProjectID, nil
}

func gcloudConfigProjectID() (string, error) {
	// attempt to load the gcloud config by executing gcloud
	// TODO: Java reads the configuration file directly?
	// attempt to execute gcloud
	cmd := exec.Command("gcloud", "config", "get-value", "core/project")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to get default project from gcloud: %s",
			err.Error())
	}
	// out contains the value with a new line
	projectID := string(bytes.TrimSpace(out))
	return projectID, nil
}
