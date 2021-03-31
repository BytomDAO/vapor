# Build Vapor in a stock Go builder container
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make git

ADD . /go/src/github.com/bytom/vapor
RUN cd /go/src/github.com/bytom/vapor/toolbar/osssync && go build -o cmd/osssync cmd/main.go

# Pull Vapor into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/bytom/vapor/toolbar/osssync/cmd/osssync /usr/local/bin/