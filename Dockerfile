# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.17-bullseye as build

WORKDIR /go/src/app
COPY *.go .

RUN go mod init
RUN go get -d -v ./...
RUN go vet -v
RUN go test -v

RUN CGO_ENABLED=0 go build -o /go/bin/app

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

COPY --from=build /go/bin/app /

CMD ["/app"]
