package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os" // For environment variables

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

// --- Placeholder types and functions (assuming these are defined elsewhere or are simplified for this context) ---

// ProcessingService orchestrates data processing.
type ProcessingService struct {
	metadataClient MetadataServiceClient
	db             *sql.DB // Database connection for storing processed data
}

// NewProcessingService creates a new ProcessingService.
func NewProcessingService(metadataClient MetadataServiceClient, db *sql.DB) *ProcessingService {
	return &ProcessingService{
		metadataClient: metadataClient,
		db:             db,
	}
}

// ProcessDataRequest is the expected input for the processing endpoint.
type ProcessDataRequest struct {
	SourceID       string                   `json:"source_id"`
	EntityTypeName string                   `json:"entity_type_name,omitempty"` // Optional: if not provided, try to derive from SourceID
	RawData        []map[string]interface{} `json:"raw_data"`
}

// MetadataServiceClient defines the interface for interacting with the metadata service.
type MetadataServiceClient interface {
	GetDataSourceConfig(sourceID string) (*DataSourceConfig, error)
	GetEntityDefinition(entityID string) (*EntityDefinition, error)
	// Add other methods as needed, e.g., GetAttributeDefinitions(entityID string)
}

// HTTPMetadataClient is an HTTP implementation of MetadataServiceClient.
type HTTPMetadataClient struct {
	baseURL string
}

// NewHTTPMetadataClient creates a new HTTPMetadataClient.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	log.Printf("Initializing HTTPMetadataClient for Processing Service with baseURL: %s", baseURL)
	return &HTTPMetadataClient{baseURL: baseURL}
}

// DataSourceConfig represents the configuration for a data source. (Simplified)
type DataSourceConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	EntityID string `json:"entity_id,omitempty"` // Associated EntityDefinition ID
}

// EntityDefinition represents a metadata entity. (Simplified)
type EntityDefinition struct {
	ID   string `json:"id"`
	Name string `json:"name"` // This would be the EntityTypeName
}

// GetDataSourceConfig simulates fetching DataSourceConfig from metadata service.
func (c *HTTPMetadataClient) GetDataSourceConfig(sourceID string) (*DataSourceConfig, error) {
	// In a real scenario, this would make an HTTP GET request to:
	// fmt.Sprintf("%s/api/v1/metadata/datasources/%s", c.baseURL, sourceID)
	log.Printf("HTTPMetadataClient: Mock fetching DataSourceConfig for SourceID: %s from %s", sourceID, c.baseURL)
	if sourceID == "known_source_with_entity" {
		return &DataSourceConfig{ID: sourceID, Name: "Known Source", EntityID: "entity123"}, nil
	}
	if sourceID == "known_source_no_entity" {
		return &DataSourceConfig{ID: sourceID, Name: "Known Source No Entity", EntityID: ""}, nil
	}
	return nil, fmt.Errorf("mock metadata service error: data source config for %s not found", sourceID)
}

// GetEntityDefinition simulates fetching EntityDefinition from metadata service.
func (c *HTTPMetadataClient) GetEntityDefinition(entityID string) (*EntityDefinition, error) {
	// In a real scenario, this would make an HTTP GET request to:
	// fmt.Sprintf("%s/api/v1/metadata/entities/%s", c.baseURL, entityID)
	log.Printf("HTTPMetadataClient: Mock fetching EntityDefinition for EntityID: %s from %s", entityID, c.baseURL)
	if entityID == "entity123" {
		return &EntityDefinition{ID: entityID, Name: "Order"}, nil
	}
	return nil, fmt.Errorf("mock metadata service error: entity definition for %s not found", entityID)
}

// ProcessAndStoreData is a placeholder for the actual data processing and storage logic.
func (s *ProcessingService) ProcessAndStoreData(sourceID, entityTypeName string, rawData []map[string]interface{}) (int, error) {
	log.Printf("ProcessingService: Processing %d records for SourceID '%s', EntityType '%s'", len(rawData), sourceID, entityTypeName)
	// 1. Fetch attribute definitions for entityTypeName from metadata service
	// 2. For each record in rawData:
	//    a. Validate and transform based on attribute definitions
	//    b. Store in the appropriate table in 's.db' (e.g., a table named after entityTypeName)
	// This is a highly simplified placeholder.
	// For now, just log and return the count of records.
	for i, record := range rawData {
		log.Printf("Processing record %d for %s: %v (actual storage not implemented)", i+1, sourceID, record)
		// Example: _, err := s.db.Exec(fmt.Sprintf("INSERT INTO %s_data (...) VALUES (...)", entityTypeName), ...record fields...)
		// if err != nil { return 0, err }
	}
	return len(rawData), nil
}

// --- Main Application ---
func main() {
	// --- Configuration ---
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8090") // Updated default
	processingServicePort := getEnv("PROCESSING_SERVICE_PORT", "8082")

	log.Printf("Configuration:")
	log.Printf("  METADATA_SERVICE_URL: %s", metadataServiceURL)
	log.Printf("  PROCESSING_SERVICE_PORT: %s", processingServicePort)


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
	log.Println("Successfully connected to the database for processing service.")

	// --- Initialize Services ---
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)
	processingSvc := NewProcessingService(metadataClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()

	// The gateway is expected to rewrite /api/v1/processing/* to /api/v1/process/*
	// So this service should listen on /api/v1/process
	apiV1 := router.Group("/api/v1")
	{
		processRoutes := apiV1.Group("/process")
		{
			processRoutes.POST("", processDataHandler(processingSvc)) // Changed from /process to "" as group is /process
		}
	}
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})


	// Start the server
	listenAddr := fmt.Sprintf(":%s", processingServicePort)
	log.Printf("Starting Data Processing Service on %s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start Data Processing Service: %v", err)
	}
}

// processDataHandler creates a gin.HandlerFunc that uses the ProcessingService.
func processDataHandler(service *ProcessingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ProcessDataRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Error binding JSON for process request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
			return
		}

		if req.SourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_id is required"})
			return
		}
		
		// Deriving EntityTypeName if not provided
		if req.EntityTypeName == "" {
			log.Printf("EntityTypeName not provided in request for SourceID %s, attempting to fetch from DataSourceConfig's EntityID", req.SourceID)
			dsConfig, err := service.metadataClient.GetDataSourceConfig(req.SourceID)
			if err != nil {
				log.Printf("Failed to fetch DataSourceConfig for SourceID %s to determine EntityTypeName: %v", req.SourceID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to fetch data source config to determine entity type: %v", err)})
				return
			}
			if dsConfig.EntityID == "" {
				log.Printf("DataSourceConfig for SourceID %s does not have an EntityID associated.", req.SourceID)
				c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type_name is required and could not be derived from data source config (missing EntityID)"})
				return
			}
			entityDef, err := service.metadataClient.GetEntityDefinition(dsConfig.EntityID)
			if err != nil {
				log.Printf("Failed to fetch EntityDefinition for EntityID %s (from SourceID %s): %v", dsConfig.EntityID, req.SourceID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to fetch entity definition for EntityID %s: %v", dsConfig.EntityID, err)})
				return
			}
			req.EntityTypeName = entityDef.Name
			log.Printf("Successfully derived EntityTypeName '%s' for SourceID %s from DataSourceConfig.EntityID %s", req.EntityTypeName, req.SourceID, dsConfig.EntityID)
		}


		if len(req.RawData) == 0 {
			log.Printf("No raw data provided in process request for SourceID: %s", req.SourceID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "raw_data cannot be empty"})
			return
		}

		log.Printf("Processing request received for SourceID: %s, EntityType: %s, Record count: %d",
			req.SourceID, req.EntityTypeName, len(req.RawData))

		processedCount, err := service.ProcessAndStoreData(req.SourceID, req.EntityTypeName, req.RawData)
		if err != nil {
			log.Printf("Error processing data for SourceID %s: %v", req.SourceID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":   "Failed to process data",
				"source_id": req.SourceID,
				"error":     err.Error(),
			})
			return
		}

		log.Printf("Successfully processed %d records for SourceID: %s", processedCount, req.SourceID)
		c.JSON(http.StatusOK, gin.H{
			"message":            "Data processed successfully",
			"source_id":          req.SourceID,
			"entity_type_name":   req.EntityTypeName,
			"records_received":   len(req.RawData),
			"records_processed":  processedCount,
		})
	}
}
