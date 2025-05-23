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

	// Start the server on port 8081
	port := ":8081"
	log.Printf("Starting Data Ingestion Service on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start Data Ingestion Service: %v", err)
	}
}

// triggerIngestionHandler is a placeholder for the actual ingestion logic.
func triggerIngestionHandler(c *gin.Context) {
	sourceID := c.Param("source_id")

	log.Printf("Received trigger for data source ID: %s", sourceID)

	// Respond with 501 Not Implemented, as the actual ingestion logic is not yet in place.
	// Or use 202 Accepted if you prefer to indicate the request is accepted but not processed yet.
	c.JSON(http.StatusNotImplemented, gin.H{
		"message":   "Ingestion triggered for source_id",
		"source_id": sourceID,
		"status":    "not_implemented_yet",
	})
}
