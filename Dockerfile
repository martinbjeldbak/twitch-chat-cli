# syntax=docker/dockerfile:1

FROM scratch
COPY twitch-chat-cli /

ENTRYPOINT ["/twitch-chat-cli"]
CMD ["--help"]
