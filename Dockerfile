FROM golang:1.16-alpine AS builder

COPY . /meditime

WORKDIR /meditime

RUN go build

FROM alpine:3.13

RUN apk add --no-cache \
	ca-certificates \
	tzdata

COPY --from=builder /meditime/meditime /usr/local/bin/meditime

ENTRYPOINT ["/usr/local/bin/meditime"]
