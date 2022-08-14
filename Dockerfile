# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.19-bullseye

WORKDIR /go/src/twitch-chat-cli
COPY *.go .

RUN go mod init
RUN go get -d -v ./...
RUN go vet -v
RUN go test -v

RUN go build -o /go/bin/twitch-chat-cli

CMD ["/twitch-chat-cli"]
