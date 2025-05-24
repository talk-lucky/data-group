package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	// --- NATS Connection ---
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL, nats.Timeout(10*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1), nats.ReconnectWait(3*time.Second))
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Successfully connected to NATS at %s", natsURL)

	// --- JetStream Context ---
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}
	log.Println("Successfully created JetStream context.")

	// --- Initialize WebhookExecutorService ---
	webhookService := NewWebhookExecutorService(js)

	// --- Start Consuming NATS messages ---
	// Run StartConsuming in a goroutine so main can handle shutdown signals
	go func() {
		if err := webhookService.StartConsuming(); err != nil {
			log.Fatalf("Error starting NATS consumer: %v", err)
		}
	}()
	log.Println("Webhook Executor Service started and consuming messages.")

	// --- Graceful Shutdown Handling ---
	// Wait for a signal to shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received, exiting...")
	// Perform any cleanup here if needed, e.g., nc.Drain() for NATS
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using fallback: %s", key, fallback)
	return fallback
}
