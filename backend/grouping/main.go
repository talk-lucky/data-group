package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	// --- Database Connection for Processed Entities ---
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
	log.Println("Successfully connected to the database for processed_entities.")

	// --- Initialize Services ---
	metadataServiceURL := getEnv("METADATA_SERVICE_URL", "http://localhost:8080")
	metadataClient := NewHTTPMetadataClient(metadataServiceURL)

	orchestrationServiceURL := getEnv("ORCHESTRATION_SERVICE_URL", "http://localhost:8084")
	orchestrationClient := NewHTTPOrchestrationClient(orchestrationServiceURL)

	groupingService := NewGroupingService(metadataClient, orchestrationClient, db)

	// --- HTTP Server Setup ---
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		groupRoutes := v1.Group("/groups")
		{
			groupRoutes.POST("/:group_id/calculate", calculateGroupHandler(groupingService)) // Changed route slightly for consistency
			groupRoutes.GET("/:group_id/results", getGroupResultsHandler(groupingService))
		}
	}

	// Start the server
	serverPort := getEnv("PORT", "8083")
	log.Printf("Starting Grouping Service on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
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
		if len(entityInstanceIDs) > 0 {
			// Log first few IDs for brevity
			limit := 5
			if len(entityInstanceIDs) < limit {
				limit = len(entityInstanceIDs)
			}
			log.Printf("First %d matching instance IDs for group %s: %v", limit, groupID, entityInstanceIDs[:limit])
		}

		// The CalculateGroup method now stores results and returns the instance IDs.
		// We can use time.Now() here for the calculated_at in the response,
		// or fetch it from GetGroupResults if we want the exact stored timestamp.
		// For simplicity, using time.Now() for the response's timestamp for the calculation event.
		c.JSON(http.StatusOK, gin.H{
			"message":       "Group calculation successful and results stored",
			"group_id":      groupID,
			"member_count":  len(entityInstanceIDs),
			"calculated_at": time.Now().UTC().Format(time.RFC3339), // Timestamp of this calculation event
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

		if len(instanceIDs) == 0 && calculatedAt.IsZero() {
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

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using fallback: %s", key, fallback)
	return fallback
}
