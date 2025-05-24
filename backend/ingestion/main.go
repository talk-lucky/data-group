package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Define API v1 group
	v1 := router.Group("/api/v1")
	{
		ingestRoutes := v1.Group("/ingest")
		{
			// Placeholder for triggering ingestion
			ingestRoutes.POST("/trigger/:source_id", triggerIngestionHandler)
		}
	}

	// Initialize Metadata Service Client
	// The metadata service is assumed to be running at http://localhost:8080
	// This URL should ideally come from configuration.
	metadataClient := NewHTTPMetadataClient("http://localhost:8080")

	// Initialize Processing Service Client
	// The processing service is assumed to be running at http://localhost:8082
	// This URL should ideally come from configuration.
	processingClient := NewHTTPProcessingServiceClient("http://localhost:8082")

	// Initialize IngestionService with both clients
	ingestionSvc := NewIngestionService(metadataClient, processingClient)

	// Define API v1 group (ensure this is not re-declaring 'v1' if it was declared before for router setup)
	// If router was already configured with v1, this might need adjustment or this is the primary setup.
	// Assuming 'v1' is freshly declared here for these routes.
	apiV1 := router.Group("/api/v1") // Use a different name to avoid conflict if v1 already exists
	{
		ingestRoutes := apiV1.Group("/ingest")
		{
			ingestRoutes.POST("/trigger/:source_id", triggerIngestionHandler(ingestionSvc))
		}
	}

	// Start the server on port 8081
	port := ":8081"
	log.Printf("Starting Data Ingestion Service on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start Data Ingestion Service: %v", err)
	}
}

// triggerIngestionHandler creates a gin.HandlerFunc that uses the IngestionService.
func triggerIngestionHandler(service *IngestionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sourceID := c.Param("source_id")
		log.Printf("Received trigger for data source ID: %s", sourceID)

		data, err := service.IngestData(sourceID)
		if err != nil {
			log.Printf("Error ingesting data for source ID %s: %v", sourceID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":   "Failed to ingest data",
				"source_id": sourceID,
				"error":     err.Error(),
			})
			return
		}

		log.Printf("Successfully ingested %d records for source ID: %s", len(data), sourceID)
		if len(data) > 0 {
			// Log a sample of the data (e.g., the first record)
			// Be cautious with logging sensitive data in production.
			log.Printf("Sample of ingested data for source ID %s (first record): %+v", sourceID, data[0])
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Data ingested successfully",
			"source_id":      sourceID,
			"records_ingested": len(data),
			// Do not return the actual data in the API response.
			// It will be passed to the processing service in the next step.
		})
	}
}
