# Dockerfile (在src目录下的版本)
FROM --platform=linux/amd64 golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# 修正：go.mod和Dockerfile在同一目录(src/)
COPY go.mod go.sum ./
RUN go mod download

# 修正：main.go也在同一目录
COPY main.go .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o shopping-cart main.go

# Final stage
FROM --platform=linux/amd64 alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/shopping-cart .
# 修正：db是当前目录的子目录
COPY db/setup.sql ./db/

RUN adduser -D -u 1001 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

CMD ["./shopping-cart"]