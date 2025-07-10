# Build stage
FROM golang:1.21-alpine AS builder

# Install git and other dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o harbinger ./cmd

# Final stage
FROM alpine:latest

# Install git (required for the application to work)
RUN apk --no-cache add git ca-certificates

# Create non-root user
RUN addgroup -g 1000 harbinger && \
    adduser -D -u 1000 -G harbinger harbinger

# Copy binary from builder
COPY --from=builder /app/harbinger /usr/local/bin/harbinger

# Switch to non-root user
USER harbinger

# Set working directory
WORKDIR /workspace

# Entry point
ENTRYPOINT ["harbinger"]
CMD ["monitor"]