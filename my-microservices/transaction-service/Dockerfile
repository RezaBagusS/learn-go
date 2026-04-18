FROM golang:1.26-alpine AS builder

WORKDIR /workspace

# Copy semua source dulu
COPY shared/ ./shared/
COPY transaction-service/ ./transaction-service/

WORKDIR /workspace/transaction-service

# Tidy akan otomatis resolve require + replace
RUN go mod edit -replace my-microservices/shared=../shared
RUN GOWORK=off go mod tidy

# Build
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOWORK=off \
    go build -o /app-binary ./cmd/api/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app-binary .
COPY transaction-service/.env .
COPY transaction-service/migrations ./migrations
EXPOSE 8080
EXPOSE 50051
CMD ["./app-binary"]