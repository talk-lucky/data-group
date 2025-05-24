package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	_ "github.com/lib/pq" // For database/sql PostgreSQL driver
)

func main() {
	// --- Database Connection for Processed Entities ---
	// (Needed by OrchestrationService to fetch entity instance data)
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "admin")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "metadata_db") // Assuming processed_entities is in the same DB
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL for processed_entities: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping PostgreSQL for processed_entities: %v", err)
	}
	log.Println("Successfully connected to the database (for processed_entities).")


	// --- NATS Connection ---
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL, nats.Timeout(10*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(5), nats.ReconnectWait(time.Second))
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


	// --- Initialize Service Clients ---
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8080")
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)

	groupingServiceURL := getEnv("GROUPING_SERVICE_URL", "http://localhost:8083")
	groupingClient := NewHTTPGroupingServiceClient(groupingServiceURL)

	// --- Initialize OrchestrationService ---
	orchestrationSvc := NewOrchestrationService(js, metadataClient, groupingClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()
	v1 := router.Group("/api/v1/orchestration")
	{
		// Endpoint to manually trigger a workflow
		v1.POST("/trigger/workflow/:workflow_id", triggerWorkflowHandler(orchestrationSvc))

		// TODO: Add listener for NATS messages (e.g., group update events)
		// This would likely involve setting up a NATS subscription in a goroutine
		// that then calls orchestrationSvc.TriggerWorkflow()
	}

	// Start the server
	serverPort := getEnv("PORT", "8084") // Default to 8084
	log.Printf("Starting Orchestration Service HTTP API on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start Orchestration Service HTTP API: %v", err)
	}
}

// triggerWorkflowHandler creates a gin.HandlerFunc to manually trigger a workflow.
func triggerWorkflowHandler(service *OrchestrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		workflowID := c.Param("workflow_id")
		log.Printf("HTTP trigger received for workflow_id: %s", workflowID)

		if err := service.TriggerWorkflow(workflowID); err != nil {
			log.Printf("Error triggering workflow %s: %v", workflowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":     "Error triggering workflow",
				"workflow_id": workflowID,
				"error":       err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Workflow triggered successfully",
			"workflow_id": workflowID,
		})
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using fallback: %s", key, fallback)
	return fallback
}
