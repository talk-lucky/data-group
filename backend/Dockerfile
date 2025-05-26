# Builder Stage
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
# These paths assume the Docker build context is the 'backend' directory.
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy the metadata service code
# This makes the "example.com/project/metadata" import work,
# assuming go.mod has a replace directive or the module structure aligns.
COPY metadata/ ./metadata/

# Copy the API Gateway's main.go
COPY main.go .

# Build the Go app (API Gateway)
# The '.' argument tells go build to compile the package in the current directory (main.go).
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api-gateway .

# Runtime Stage
FROM alpine:latest

# Add CA certificates for HTTPS outbound requests (e.g., if proxying to HTTPS services, or for metadata DB SSL)
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the builder stage
COPY --from=builder /app/api-gateway .

# Expose port 8080 to the outside world (default for the API Gateway)
EXPOSE 8080

# Command to run the executable
CMD ["./api-gateway"]
