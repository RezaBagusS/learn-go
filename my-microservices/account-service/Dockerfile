# ==========================================
# Tahap 1: Builder
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /workspace
COPY shared/go.mod shared/go.sum ./shared/
COPY account-service/go.mod account-service/go.sum ./account-service/

# Pindah ke service folder
WORKDIR /workspace/account-service
RUN go mod edit -replace my-microservices/shared=../shared
RUN --mount=type=cache,target=/go/pkg/mod GOWORK=off go mod download

WORKDIR /workspace
COPY shared/ ./shared/
COPY account-service/ ./account-service/

WORKDIR /workspace/account-service
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOWORK=off \
    go build -o /app-binary ./cmd/api/main.go

# ==========================================
# Tahap 2: Runner
# ==========================================
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app-binary .
COPY account-service/.env .
COPY account-service/migrations ./migrations
EXPOSE 8080
EXPOSE 50051
CMD ["./app-binary"]