# Build Vapor in a stock Go builder container
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make git

ADD . /go/src/github.com/bytom/vapor
RUN cd /go/src/github.com/bytom/vapor && make vapord && make vaporcli

# Pull Vapor into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/bytom/vapor/cmd/vapord/vapord /usr/local/bin/
COPY --from=builder /go/src/github.com/bytom/vapor/cmd/vaporcli/vaporcli /usr/local/bin/

EXPOSE 9889 56656 56657 56658
