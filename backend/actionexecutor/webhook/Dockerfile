# Use the official Golang image as a base image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files (assuming Docker context is backend/actionexecutor/webhook/)
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
# This will copy main.go, models.go, service.go, etc. from the service's root directory.
COPY *.go .

# Build the Go app
# The '.' argument tells go build to compile the package in the current directory.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/webhook-action-executor .


# Start a new stage from scratch for a lightweight image
FROM alpine:latest

# Add CA certificates (important for making HTTPS calls if webhooks target HTTPS endpoints)
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/webhook-action-executor .

# This service does not expose any ports directly as it's a NATS consumer.
# Environment variables for NATS_URL etc. will be configured during deployment.

# Command to run the executable
CMD ["./webhook-action-executor"]
