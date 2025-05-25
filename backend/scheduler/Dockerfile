# Builder Stage
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
# These paths assume the Docker build context is the 'backend/scheduler' directory.
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy the source code into the container
# This copies all .go files from the 'backend/scheduler' directory into /app in the container.
COPY *.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/scheduler-service .

# Runtime Stage
FROM alpine:latest

# Add CA certificates for HTTPS outbound requests (e.g., to metadata service)
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the builder stage
COPY --from=builder /app/scheduler-service .

# No EXPOSE needed if the service itself doesn't listen on a port (e.g., only a cron runner)
# If an HTTP health check or management API is added later on a port (e.g., 8085), 
# then uncomment and set the EXPOSE directive:
# EXPOSE 8085

# Command to run the executable
CMD ["./scheduler-service"]
