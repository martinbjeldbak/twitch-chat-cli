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

# Optional: Run a local HTTP server to receive OAuth token from Twitch.tv (using the `auth` command)
EXPOSE 8090

ENTRYPOINT ["/twitch-chat-cli"]
CMD ["--help"]
