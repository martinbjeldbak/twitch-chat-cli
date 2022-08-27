# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.19-bullseye as build

WORKDIR /go/src/twitch-chat-cli
COPY . .

RUN go mod download
RUN go vet -v
RUN go test -v

RUN CGO_ENABLED=0 go build -o /go/bin/twitch-chat-cli

FROM gcr.io/distroless/static-debian11

COPY --from=build /go/bin/twitch-chat-cli /
CMD ["/twitch-chat-cli"]
