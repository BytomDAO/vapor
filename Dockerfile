# Build Bytom in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make git

ADD . /go/src/github.com/vapor
RUN cd /go/src/github.com/vapor && make vapor && make vaporcli

# Pull Bytom into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/vapor/cmd/vapor/vapor /usr/local/bin/
COPY --from=builder /go/src/github.com/vapor/cmd/vaporcli/vaporcli /usr/local/bin/

EXPOSE 1999 46656 46657 9888
