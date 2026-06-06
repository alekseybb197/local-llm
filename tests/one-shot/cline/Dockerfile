FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/main .

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set environment variables
ENV OAUTH_CLIENT_ID=""
ENV OAUTH_CLIENT_SECRET=""
ENV OAUTH_AUTH_SERVER_URL="http://localhost:8000"
ENV OAUTH_REDIRECT_URI="http://localhost:8080/oauth/callback"
ENV LOG_LEVEL="info"

# Run the application
CMD ["./main"]
