# Use the official Golang image as a base image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app/backend/grouping

# Copy go mod and sum files
COPY backend/grouping/go.mod backend/grouping/go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY backend/grouping/main.go .
# If service.go was to contain actual service logic and be a separate file:
# COPY backend/grouping/service.go .

# Build the Go app
# Make sure the output file name matches what will be used in the runtime stage
RUN CGO_ENABLED=0 GOOS=linux go build -o /grouping-service main.go
# If service.go was used and part of the build:
# RUN CGO_ENABLED=0 GOOS=linux go build -o /grouping-service main.go service.go


# Start a new stage from scratch for a lightweight image
FROM alpine:latest

# Add CA certificates
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /grouping-service .

# Expose port 8083 to the outside world
EXPOSE 8083

# Command to run the executable
CMD ["./grouping-service"]
