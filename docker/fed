# Build fedd in a stock Go builder container
FROM golang:1.12-alpine as builder

RUN apk add --no-cache make git

ADD . /go/src/github.com/bytom/vapor
WORKDIR /go/src/github.com/bytom/vapor/cmd/fedd

RUN GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix cgo -o fed main.go

# Pull Bytom into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/bytom/vapor/cmd/fedd/fed /usr/local/bin/

EXPOSE 9886
