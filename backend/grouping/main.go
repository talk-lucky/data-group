package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time" // Added for handler response

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
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

// GroupingService defines operations for calculating and retrieving groups.
type GroupingService struct {
	metadataClient      MetadataServiceClient
	orchestrationClient OrchestrationServiceClient
	db                  *sql.DB // For accessing processed entity data and storing group results
}

// NewGroupingService creates a new GroupingService.
func NewGroupingService(metadataClient MetadataServiceClient, orchestrationClient OrchestrationServiceClient, db *sql.DB) *GroupingService {
	return &GroupingService{
		metadataClient:      metadataClient,
		orchestrationClient: orchestrationClient,
		db:                  db,
	}
}

// MetadataServiceClient defines an interface for metadata service interactions.
type MetadataServiceClient interface {
	GetGroupDefinition(groupID string) (*GroupDefinition, error)
	// Add GetEntityDefinition, GetAttributeDefinitions as needed for rule evaluation
}

// HTTPMetadataClient is an HTTP implementation of MetadataServiceClient.
type HTTPMetadataClient struct {
	baseURL string
}

// NewHTTPMetadataClient creates a new HTTPMetadataClient.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	log.Printf("Initializing HTTPMetadataClient for Grouping Service with baseURL: %s", baseURL)
	return &HTTPMetadataClient{baseURL: baseURL}
}

// GroupDefinition represents a group definition from the metadata service. (Simplified)
type GroupDefinition struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	EntityID  string `json:"entity_id"`
	RulesJSON string `json:"rules_json"`
}

// GetGroupDefinition simulates fetching a group definition.
func (c *HTTPMetadataClient) GetGroupDefinition(groupID string) (*GroupDefinition, error) {
	log.Printf("HTTPMetadataClient: Mock fetching GroupDefinition for groupID: %s from %s", groupID, c.baseURL)
	// Simulate HTTP call, e.g., to fmt.Sprintf("%s/api/v1/metadata/groups/%s", c.baseURL, groupID)
	if groupID == "active_users_group" {
		return &GroupDefinition{
			ID:        groupID,
			Name:      "Active Users",
			EntityID:  "user_entity_id",
			RulesJSON: `{"condition": "AND", "rules": [{"field": "last_login", "operator": "within_last_days", "value": "30"}]}`,
		}, nil
	}
	return nil, fmt.Errorf("mock metadata error: group definition for %s not found", groupID)
}

// OrchestrationServiceClient defines an interface for orchestration service interactions.
type OrchestrationServiceClient interface {
	TriggerWorkflow(workflowID string, payload map[string]interface{}) error
}

// HTTPOrchestrationClient is an HTTP implementation of OrchestrationServiceClient.
type HTTPOrchestrationClient struct {
	baseURL string
}

// NewHTTPOrchestrationClient creates a new HTTPOrchestrationClient.
func NewHTTPOrchestrationClient(baseURL string) *HTTPOrchestrationClient {
	log.Printf("Initializing HTTPOrchestrationClient for Grouping Service with baseURL: %s", baseURL)
	return &HTTPOrchestrationClient{baseURL: baseURL}
}

// TriggerWorkflow simulates triggering a workflow.
func (c *HTTPOrchestrationClient) TriggerWorkflow(workflowID string, payload map[string]interface{}) error {
	log.Printf("HTTPOrchestrationClient: Mock triggering workflow %s with payload %v via %s", workflowID, payload, c.baseURL)
	// Simulate HTTP POST, e.g., to fmt.Sprintf("%s/api/v1/orchestration/workflows/%s/trigger", c.baseURL, workflowID)
	return nil
}

// CalculateGroup simulates calculating group members based on rules and storing them.
func (s *GroupingService) CalculateGroup(groupID string) ([]string, error) {
	log.Printf("GroupingService: Calculating group for groupID: %s", groupID)
	// 1. Fetch GroupDefinition from metadataClient
	// groupDef, err := s.metadataClient.GetGroupDefinition(groupID)
	// if err != nil { return nil, err }
	// 2. Fetch EntityDefinition and AttributeDefinitions for groupDef.EntityID
	// 3. Parse groupDef.RulesJSON
	// 4. Construct SQL query based on rules to query processed data in s.db
	//    (e.g., SELECT instance_id FROM processed_user_entity_id WHERE last_login > NOW() - INTERVAL '30 days')
	// 5. Execute query and get list of entityInstanceIDs
	// 6. Store these IDs and calculation timestamp in a dedicated table (e.g., group_calculation_results)
	// For now, returning mock data and simulating storage:
	mockInstanceIDs := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	// Simulate storing results:
	// _, err = s.db.Exec("INSERT INTO group_results (group_id, instance_ids, calculated_at) VALUES ($1, $2, $3)", groupID, pq.Array(mockInstanceIDs), time.Now().UTC())
	// if err != nil { return nil, err }

	// 7. Trigger orchestration workflow (if configured)
	// err = s.orchestrationClient.TriggerWorkflow("on_group_calculated", map[string]interface{}{"group_id": groupID, "member_count": len(mockInstanceIDs)})
	// if err != nil { log.Printf("Warning: Failed to trigger orchestration workflow for group %s: %v", groupID, err) }

	return mockInstanceIDs, nil
}

// GetGroupResults simulates retrieving pre-calculated group members.
func (s *GroupingService) GetGroupResults(groupID string) ([]string, time.Time, error) {
	log.Printf("GroupingService: Getting results for groupID: %s", groupID)
	// Simulate fetching from a 'group_calculation_results' table
	// var instanceIDs []string
	// var calculatedAt time.Time
	// err := s.db.QueryRow("SELECT instance_ids, calculated_at FROM group_results WHERE group_id = $1 ORDER BY calculated_at DESC LIMIT 1", groupID).Scan(pq.Array(&instanceIDs), &calculatedAt)
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) { return nil, time.Time{}, nil } // No results yet
	// 	return nil, time.Time{}, err
	// }
	// return instanceIDs, calculatedAt, nil
	return []string{uuid.NewString()}, time.Now().UTC().Add(-5 * time.Minute), nil // Mock data
}

// --- Main Application ---
func main() {
	// --- Configuration ---
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8090") // Updated default
	orchestrationServiceURL := getEnv("ORCHESTRATION_SERVICE_URL", "http://localhost:8084")
	groupingServicePort := getEnv("GROUPING_SERVICE_PORT", "8083") // Standardized port variable

	log.Printf("Configuration:")
	log.Printf("  METADATA_SERVICE_URL: %s", metadataServiceURL)
	log.Printf("  ORCHESTRATION_SERVICE_URL: %s", orchestrationServiceURL)
	log.Printf("  GROUPING_SERVICE_PORT: %s", groupingServicePort)

	// --- Database Connection for Processed Entities (remains the same) ---
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
		log.Fatalf("Failed to connect to PostgreSQL for processed_entities: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping PostgreSQL for processed_entities: %v", err)
	}
	log.Println("Successfully connected to the database for Grouping Service.")

	// --- Initialize Services ---
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)
	orchestrationClient := NewHTTPOrchestrationClient(orchestrationServiceURL)
	groupingService := NewGroupingService(metadataClient, orchestrationClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()
	// API Gateway expects /api/v1/groups/* to be handled by this service.
	// So, the service should listen on these paths.
	v1 := router.Group("/api/v1")
	{
		groupRoutes := v1.Group("/groups")
		{
			// POST /api/v1/groups/{group_id}/calculate
			groupRoutes.POST("/:group_id/calculate", calculateGroupHandler(groupingService))
			// GET /api/v1/groups/{group_id}/results
			groupRoutes.GET("/:group_id/results", getGroupResultsHandler(groupingService))
		}
	}
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Start the server
	listenAddr := fmt.Sprintf(":%s", groupingServicePort)
	log.Printf("Starting Grouping Service on %s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start Grouping Service: %v", err)
	}
}

// calculateGroupHandler creates a gin.HandlerFunc that uses the GroupingService
func calculateGroupHandler(service *GroupingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("group_id")
		log.Printf("Received group calculation request for group_id: %s", groupID)

		entityInstanceIDs, err := service.CalculateGroup(groupID)
		if err != nil {
			log.Printf("Error calculating group for groupID %s: %v", groupID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":  "Error calculating group",
				"group_id": groupID,
				"error":    err.Error(),
			})
			return
		}

		log.Printf("Successfully calculated group %s. Found %d matching entity instances.", groupID, len(entityInstanceIDs))
		c.JSON(http.StatusOK, gin.H{
			"message":       "Group calculation successful and results stored",
			"group_id":      groupID,
			"member_count":  len(entityInstanceIDs),
			"calculated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// getGroupResultsHandler creates a gin.HandlerFunc to retrieve group calculation results.
func getGroupResultsHandler(service *GroupingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("group_id")
		log.Printf("Received request for group results for group_id: %s", groupID)

		instanceIDs, calculatedAt, err := service.GetGroupResults(groupID)
		if err != nil {
			log.Printf("Error getting group results for groupID %s: %v", groupID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":  "Error retrieving group results",
				"group_id": groupID,
				"error":    err.Error(),
			})
			return
		}

		if len(instanceIDs) == 0 && calculatedAt.IsZero() { // Check if results are genuinely empty
			log.Printf("No results found for groupID %s.", groupID)
			c.JSON(http.StatusNotFound, gin.H{
				"message":  "No results found for this group",
				"group_id": groupID,
			})
			return
		}

		log.Printf("Successfully retrieved %d results for groupID %s, calculated at %s", len(instanceIDs), groupID, calculatedAt.Format(time.RFC3339))
		c.JSON(http.StatusOK, gin.H{
			"group_id":      groupID,
			"member_ids":    instanceIDs,
			"calculated_at": calculatedAt.Format(time.RFC3339),
			"member_count":  len(instanceIDs),
		})
	}
}

// Added uuid import for placeholder data
type uuid struct{}
func (uuid) NewString() string { return "mock-uuid-string" }
// This is a very basic mock. In a real scenario, use "github.com/google/uuid".
// var uuid = 실제UUID패키지.New() // 예시
// For the purpose of this exercise, direct import is avoided to keep it self-contained for copy-pasting if needed.
// However, a real project should use the actual library.
func NewString() string { // Helper to mimic uuid.NewString()
	return "mock-uuid-" + fmt.Sprintf("%d", time.Now().UnixNano())
}
