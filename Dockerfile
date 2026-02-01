FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /fissio ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1000 fissio && \
    mkdir -p /data && chown fissio:fissio /data

USER fissio
WORKDIR /app
COPY --from=builder /fissio /app/fissio

EXPOSE 8000
VOLUME ["/data"]
ENV FISSIO_DATA_DIR=/data

CMD ["/app/fissio"]
