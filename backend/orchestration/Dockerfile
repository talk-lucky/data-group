# Use the official Golang image as a base image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app/backend/orchestration

# Copy go mod and sum files
COPY backend/orchestration/go.mod backend/orchestration/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
# Copy models.go, service.go, and main.go
COPY backend/orchestration/models.go .
COPY backend/orchestration/service.go .
COPY backend/orchestration/main.go .


# Build the Go app
# Ensure all necessary .go files are included in the build command if they are in the same package
RUN CGO_ENABLED=0 GOOS=linux go build -o /orchestration-service main.go models.go service.go


# Start a new stage from scratch for a lightweight image
FROM alpine:latest

# Add CA certificates
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /orchestration-service .

# Expose port 8084 to the outside world (or whatever port is configured in main.go)
EXPOSE 8084

# Command to run the executable
CMD ["./orchestration-service"]
