#!/bin/bash
set -euf -o pipefail
go test .
go vet .
~/google-cloud-sdk/bin/gcloud app deploy logdemo.yaml --project=bigquery-tools
