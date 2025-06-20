# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for go mod)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o foundation-inquiry-bot .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and sqlite
RUN apk --no-cache add ca-certificates sqlite

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/foundation-inquiry-bot .

# Copy configuration example
COPY --from=builder /app/config.env.example .

# Create data directory for SQLite database
RUN mkdir -p data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./foundation-inquiry-bot"] 