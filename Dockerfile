# Go build image
FROM golang:1.20.1-bullseye AS go_builder
COPY . gcplogs
WORKDIR gcplogs
RUN go install --mod=readonly -v ./logdemo ./zapdemo

# logdemo image
FROM gcr.io/distroless/base-debian11:nonroot AS logdemo
COPY --from=go_builder /go/bin/logdemo /
ENTRYPOINT ["/logdemo"]

# zapdemo image
FROM gcr.io/distroless/base-debian11:nonroot AS zapdemo
COPY --from=go_builder /go/bin/zapdemo /
ENTRYPOINT ["/zapdemo"]
