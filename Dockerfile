# Build stage
FROM golang:1.25.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install swag for swagger generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger docs
RUN /go/bin/swag init -g cmd/main.go -o docs --parseDependency --parseInternal

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/aci ./cmd/main.go

# Build the seed command
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/seed ./cmd/seed/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates postgresql-client

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/aci /app/aci
COPY --from=builder /app/bin/seed /app/seed

# Copy configuration files
COPY config.yaml /app/config.yaml
COPY db/migrations /app/db/migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["/app/aci"]

