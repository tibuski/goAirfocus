# Build stage
FROM golang:1.21-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o airfocus-tools .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/airfocus-tools .

# Copy static files and templates
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./airfocus-tools"] 