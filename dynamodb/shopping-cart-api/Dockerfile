# Build stage - Use official Go image (Debian-based, AWS-compatible)
FROM golang:1.21-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY *.go ./

# Build the application with optimizations
# CGO_ENABLED=0 for static binary (works on any Linux)
# Reduce binary size with -ldflags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o shopping-cart-api .

# Runtime stage - Use distroless for minimal attack surface (Google's minimal image)
# Alternative: Use debian:bookworm-slim or amazonlinux:2023
FROM gcr.io/distroless/static-debian12:latest

# Copy CA certificates and binary from builder
COPY --from=builder /app/shopping-cart-api /app/shopping-cart-api

# Set working directory
WORKDIR /app

# Expose port 8080
EXPOSE 8080

# Set environment variables (can be overridden by ECS task definition)
ENV PORT=8080
ENV AWS_REGION=us-west-2
ENV DYNAMODB_TABLE_NAME=ShoppingCarts

# Use non-root user (distroless provides 'nonroot' user)
USER nonroot:nonroot

# Run the application
ENTRYPOINT ["/app/shopping-cart-api"]
