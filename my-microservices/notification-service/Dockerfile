# Build
FROM golang:1.26-alpine AS builder

# Directory Container
WORKDIR /app

# Config Golang
COPY go.mod go.sum ./
RUN go mod download

# Copy semua source code (Hanya dari folder service ini)
COPY . .


# Build -> main (Ditambahkan CGO_ENABLED agar aman di Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# Build lebih ringan
FROM alpine:latest
WORKDIR /root/

# Copy dari build -> main
COPY --from=builder /app/main .

# Copy migrations (Sangat bagus kamu menambahkan ini)
COPY --from=builder /app/migrations ./migrations  

# Port 
EXPOSE 8080

# Start app
CMD ["./main"]