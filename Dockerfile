FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /fissio-server ./cmd/fissio-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /fissio-server /fissio-server
EXPOSE 8000
CMD ["/fissio-server"]
