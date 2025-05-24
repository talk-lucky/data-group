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

func main() {
	// --- Database Connection ---
	// Read DB connection details from environment variables or use defaults
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "admin") // Default user
	dbPassword := getEnv("DB_PASSWORD", "password") // Default password
	dbName := getEnv("DB_NAME", "metadata_db")    // Default database name
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
	// Metadata service client (assuming it runs on localhost:8080)
	// This URL should ideally come from configuration.
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8080")
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)

	processingSvc := NewProcessingService(metadataClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		v1.POST("/process", processDataHandler(processingSvc))
	}

	// Start the server on port 8082
	serverPort := ":8082"
	log.Printf("Starting Data Processing Service on port %s", serverPort)
	if err := router.Run(serverPort); err != nil {
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
		if req.EntityTypeName == "" {
			// Attempt to fetch EntityTypeName from DataSourceConfig if not provided
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

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
