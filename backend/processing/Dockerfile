# Builder Stage
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
# These paths assume the Docker build context is the 'backend/processing' directory.
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy the source code into the container
# This copies all .go files from the 'backend/processing' directory into /app in the container.
COPY *.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/processing-service .

# Runtime Stage
FROM alpine:latest

# Add CA certificates for HTTPS outbound requests
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the builder stage
COPY --from=builder /app/processing-service .

# Expose port 8082 to the outside world (default for the processing service)
EXPOSE 8082

# Command to run the executable
CMD ["./processing-service"]
