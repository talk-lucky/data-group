package main

import (
	"log"

	"example.com/project/metadata" // Adjusted import path
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize the in-memory store
	store := metadata.NewStore()

	// Initialize the API with the store
	metaAPI := metadata.NewAPI(store)

	// Initialize Gin router
	router := gin.Default()

	// Register API routes
	metaAPI.RegisterRoutes(router)

	// Start the server
	log.Println("Starting server on :8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
