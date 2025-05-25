package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}

func main() {
	log.Println("Starting Scheduler Service...")

	// Configuration (placeholders for now, actual clients will be initialized later)
	// schedulerPort := getEnv("SCHEDULER_SERVICE_PORT", "8085") // If exposing an API
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8090") // For fetching schedules
	// ingestionServiceURL := getEnv("INGESTION_SERVICE_URL", "http://localhost:8081") // For triggering ingestion tasks
	// Add other service URLs as needed (e.g., grouping, orchestration)

	log.Printf("Configuration:")
	log.Printf("  METADATA_SERVICE_URL: %s", metadataServiceURL)
	// log.Printf("  INGESTION_SERVICE_URL: %s", ingestionServiceURL)
	// log.Printf("  SCHEDULER_SERVICE_PORT: %s", schedulerPort)


	// --- Initialize Service Clients (Actual HTTP clients to be implemented later) ---
	// For now, using nil or placeholder implementations for the service constructor.
	// In a real scenario, you'd initialize actual HTTP clients here.
	var metaClient MetadataServiceClient   // Placeholder, will be a real client
	var ingestClient IngestionServiceClient // Placeholder
	var groupingClient GroupingServiceClient // Placeholder
	var orchestrationClient OrchestrationServiceClient // Placeholder

	// --- Initialize SchedulerService ---
	schedulerService := NewSchedulerService(metaClient, ingestClient, groupingClient, orchestrationClient)

	// --- Start Scheduler ---
	// The Start method will load schedules and start the cron runner.
	// Running it in a goroutine allows main to handle shutdown signals.
	go func() {
		if err := schedulerService.Start(); err != nil {
			log.Fatalf("Failed to start Scheduler Service: %v", err)
		}
	}()


	// --- Graceful Shutdown Handling ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	<-quit // Block until a signal is received
	
	log.Println("Scheduler Service is shutting down...")

	// Create a context with timeout for shutdown
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Increased timeout
	// defer cancel()

	// Call Stop on the scheduler service
	schedulerService.Stop()
	
	// Any other cleanup tasks can go here

	log.Println("Scheduler Service stopped gracefully.")
}

// Placeholder client interfaces - these will be fleshed out or replaced
// by actual client implementations when tasks are integrated.
// For now, they satisfy the NewSchedulerService signature.

// MetadataServiceClient placeholder (actual implementation will be an HTTP client)
// type MetadataServiceClient interface {
// 	ListScheduleDefinitions() ([]ScheduleDefinition, error)
// 	// Potentially other methods like GetWorkflowDefinition if scheduler triggers workflows directly
// }

// IngestionServiceClient placeholder
// type IngestionServiceClient interface {
// 	TriggerIngestion(sourceID string) error
// }

// GroupingServiceClient placeholder
// type GroupingServiceClient interface {
// 	CalculateGroup(groupID string) error
// }

// OrchestrationServiceClient placeholder
// type OrchestrationServiceClient interface {
// 	TriggerWorkflow(workflowID string) error
// }

// These interfaces are better defined in service.go where SchedulerService uses them.
// For now, main.go can proceed with nil clients for the constructor call.
// The actual clients will be initialized in main.go and passed to NewSchedulerService.
// The service.go file will define the interfaces it expects.
// This allows main.go to not depend on the full interface definitions yet.
// We will use the interfaces defined in service.go for the variables above.
// This means service.go needs to be created before this main.go can fully compile
// without type errors for metaClient, ingestClient etc.
// For the boilerplate generation, this structure is okay.
// The actual client initialization will be:
// metaClient = NewHTTPMetadataClient(metadataServiceURL) etc.
// For now, we pass nil which NewSchedulerService should handle or service.go's NewSchedulerService
// should take potentially nil clients if it can operate without them initially (e.g. Start populates them).
// However, the current plan for service.go's NewSchedulerService takes these as arguments.
// So, for now, they are declared as their interface types and will be nil.
// This means service.go needs to be robust to nil clients if Start is called immediately.
// Better: Define the interfaces in service.go and have main.go use them.
// For this subtask, we'll assume the interfaces are defined in service.go and use them here.
// (The actual client initializations will be added in a later subtask).
