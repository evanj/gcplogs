# Go build image
FROM golang:1.13.5-buster AS go_builder
COPY . gcplogs
WORKDIR gcplogs
RUN go install --mod=readonly -v ./logdemo ./zapdemo

# logdemo image
FROM gcr.io/distroless/base-debian10:nonroot AS logdemo
COPY --from=go_builder /go/bin/logdemo /
ENTRYPOINT ["/logdemo"]

# zapdemo image
FROM gcr.io/distroless/base-debian10:nonroot AS zapdemo
COPY --from=go_builder /go/bin/zapdemo /
ENTRYPOINT ["/zapdemo"]
