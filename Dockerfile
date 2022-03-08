FROM golang:1.17-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY qbittorrent.go .

RUN go build -o /app/grafana-qbittorrent-go
RUN rm go.mod go.sum main.go qbittorrent.go

FROM alpine:3.15

WORKDIR /app

COPY --from=builder /app/grafana-qbittorrent-go /app/grafana-qbittorrent-go
COPY config.toml /app/config.toml
ENTRYPOINT [ "/app/grafana-qbittorrent-go", "-c", "/app/config.toml" ]