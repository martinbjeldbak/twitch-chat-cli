# syntax=docker/dockerfile:1

FROM gcr.io/distroless/static-debian11
COPY twitch-chat-cli /

ENTRYPOINT ["/twitch-chat-cli"]
CMD ["--help"]
