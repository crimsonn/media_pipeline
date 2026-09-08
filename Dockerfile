# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/transcoder ./cmd/transcoder
RUN CGO_ENABLED=0 go build -o /out/watchdog ./cmd/watchdog

FROM debian:bookworm-slim AS watchdog
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/watchdog /usr/local/bin/watchdog
CMD ["watchdog"]

FROM debian:bookworm-slim AS transcoder
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates ffmpeg \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/transcoder /usr/local/bin/transcoder
CMD ["transcoder"]
