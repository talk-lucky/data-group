FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ensure main.go is the entry point for the build
RUN go build -o /email-executor main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /email-executor /email-executor
# Add any necessary CA certificates if your SMTP server uses TLS and needs them
# RUN apk --no-cache add ca-certificates
ENV NATS_URL="nats://nats:4222"
ENV SMTP_HOST="" # To be configured
ENV SMTP_PORT="587"
ENV SMTP_USER=""
ENV SMTP_PASS=""
ENV DEFAULT_FROM_EMAIL="noreply@example.com"
ENTRYPOINT ["/email-executor"]
