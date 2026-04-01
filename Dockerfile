# Build
FROM golang:1.26-alpine AS builder

# Directory Container
WORKDIR /app

# Config Golang
COPY go.mod go.sum ./
RUN go mod download

# Copy semua
COPY . .

# Build -> main
RUN go build -o main .

# Build lebih ringan
FROM alpine:latest
WORKDIR /root/

# Copy dari build -> main
COPY --from=builder /app/main .
# Copy env
COPY .env . 

# Port 
EXPOSE 8080

# Start app
CMD ["./main"]