package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// IngestionService defines the operations for data ingestion.
// This is a placeholder for the actual service logic.
type IngestionService struct {
	metadataClient   MetadataServiceClient
	processingClient ProcessingServiceClient
}

// NewIngestionService creates a new instance of IngestionService.
func NewIngestionService(metadataClient MetadataServiceClient, processingClient ProcessingServiceClient) *IngestionService {
	return &IngestionService{
		metadataClient:   metadataClient,
		processingClient: processingClient,
	}
}

// IngestData simulates fetching data for a given source ID and then "ingesting" it.
// In a real scenario, this would involve fetching from an external source,
// transforming, and then possibly sending to a processing service or storage.
func (s *IngestionService) IngestData(sourceID string) ([]map[string]interface{}, error) {
	log.Printf("IngestionService: Attempting to fetch metadata for source ID: %s", sourceID)
	// Step 1: (Simulated) Fetch metadata about the data source
	// metadata, err := s.metadataClient.GetMetadata(sourceID)
	// if err != nil {
	// 	log.Printf("IngestionService: Error fetching metadata for source ID %s: %v", sourceID, err)
	// 	return nil, fmt.Errorf("failed to get metadata for source %s: %w", sourceID, err)
	// }
	// log.Printf("IngestionService: Successfully fetched metadata for source ID %s: %+v", sourceID, metadata)

	// Step 2: (Simulated) Use metadata to fetch actual data from the source
	// This is highly simplified. Actual data fetching would use details from metadata.
	log.Printf("IngestionService: Simulating data fetch for source ID: %s", sourceID)
	simulatedData := []map[string]interface{}{
		{"id": "record1", "value": 100, "source": sourceID},
		{"id": "record2", "value": 200, "source": sourceID},
	}
	log.Printf("IngestionService: Successfully fetched %d records for source ID: %s", len(simulatedData), sourceID)

	// Step 3: (Simulated) Send data to processing service (asynchronously)
	// go func() {
	// 	if err := s.processingClient.ProcessData(sourceID, simulatedData); err != nil {
	// 		log.Printf("IngestionService: Error sending data to processing service for source %s: %v", sourceID, err)
	// 		// Handle error appropriately (e.g., retry, dead-letter queue)
	// 	} else {
	// 		log.Printf("IngestionService: Successfully sent data for source %s to processing service.", sourceID)
	// 	}
	// }()

	return simulatedData, nil
}

// --- Placeholder Clients ---

// MetadataServiceClient defines the interface for a metadata service client.
type MetadataServiceClient interface {
	GetMetadata(sourceID string) (map[string]string, error)
}

// HTTPMetadataClient is an HTTP implementation of MetadataServiceClient.
type HTTPMetadataClient struct {
	baseURL string
}

// NewHTTPMetadataClient creates a new HTTPMetadataClient.
// This is a placeholder. A real client would use net/http to make calls.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	log.Printf("Initializing HTTPMetadataClient with baseURL: %s", baseURL)
	return &HTTPMetadataClient{baseURL: baseURL}
}

// GetMetadata is a placeholder implementation.
func (c *HTTPMetadataClient) GetMetadata(sourceID string) (map[string]string, error) {
	log.Printf("HTTPMetadataClient: Called GetMetadata for sourceID: %s (URL: %s/metadata/%s)", sourceID, c.baseURL, sourceID)
	// Simulate API call: resp, err := http.Get(fmt.Sprintf("%s/api/v1/metadata/sources/%s", c.baseURL, sourceID))
	// For now, return mock data.
	if sourceID == "error_source" {
		return nil, fmt.Errorf("mock error fetching metadata for %s", sourceID)
	}
	return map[string]string{"source_type": "database", "connection_string": "mock_connection"}, nil
}

// ProcessingServiceClient defines the interface for a processing service client.
type ProcessingServiceClient interface {
	ProcessData(sourceID string, data []map[string]interface{}) error
}

// HTTPProcessingServiceClient is an HTTP implementation of ProcessingServiceClient.
type HTTPProcessingServiceClient struct {
	baseURL string
}

// NewHTTPProcessingServiceClient creates a new HTTPProcessingServiceClient.
// This is a placeholder.
func NewHTTPProcessingServiceClient(baseURL string) *HTTPProcessingServiceClient {
	log.Printf("Initializing HTTPProcessingServiceClient with baseURL: %s", baseURL)
	return &HTTPProcessingServiceClient{baseURL: baseURL}
}

// ProcessData is a placeholder implementation.
func (c *HTTPProcessingServiceClient) ProcessData(sourceID string, data []map[string]interface{}) error {
	log.Printf("HTTPProcessingServiceClient: Called ProcessData for sourceID: %s with %d records (URL: %s/api/v1/process/)", sourceID, len(data), c.baseURL)
	// Simulate API call: _, err := http.Post(fmt.Sprintf("%s/api/v1/process/%s", c.baseURL, sourceID), "application/json", bytes.NewBuffer(jsonData))
	if sourceID == "processing_error_source" {
		return fmt.Errorf("mock error processing data for %s", sourceID)
	}
	log.Printf("HTTPProcessingServiceClient: Mock call to process data for %s successful.", sourceID)
	return nil
}

// getEnv reads an environment variable with a fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	// Configuration with environment variables and defaults
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8090")
	processingServiceURL := getEnv("PROCESSING_SERVICE_URL", "http://localhost:8082")
	ingestionServicePort := getEnv("INGESTION_SERVICE_PORT", "8081")

	log.Printf("Configuration:")
	log.Printf("  Metadata Service URL: %s", metadataServiceURL)
	log.Printf("  Processing Service URL: %s", processingServiceURL)
	log.Printf("  Ingestion Service Port: %s", ingestionServicePort)

	router := gin.Default()

	// Initialize Metadata Service Client
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)

	// Initialize Processing Service Client
	processingClient := NewHTTPProcessingServiceClient(processingServiceURL)

	// Initialize IngestionService with both clients
	ingestionSvc := NewIngestionService(metadataClient, processingClient)

	// Define API v1 group
	v1 := router.Group("/api/v1")
	{
		ingestRoutes := v1.Group("/ingest")
		{
			ingestRoutes.POST("/trigger/:source_id", triggerIngestionHandler(ingestionSvc))
		}
	}

	// Start the server
	listenAddr := fmt.Sprintf(":%s", ingestionServicePort)
	log.Printf("Starting Data Ingestion Service on %s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
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
		// In a real application, avoid logging potentially large or sensitive data directly.
		// if len(data) > 0 {
		// 	log.Printf("Sample of ingested data for source ID %s (first record): %+v", sourceID, data[0])
		// }

		c.JSON(http.StatusOK, gin.H{
			"message":        "Data ingested successfully",
			"source_id":      sourceID,
			"records_ingested": len(data),
			// Data is not returned in the response as it's processed internally.
		})
	}
}
