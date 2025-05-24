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

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}

// --- Placeholder types and functions (assuming these are defined elsewhere or are simplified) ---

// OrchestrationService defines operations for managing and triggering workflows.
type OrchestrationService struct {
	js             nats.JetStreamContext
	metadataClient MetadataServiceClient
	groupingClient GroupingServiceClient
	db             *sql.DB // For accessing entity instance data
}

// NewOrchestrationService creates a new OrchestrationService.
func NewOrchestrationService(js nats.JetStreamContext, metadataClient MetadataServiceClient, groupingClient GroupingServiceClient, db *sql.DB) *OrchestrationService {
	return &OrchestrationService{
		js:             js,
		metadataClient: metadataClient,
		groupingClient: groupingClient,
		db:             db,
	}
}

// MetadataServiceClient defines an interface for metadata service interactions.
type MetadataServiceClient interface {
	GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error)
	GetActionTemplate(templateID string) (*ActionTemplate, error)
	// Add other methods as needed
}

// HTTPMetadataClient is an HTTP implementation of MetadataServiceClient.
type HTTPMetadataClient struct {
	baseURL string
}

// NewHTTPMetadataClient creates a new HTTPMetadataClient.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	log.Printf("Initializing HTTPMetadataClient for Orchestration Service with baseURL: %s", baseURL)
	return &HTTPMetadataClient{baseURL: baseURL}
}

// WorkflowDefinition represents a workflow definition from the metadata service. (Simplified)
type WorkflowDefinition struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ActionSequenceJSON string `json:"action_sequence_json"` // JSON array of action steps
}

// ActionTemplate represents an action template from the metadata service. (Simplified)
type ActionTemplate struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ActionType      string `json:"action_type"` // e.g., "webhook", "nats_publish"
	TemplateContent string `json:"template_content"`
}

// GetWorkflowDefinition simulates fetching a workflow definition.
func (c *HTTPMetadataClient) GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error) {
	log.Printf("HTTPMetadataClient: Mock fetching WorkflowDefinition for workflowID: %s from %s", workflowID, c.baseURL)
	// Simulate HTTP call, e.g., to fmt.Sprintf("%s/api/v1/metadata/workflows/%s", c.baseURL, workflowID)
	if workflowID == "sample_workflow" {
		return &WorkflowDefinition{
			ID:                 workflowID,
			Name:               "Sample Notification Workflow",
			ActionSequenceJSON: `[{"action_template_id": "notify_user_template", "params": {"user_id_field": "customer_id"}}]}`,
		}, nil
	}
	return nil, fmt.Errorf("mock metadata error: workflow definition for %s not found", workflowID)
}

// GetActionTemplate simulates fetching an action template.
func (c *HTTPMetadataClient) GetActionTemplate(templateID string) (*ActionTemplate, error) {
	log.Printf("HTTPMetadataClient: Mock fetching ActionTemplate for templateID: %s from %s", templateID, c.baseURL)
	if templateID == "notify_user_template" {
		return &ActionTemplate{
			ID:              templateID,
			Name:            "Notify User Template",
			ActionType:      "nats_publish",
			TemplateContent: `{"subject": "user.notification.{{entity_type}}", "message": "Notification for {{entity_id}}: {{event_type}}"}`,
		}, nil
	}
	return nil, fmt.Errorf("mock metadata error: action template for %s not found", templateID)
}

// GroupingServiceClient defines an interface for grouping service interactions.
type GroupingServiceClient interface {
	GetGroupResults(groupID string) ([]string, time.Time, error) // Returns member IDs and calculation time
}

// HTTPGroupingServiceClient is an HTTP implementation of GroupingServiceClient.
type HTTPGroupingServiceClient struct {
	baseURL string
}

// NewHTTPGroupingServiceClient creates a new HTTPGroupingServiceClient.
func NewHTTPGroupingServiceClient(baseURL string) *HTTPGroupingServiceClient {
	log.Printf("Initializing HTTPGroupingServiceClient for Orchestration Service with baseURL: %s", baseURL)
	return &HTTPGroupingServiceClient{baseURL: baseURL}
}

// GetGroupResults simulates fetching group results.
func (c *HTTPGroupingServiceClient) GetGroupResults(groupID string) ([]string, time.Time, error) {
	log.Printf("HTTPGroupingServiceClient: Mock fetching results for groupID: %s from %s", groupID, c.baseURL)
	// Simulate HTTP call, e.g., to fmt.Sprintf("%s/api/v1/groups/%s/results", c.baseURL, groupID)
	if groupID == "active_users_group" {
		return []string{"user123", "user456"}, time.Now().Add(-10 * time.Minute), nil
	}
	return nil, time.Time{}, fmt.Errorf("mock grouping error: results for group %s not found", groupID)
}

// TriggerWorkflow simulates the logic for triggering a workflow.
func (s *OrchestrationService) TriggerWorkflow(workflowID string) error {
	log.Printf("OrchestrationService: Triggering workflowID: %s", workflowID)
	// 1. Fetch WorkflowDefinition from metadataClient
	// wfDef, err := s.metadataClient.GetWorkflowDefinition(workflowID)
	// if err != nil { return err }
	// 2. Parse wfDef.ActionSequenceJSON
	// 3. For each action in sequence:
	//    a. Fetch ActionTemplate from metadataClient
	//    b. Populate template with context (e.g., entity data from s.db, trigger payload)
	//    c. Execute action (e.g., publish to NATS using s.js, call webhook)
	log.Printf("OrchestrationService: Workflow %s steps would be executed here (mock).", workflowID)
	// Example: Publishing a NATS message
	// _, err = s.js.Publish("orchestration.events", []byte(fmt.Sprintf(`{"workflow_id": "%s", "status": "completed"}`, workflowID)))
	// if err != nil { return err }
	return nil
}

// --- Main Application ---
func main() {
	// --- Configuration ---
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8090") // Updated default
	groupingServiceURL := getEnv("GROUPING_SERVICE_URL", "http://localhost:8083")
	orchestrationServicePort := getEnv("ORCHESTRATION_SERVICE_PORT", "8084") // Standardized port variable
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

	log.Printf("Configuration:")
	log.Printf("  METADATA_SERVICE_URL: %s", metadataServiceURL)
	log.Printf("  GROUPING_SERVICE_URL: %s", groupingServiceURL)
	log.Printf("  ORCHESTRATION_SERVICE_PORT: %s", orchestrationServicePort)
	log.Printf("  NATS_URL: %s", natsURL)


	// --- Database Connection (remains the same) ---
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "admin")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "metadata_db")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	log.Println("Successfully connected to the database for Orchestration Service.")

	// --- NATS Connection (remains the same) ---
	nc, err := nats.Connect(natsURL, nats.Timeout(10*time.Second), nats.RetryOnFailedConnect(true), nats.MaxReconnects(5), nats.ReconnectWait(time.Second))
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Successfully connected to NATS at %s", natsURL)

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}
	log.Println("Successfully created JetStream context.")

	// --- Initialize Service Clients ---
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)
	groupingClient := NewHTTPGroupingServiceClient(groupingServiceURL)

	// --- Initialize OrchestrationService ---
	orchestrationSvc := NewOrchestrationService(js, metadataClient, groupingClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()
	// API Gateway expects /api/v1/orchestration/* to be handled by this service.
	v1 := router.Group("/api/v1/orchestration")
	{
		v1.POST("/trigger/workflow/:workflow_id", triggerWorkflowHandler(orchestrationSvc))
		// NATS listeners would be set up here as goroutines, not HTTP routes
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Start the server
	listenAddr := fmt.Sprintf(":%s", orchestrationServicePort)
	log.Printf("Starting Orchestration Service HTTP API on %s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start Orchestration Service HTTP API: %v", err)
	}
}

// triggerWorkflowHandler creates a gin.HandlerFunc to manually trigger a workflow.
func triggerWorkflowHandler(service *OrchestrationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		workflowID := c.Param("workflow_id")
		log.Printf("HTTP trigger received for workflow_id: %s", workflowID)

		// In a real scenario, you might parse a payload from c.Request.Body
		// For now, TriggerWorkflow is simplified to only take workflowID
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
