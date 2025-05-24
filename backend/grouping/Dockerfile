# Use the official Golang image as a base image
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files (assuming Docker context is backend/grouping/)
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
# If there are other .go files (e.g., service.go, models.go), they should be copied too.
# Using *.go . is a common pattern if all .go files in the directory are part of the main package.
COPY *.go .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/grouping-service .
# The above command assumes main.go (or all *.go files) are in the root of the service directory.


# Start a new stage from scratch for a lightweight image
FROM alpine:latest

# Add CA certificates, git (if needed for any runtime operations, though unlikely for compiled Go)
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/grouping-service .

# Expose port 8083 to the outside world
EXPOSE 8083

# Command to run the executable
CMD ["./grouping-service"]
