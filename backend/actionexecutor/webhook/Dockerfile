# Use the official Golang image as a base image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app/backend/actionexecutor/webhook

# Copy go mod and sum files
COPY backend/actionexecutor/webhook/go.mod backend/actionexecutor/webhook/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
COPY backend/actionexecutor/webhook/models.go .
COPY backend/actionexecutor/webhook/service.go .
COPY backend/actionexecutor/webhook/main.go .

# Build the Go app
# Ensure all necessary .go files are included in the build command
RUN CGO_ENABLED=0 GOOS=linux go build -o /webhook-action-executor main.go models.go service.go


# Start a new stage from scratch for a lightweight image
FROM alpine:latest

# Add CA certificates (important for making HTTPS calls if webhooks target HTTPS endpoints)
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /webhook-action-executor .

# This service does not expose any ports directly as it's a NATS consumer.
# Environment variables for NATS_URL etc. will be configured during deployment.

# Command to run the executable
CMD ["./webhook-action-executor"]
