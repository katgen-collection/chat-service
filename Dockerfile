# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main ./cmd/api/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy from builder
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
# Optional: also copy docs if you want to serve swagger
COPY --from=builder /app/docs ./docs

EXPOSE 3001

CMD ["./main"]
