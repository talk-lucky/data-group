package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/project/metadata" // Assuming this path is correct as per project structure
	"github.com/gin-gonic/gin"
)

// getEnv reads an environment variable with a fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using default: %s", key, fallback)
	return fallback
}

// proxyTo creates a reverse proxy handler for the given target URL.
// (This function remains unchanged from the previous version)
func proxyTo(targetServiceBaseUrl string, prefixToStrip string, pathPrefixToReplace map[string]string) gin.HandlerFunc {
	target, err := url.Parse(targetServiceBaseUrl)
	if err != nil {
		log.Fatalf("Error parsing target URL %s: %v", targetServiceBaseUrl, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		
		originalPath := req.URL.Path
		
		if len(pathPrefixToReplace) > 0 {
			for oldPrefix, newPrefix := range pathPrefixToReplace {
				if strings.HasPrefix(originalPath, oldPrefix) {
					originalPath = strings.Replace(originalPath, oldPrefix, newPrefix, 1)
					break 
				}
			}
		}

		if prefixToStrip != "" {
			req.URL.Path = strings.TrimPrefix(originalPath, prefixToStrip)
		} else {
			req.URL.Path = originalPath
		}

		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		
		req.Host = target.Host 
		log.Printf("Proxying request: %s %s%s -> %s%s", req.Method, req.Host, req.URL.Path, target.Scheme, target.Host+req.URL.Path)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("HTTP proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// runMetadataService starts the metadata service with a PostgreSQL backend.
func runMetadataService(ctx context.Context, serviceAddr string, dbDataSourceName string) {
	log.Printf("Attempting to connect to metadata database with DSN: %s (details omitted for security if password was included)", dbDataSourceName) // Simplified DSN logging
	
	// Initialize the PostgreSQL store for metadata
	// The metadata.Store interface is implemented by metadata.PostgresStore
	store, err := metadata.NewPostgresStore(dbDataSourceName)
	if err != nil {
		log.Fatalf("Failed to initialize metadata PostgreSQL store: %v", err)
	}
	defer func() {
		log.Println("Closing metadata store connection...")
		if err := store.Close(); err != nil {
			log.Printf("Error closing metadata store: %v", err)
		}
	}()

	// Initialize the API with the store for metadata
	// metadata.NewAPI expects an argument that satisfies the metadata.Store interface.
	// PostgresStore should satisfy this interface.
	metadataAPI := metadata.NewAPI(store)

	// Run Metadata service on a separate port in a goroutine
	metaRouter := gin.New()
	metaRouter.Use(gin.Logger())
	metaRouter.Use(gin.Recovery())
	metadataAPI.RegisterRoutes(metaRouter) // This registers routes like /items, /items/:id

	metadataServer := &http.Server{
		Addr:    serviceAddr,
		Handler: metaRouter,
	}

	log.Printf("Starting internal Metadata service on %s", serviceAddr)
	go func() {
		if err := metadataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metadata service ListenAndServe error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down internal Metadata service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := metadataServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Metadata service shutdown error: %v", err)
	}
	log.Println("Internal Metadata service stopped.")
}

func main() {
	log.Println("Starting API Gateway...")

	// --- Metadata Service Configuration ---
	dbHost := getEnv("METADATA_DB_HOST", "localhost")
	dbPort := getEnv("METADATA_DB_PORT", "5432")
	dbUser := getEnv("METADATA_DB_USER", "admin")
	dbPassword := getEnv("METADATA_DB_PASSWORD", "password") // Be cautious with logging this
	dbName := getEnv("METADATA_DB_NAME", "metadata_db")
	dbSSLMode := getEnv("METADATA_DB_SSLMODE", "disable")

	// Construct PostgreSQL dataSourceName
	dbDataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	
	// Log the DSN for debugging, mask password in real production logs
	log.Printf("Metadata DB DSN: host=%s port=%s user=%s password=*** dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbName, dbSSLMode)


	// Context for metadata service shutdown
	metadataServiceCtx, metadataServiceCancel := context.WithCancel(context.Background())
	
	// Start Metadata Service in a goroutine
	metadataInternalAddr := "localhost:8090" // Internal port for metadata service
	go runMetadataService(metadataServiceCtx, metadataInternalAddr, dbDataSourceName)


	// --- Initialize Gin router for the Gateway ---
	gatewayRouter := gin.New()
	gatewayRouter.Use(gin.Logger())
	gatewayRouter.Use(gin.Recovery())

	// --- Setup Gateway Routes ---
	// --- Setup Gateway Routes ---
	metadataTarget := "http://" + metadataInternalAddr

	// Standardized Metadata Service Routes:
	// These routes proxy directly to the internal metadata service,
	// and the internal metadata service's router handles the full /api/v1/... path.
	// The `prefixToStrip` is empty, meaning the gateway path is sent as-is to the target.
	gatewayRouter.Any("/api/v1/entities/*any", proxyTo(metadataTarget, "", nil))
	gatewayRouter.Any("/api/v1/datasources/*any", proxyTo(metadataTarget, "", nil)) // Covers nested /mappings
	gatewayRouter.Any("/api/v1/group-definitions/*any", proxyTo(metadataTarget, "", nil)) // New route for metadata GroupDefinitions
	gatewayRouter.Any("/api/v1/workflows/*any", proxyTo(metadataTarget, "", nil))
	gatewayRouter.Any("/api/v1/actiontemplates/*any", proxyTo(metadataTarget, "", nil))
	gatewayRouter.Any("/api/v1/schedules/*any", proxyTo(metadataTarget, "", nil))


	// Ingest Service: /api/v1/ingest/*any -> http://localhost:8081 (target handles full path)
	ingestTargetURL := getEnv("INGEST_SERVICE_URL", "http://localhost:8081") 
	gatewayRouter.Any("/api/v1/ingest/*any", proxyTo(ingestTargetURL, "", nil))

	// Processing Service: /api/v1/processing/*any -> http://localhost:8082/api/v1/process/*any (path prefix replacement)
	processingTargetURL := getEnv("PROCESSING_SERVICE_URL", "http://localhost:8082")
	processingPathReplace := map[string]string{"/api/v1/processing": "/api/v1/process"}
	gatewayRouter.Any("/api/v1/processing/*any", proxyTo(processingTargetURL, "", processingPathReplace))

	// Grouping Service (for group calculations and results - path remains /api/v1/groups): 
	// /api/v1/groups/*any -> http://localhost:8083 (target handles full path)
	groupsCalculationTargetURL := getEnv("GROUPS_SERVICE_URL", "http://localhost:8083")
	gatewayRouter.Any("/api/v1/groups/*any", proxyTo(groupsCalculationTargetURL, "", nil))

	// Orchestration Service: /api/v1/orchestration/*any -> http://localhost:8084 (target handles full path)
	orchestrationTargetURL := getEnv("ORCHESTRATION_SERVICE_URL", "http://localhost:8084")
	gatewayRouter.Any("/api/v1/orchestration/*any", proxyTo(orchestrationTargetURL, "", nil))
	
	gatewayRouter.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "API Gateway is running"})
	})


	// Start the Gateway server
	gatewayServiceAddr := ":" + getEnv("API_GATEWAY_PORT", "8080")
	log.Printf("API Gateway listening on %s", gatewayServiceAddr)
	
	gatewayServer := &http.Server{
		Addr:    gatewayServiceAddr,
		Handler: gatewayRouter,
	}

	// Graceful shutdown for Gateway and dependent services
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		if err := gatewayServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway ListenAndServe error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down API Gateway...")

	// Shutdown Gateway
	gatewayShutdownCtx, gatewayShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout for gateway + metadata
	defer gatewayShutdownCancel()

	if err := gatewayServer.Shutdown(gatewayShutdownCtx); err != nil {
		log.Fatalf("API Gateway server forced to shutdown: %v", err)
	}
	log.Println("API Gateway server exited properly.")

	// Signal Metadata Service to shutdown
	log.Println("Signaling internal Metadata service to shutdown...")
	metadataServiceCancel() 

	// Potentially wait for metadata service to confirm shutdown if runMetadataService had a waitgroup or channel
	// For now, rely on the sequence and store.Close() defer. A more robust system might use a sync.WaitGroup.
	// The current runMetadataService will block on its own shutdown, and its defer store.Close() will run.

	log.Println("All services should be stopped.")
}
