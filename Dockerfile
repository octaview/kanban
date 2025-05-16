# Build stage
FROM golang:1.24-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kanban ./cmd/server/main.go

# Final stage
FROM alpine:3.21

# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/kanban .

# Create directories for any necessary files
RUN mkdir -p /app/data

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./kanban"]