# Build stage
FROM golang:1.24-bullseye AS builder
RUN apt-get update && apt-get install -y \
    gcc g++ make pkg-config librdkafka-dev \
    && rm -rf /var/lib/apt/lists/* \
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN mkdir -p /var/log && touch /var/log/wbOrderSaver.log && chmod 644 /var/log/wbOrderSaver.log
COPY . .
RUN go build -o /app/bin/wbOrderSaver ./cmd/wbOrderSaver/main.go

# Runtime stage
FROM debian:bullseye-slim
WORKDIR /app
RUN apt-get update && apt-get install -y librdkafka1 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/bin/wbOrderSaver /app/wbOrderSaver
CMD ["/app/wbOrderSaver"]
