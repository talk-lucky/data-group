package ingestion

import (
	"bytes" // Used by HTTPProcessingServiceClient
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
	// Consider using pgx for more advanced features if needed: _ "github.com/jackc/pgx/v4/stdlib"
)

// MetadataServiceAPIClient defines the interface for an API client to fetch metadata.
// This allows for easier testing and decoupling.
type MetadataServiceAPIClient interface {
	GetDataSourceConfig(sourceID string) (*DataSourceConfig, error)
}

// HTTPMetadataClient is an implementation of MetadataServiceAPIClient using HTTP.
type HTTPMetadataClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// DataSourceConfig mirrors the structure in the metadata service for unmarshalling.
// Only include fields relevant to ingestion.
type DataSourceConfig struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	ConnectionDetails string `json:"connection_details"` // JSON string
	// EntityID might be used later for schema mapping or validation
	EntityID string `json:"entity_id,omitempty"`
}

// ConnectionParams defines the structure for PostgreSQL connection details.
type ConnectionParams struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password"`
	DBName        string `json:"dbname"`
	TableOrQuery  string `json:"table_or_query"`
	SSLMode       string `json:"sslmode,omitempty"` // e.g., "disable", "require", "verify-full"
}

// CSVConnectionParams defines the structure for CSV connection details.
type CSVConnectionParams struct {
	Filepath string `json:"filepath"`
}

// NewHTTPMetadataClient creates a new client for the metadata service.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	return &HTTPMetadataClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{},
	}
}

// GetDataSourceConfig fetches a DataSourceConfig from the metadata service.
func (c *HTTPMetadataClient) GetDataSourceConfig(sourceID string) (*DataSourceConfig, error) {
	url := fmt.Sprintf("%s/api/v1/datasources/%s", c.BaseURL, sourceID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to metadata service: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call metadata service at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error message from metadata service
		return nil, fmt.Errorf("metadata service returned non-OK status: %d for source ID %s", resp.StatusCode, sourceID)
	}

	var config DataSourceConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response from metadata service: %w", err)
	}
	return &config, nil
}

// ProcessingServiceAPIClient defines the interface for calling the processing service.
type ProcessingServiceAPIClient interface {
	CallProcessData(payload ProcessDataRequest) error
}

// HTTPProcessingServiceClient is an implementation of ProcessingServiceAPIClient.
type HTTPProcessingServiceClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPProcessingServiceClient creates a new client for the processing service.
func NewHTTPProcessingServiceClient(baseURL string) *HTTPProcessingServiceClient {
	return &HTTPProcessingServiceClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{}, // Consider adding timeout
	}
}

// ProcessDataRequest is the payload for the processing service.
type ProcessDataRequest struct {
	SourceID       string                   `json:"source_id"`
	EntityTypeName string                   `json:"entity_type_name"`
	RawData        []map[string]interface{} `json:"raw_data"`
}

// CallProcessData makes a POST request to the processing service.
func (c *HTTPProcessingServiceClient) CallProcessData(payload ProcessDataRequest) error {
	url := fmt.Sprintf("%s/api/v1/process", c.BaseURL)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal process data payload: %w", err)
	}

	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to call processing service at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error message from processing service
		return fmt.Errorf("processing service returned non-OK status: %d for source ID %s", resp.StatusCode, payload.SourceID)
	}
	log.Printf("Successfully called processing service for SourceID: %s. Status: %d", payload.SourceID, resp.StatusCode)
	return nil
}

// IngestionService handles the data ingestion logic.
type IngestionService struct {
	metadataClient   MetadataServiceAPIClient
	processingClient ProcessingServiceAPIClient
}

// NewIngestionService creates a new IngestionService.
func NewIngestionService(metaClient MetadataServiceAPIClient, procClient ProcessingServiceAPIClient) *IngestionService {
	return &IngestionService{
		metadataClient:   metaClient,
		processingClient: procClient,
	}
}

// IngestData fetches data from a configured data source and sends it for processing.
func (s *IngestionService) IngestData(sourceID string) ([]map[string]interface{}, error) {
	log.Printf("Starting ingestion for source ID: %s", sourceID)

	// 1. Fetch DataSourceConfig from metadata service
	dsConfig, err := s.metadataClient.GetDataSourceConfig(sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source config for ID %s: %w", sourceID, err)
	}

	var results []map[string]interface{}

	if strings.ToLower(dsConfig.Type) == "csv" {
		log.Printf("Starting CSV ingestion for source ID: %s. ConnectionDetails: %s", sourceID, dsConfig.ConnectionDetails)
		var params CSVConnectionParams
		if err := json.Unmarshal([]byte(dsConfig.ConnectionDetails), &params); err != nil {
			return nil, fmt.Errorf("failed to parse CSV connection details for source ID %s: %w", sourceID, err)
		}

		if params.Filepath == "" {
			return nil, fmt.Errorf("filepath is required for CSV data source type, source ID %s", sourceID)
		}

		file, err := os.Open(params.Filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to open CSV file %s for source ID %s: %w", params.Filepath, sourceID, err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		headers, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				log.Printf("CSV file %s for source ID %s is empty or only contains headers.", params.Filepath, sourceID)
				return results, nil // Empty results if only headers or empty
			}
			return nil, fmt.Errorf("failed to read header row from CSV file %s for source ID %s: %w", params.Filepath, sourceID, err)
		}

		results = make([]map[string]interface{}, 0)
		for {
			row, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break // End of file
				}
				return nil, fmt.Errorf("failed to read row from CSV file %s for source ID %s: %w", params.Filepath, sourceID, err)
			}

			rowData := make(map[string]interface{})
			for i, header := range headers {
				if i < len(row) {
					rowData[header] = row[i]
				} else {
					rowData[header] = "" // Handle short rows if necessary
				}
			}
			results = append(results, rowData)
		}
		log.Printf("Successfully ingested %d records for source ID: %s from CSV file %s.", len(results), sourceID, params.Filepath)

	} else if strings.ToLower(dsConfig.Type) == "postgresql" {
		// 2. Parse ConnectionDetails for PostgreSQL
		var params ConnectionParams
		if err := json.Unmarshal([]byte(dsConfig.ConnectionDetails), &params); err != nil {
			return nil, fmt.Errorf("failed to parse PostgreSQL connection details for source ID %s: %w", sourceID, err)
		}

		if params.Host == "" || params.Port == 0 || params.User == "" || params.DBName == "" || params.TableOrQuery == "" {
			return nil, fmt.Errorf("missing required PostgreSQL connection parameters (host, port, user, dbname, table_or_query) for source ID %s", sourceID)
		}
		if params.SSLMode == "" {
			params.SSLMode = "disable" // Default SSL mode
		}

		// 3. Construct PostgreSQL connection string
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			params.Host, params.Port, params.User, params.Password, params.DBName, params.SSLMode)

		// 4. Connect to PostgreSQL database
		log.Printf("Connecting to PostgreSQL database: %s:%d/%s with user %s", params.Host, params.Port, params.DBName, params.User)
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL for source ID %s: %w", sourceID, err)
		}
		defer db.Close()

		err = db.Ping()
		if err != nil {
			return nil, fmt.Errorf("failed to ping PostgreSQL for source ID %s: %w", sourceID, err)
		}
		log.Printf("Successfully connected to PostgreSQL for source ID: %s", sourceID)

		// 5. Determine query (simple table name or full query)
		query := params.TableOrQuery
		if !strings.Contains(query, " ") && !strings.HasPrefix(strings.ToUpper(query), "SELECT") {
			query = fmt.Sprintf("SELECT * FROM %q", params.TableOrQuery)
		}
		log.Printf("Executing query for source ID %s: %s", sourceID, query)

		// 6. Execute the query
		rows, err := db.Query(query)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query for source ID %s: %w", sourceID, err)
		}
		defer rows.Close()

		// 7. Iterate over rows and scan into map[string]interface{}
		columns, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for query result for source ID %s: %w", sourceID, err)
		}

		results = make([]map[string]interface{}, 0) // Initialize here for clarity
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, fmt.Errorf("failed to scan row for source ID %s: %w", sourceID, err)
			}

			rowData := make(map[string]interface{})
			for i, colName := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					rowData[colName] = string(b)
				} else {
					rowData[colName] = val
				}
			}
			results = append(results, rowData)
		}

		if err = rows.Err(); err != nil { // Check for errors encountered during iteration
			return nil, fmt.Errorf("error iterating rows for source ID %s: %w", sourceID, err)
		}
		log.Printf("Successfully ingested %d records for source ID: %s from database.", len(results), sourceID)

	} else {
		// Corrected bug: using dsConfig.Type instead of config.Type
		return nil, fmt.Errorf("unsupported data source type: %s. Only PostgreSQL and CSV are currently supported", dsConfig.Type)
	}

	// After successful ingestion, call the processing service
	if len(results) > 0 {
		entityTypeName := dsConfig.EntityID // Using EntityID as EntityTypeName as per previous logic.
		// If EntityID is an actual ID, we would need another call to metadata service to get EntityDefinition.Name
		// This logic is consistent with how it was handled before for PostgreSQL.

		if dsConfig.EntityID == "" {
			log.Printf("Warning: DataSourceConfig.EntityID is empty for source %s. Processing might not determine entity type correctly.", sourceID)
		}

		processPayload := ProcessDataRequest{
			SourceID:       sourceID,
			EntityTypeName: entityTypeName,
			RawData:        results,
		}

		log.Printf("Sending %d ingested records for SourceID %s (EntityID from DSConfig: '%s') to processing service.", len(results), sourceID, dsConfig.EntityID)
		err = s.processingClient.CallProcessData(processPayload)
		if err != nil {
			log.Printf("Error calling processing service for source ID %s: %v", sourceID, err)
			// Optionally, return this error depending on desired criticality
			// return nil, fmt.Errorf("failed to send data to processing service: %w", err)
		} else {
			log.Printf("Successfully sent data for source ID %s to processing service.", sourceID)
		}
	} else {
		log.Printf("No records ingested for source ID %s. Skipping call to processing service.", sourceID)
	}

	return results, nil
}
