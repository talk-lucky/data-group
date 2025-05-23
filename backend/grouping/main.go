package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GroupingService placeholder for business logic
type GroupingService struct {
	// Dependencies like metadata client or DB connection will be added later
}

// NewGroupingService creates a new GroupingService
func NewGroupingService() *GroupingService {
	return &GroupingService{}
}

// CalculateGroup is a placeholder for the actual group calculation logic
func (s *GroupingService) CalculateGroup(groupID string) error {
	log.Printf("GroupingService: CalculateGroup called for groupID: %s. Logic not yet implemented.", groupID)
	// In the future, this will fetch group rules, query processed data, and store results.
	return nil // Or return an error like fmt.Errorf("not implemented")
}

func main() {
	router := gin.Default()
	groupingService := NewGroupingService()

	// Define API v1 group
	v1 := router.Group("/api/v1")
	{
		groupRoutes := v1.Group("/groups")
		{
			groupRoutes.POST("/calculate/:group_id", calculateGroupHandler(groupingService))
		}
	}

	// Start the server
	port := getEnv("PORT", "8083") // Default to 8083 if PORT env var is not set
	log.Printf("Starting Grouping Service on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Grouping Service: %v", err)
	}
}

// calculateGroupHandler creates a gin.HandlerFunc that uses the GroupingService
func calculateGroupHandler(service *GroupingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.Param("group_id")

		log.Printf("Received group calculation request for group_id: %s", groupID)

		err := service.CalculateGroup(groupID)
		if err != nil {
			// If CalculateGroup returns an actual error, handle it
			// For now, it's a placeholder, so we might not expect errors.
			log.Printf("Error calling CalculateGroup for groupID %s: %v", groupID, err)
			// c.JSON(http.StatusInternalServerError, gin.H{
			// 	"message": "Error initiating group calculation",
			// 	"group_id": groupID,
			// 	"error": err.Error(),
			// })
			// return
		}

		// Respond with 202 Accepted or 501 Not Implemented
		c.JSON(http.StatusAccepted, gin.H{
			"message":  "Group calculation request accepted, processing not yet implemented.",
			"group_id": groupID,
		})
		// Or use http.StatusNotImplemented:
		// c.JSON(http.StatusNotImplemented, gin.H{
		//  "message": "Group calculation for group_id " + groupID + " is not yet implemented.",
		//  "group_id": groupID,
		// })
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
