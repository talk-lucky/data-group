package processing

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// MetadataServiceAPIClient defines the interface for an API client to fetch metadata.
type MetadataServiceAPIClient interface {
	GetDataSourceFieldMappings(sourceID string) ([]DataSourceFieldMapping, error)
	GetAttributeDefinition(attributeID string, entityID string) (*AttributeDefinition, error) // entityID might be needed for context
	GetEntityDefinition(entityID string) (*EntityDefinition, error)
	GetDataSourceConfig(sourceID string) (*DataSourceConfig, error) // Added to fetch EntityID from DataSource
}

// HTTPMetadataClient is an implementation of MetadataServiceAPIClient using HTTP.
type HTTPMetadataClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPMetadataClient creates a new client for the metadata service.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	return &HTTPMetadataClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetDataSourceFieldMappings fetches field mappings for a data source.
func (c *HTTPMetadataClient) GetDataSourceFieldMappings(sourceID string) ([]DataSourceFieldMapping, error) {
	url := fmt.Sprintf("%s/api/v1/datasources/%s/mappings", c.BaseURL, sourceID)
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get field mappings from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned non-OK status %d for field mappings at %s", resp.StatusCode, url)
	}

	var mappings []DataSourceFieldMapping
	if err := json.NewDecoder(resp.Body).Decode(&mappings); err != nil {
		return nil, fmt.Errorf("failed to decode field mappings response: %w", err)
	}
	return mappings, nil
}

// GetAttributeDefinition fetches a single attribute definition.
// Note: The metadata API might need adjustment if it doesn't support fetching a single attribute by ID directly.
// It might be /api/v1/entities/{entity_id}/attributes/{attribute_id}
func (c *HTTPMetadataClient) GetAttributeDefinition(attributeID string, entityID string) (*AttributeDefinition, error) {
	url := fmt.Sprintf("%s/api/v1/entities/%s/attributes/%s", c.BaseURL, entityID, attributeID)
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute definition from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned non-OK status %d for attribute definition at %s", resp.StatusCode, url)
	}

	var attrDef AttributeDefinition
	if err := json.NewDecoder(resp.Body).Decode(&attrDef); err != nil {
		return nil, fmt.Errorf("failed to decode attribute definition response: %w", err)
	}
	return &attrDef, nil
}

// GetEntityDefinition fetches an entity definition by its ID.
func (c *HTTPMetadataClient) GetEntityDefinition(entityID string) (*EntityDefinition, error) {
	url := fmt.Sprintf("%s/api/v1/entities/%s", c.BaseURL, entityID)
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity definition from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned non-OK status %d for entity definition at %s", resp.StatusCode, url)
	}

	var entityDef EntityDefinition
	if err := json.NewDecoder(resp.Body).Decode(&entityDef); err != nil {
		return nil, fmt.Errorf("failed to decode entity definition response: %w", err)
	}
	return &entityDef, nil
}

// GetDataSourceConfig fetches a DataSourceConfig from the metadata service. (Copied from ingestion service for now)
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
		return nil, fmt.Errorf("metadata service returned non-OK status: %d for source ID %s", resp.StatusCode, sourceID)
	}

	var config DataSourceConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response from metadata service: %w", err)
	}
	return &config, nil
}

// ProcessingService handles data processing and storage.
type ProcessingService struct {
	metadataClient MetadataServiceAPIClient
	db             *sql.DB
}

// NewProcessingService creates a new ProcessingService.
func NewProcessingService(client MetadataServiceAPIClient, db *sql.DB) *ProcessingService {
	return &ProcessingService{
		metadataClient: client,
		db:             db,
	}
}

// ProcessAndStoreData processes raw data based on mappings and stores it.
func (s *ProcessingService) ProcessAndStoreData(sourceID string, entityTypeName string, rawData []map[string]interface{}) (int, error) {
	log.Printf("Processing data for sourceID: %s, entityTypeName: %s. Records received: %d", sourceID, entityTypeName, len(rawData))

	mappings, err := s.metadataClient.GetDataSourceFieldMappings(sourceID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch field mappings for source %s: %w", sourceID, err)
	}
	if len(mappings) == 0 {
		log.Printf("No field mappings found for source %s. Skipping processing.", sourceID)
		return 0, nil // Or an error, depending on desired behavior
	}
	log.Printf("Fetched %d field mappings for source %s", len(mappings), sourceID)

	// Cache attribute definitions to avoid refetching for each record
	attributeDefs := make(map[string]*AttributeDefinition)
	for _, mapping := range mappings {
		if _, exists := attributeDefs[mapping.AttributeID]; !exists {
			attrDef, err := s.metadataClient.GetAttributeDefinition(mapping.AttributeID, mapping.EntityID)
			if err != nil {
				return 0, fmt.Errorf("failed to fetch attribute definition for ID %s (EntityID %s): %w", mapping.AttributeID, mapping.EntityID, err)
			}
			attributeDefs[mapping.AttributeID] = attrDef
			log.Printf("Fetched attribute definition for ID %s: Name '%s'", mapping.AttributeID, attrDef.Name)
		}
	}

	processedCount := 0
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	stmt, err := tx.Prepare("INSERT INTO processed_entities (id, entity_type_name, data, raw_record_identifier, processed_at) VALUES ($1, $2, $3, $4, $5)")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for i, rawRecord := range rawData {
		processedRecordData := make(map[string]interface{})
		var rawRecordIdentifier string // Optional: try to find a unique ID

		for _, mapping := range mappings {
			rawValue, ok := rawRecord[mapping.SourceFieldName]
			if !ok {
				// log.Printf("Source field '%s' not found in raw record #%d for source %s. Skipping field.", mapping.SourceFieldName, i+1, sourceID)
				continue // Or assign a default, or error out, based on requirements
			}

			// Attempt to create a simple raw record identifier for tracing
			// This is very basic; a hash of the raw record might be better if a unique ID field isn't available
			if idVal, idOk := rawRecord["id"].(string); idOk && rawRecordIdentifier == "" {
				rawRecordIdentifier = idVal
			} else if idNum, idNumOk := rawRecord["id"].(float64); idNumOk && rawRecordIdentifier == "" { // JSON numbers are float64
				rawRecordIdentifier = fmt.Sprintf("%.0f", idNum)
			}


			targetAttrDef, ok := attributeDefs[mapping.AttributeID]
			if !ok {
				log.Printf("Attribute definition for ID %s not found in cache. This should not happen.", mapping.AttributeID)
				continue // Should have been fetched already
			}
			targetAttrName := targetAttrDef.Name

			// Apply transformation rules (simple example)
			transformedValue := rawValue
			if mapping.TransformationRule != "" {
				log.Printf("Applying transformation rule '%s' to field '%s' for attribute '%s'", mapping.TransformationRule, mapping.SourceFieldName, targetAttrName)
				switch strings.ToLower(mapping.TransformationRule) {
				case "lowercase":
					if strVal, isStr := rawValue.(string); isStr {
						transformedValue = strings.ToLower(strVal)
					} else {
						log.Printf("Warning: 'lowercase' rule applied to non-string field '%s' (type %T)", mapping.SourceFieldName, rawValue)
					}
				case "trim":
					if strVal, isStr := rawValue.(string); isStr {
						transformedValue = strings.TrimSpace(strVal)
					} else {
						log.Printf("Warning: 'trim' rule applied to non-string field '%s' (type %T)", mapping.SourceFieldName, rawValue)
					}
				default:
					log.Printf("Warning: Transformation rule '%s' for field '%s' is not implemented.", mapping.TransformationRule, mapping.SourceFieldName)
					// Keep original value if rule is not implemented
				}
			}
			processedRecordData[targetAttrName] = transformedValue
		}
		
		if len(processedRecordData) == 0 {
			log.Printf("Record #%d for source %s resulted in empty processed data. Skipping.", i+1, sourceID)
			continue
		}

		// If rawRecordIdentifier is still empty after checking "id", try to use the first field's value as a fallback.
		// This is a very basic fallback and might not be unique.
		if rawRecordIdentifier == "" && len(rawRecord) > 0 {
			for _, v := range rawRecord {
				rawRecordIdentifier = fmt.Sprintf("%v", v) // Convert first value to string
				break
			}
		}


		jsonData, err := json.Marshal(processedRecordData)
		if err != nil {
			log.Printf("Failed to marshal processed record #%d for source %s: %v. Skipping.", i+1, sourceID, err)
			continue // Or handle error more gracefully
		}

		recordID := uuid.New()
		_, err = stmt.Exec(recordID, entityTypeName, jsonData, rawRecordIdentifier, time.Now().UTC())
		if err != nil {
			// Attempt to rollback and return error immediately if a single insert fails.
			// Depending on requirements, one might choose to continue and report errors later.
			log.Printf("Failed to insert processed record #%d (ID: %s) for source %s: %v", i+1, recordID, sourceID, err)
			// tx.Rollback() // Done by defer
			return processedCount, fmt.Errorf("failed to insert record %s: %w", recordID, err)
		}
		processedCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit database transaction: %w", err)
	}

	log.Printf("Successfully processed and stored %d records for sourceID: %s, entityTypeName: %s", processedCount, sourceID, entityTypeName)
	return processedCount, nil
}

// Helper function to make HTTP POST requests with JSON body
func (c *HTTPMetadataClient) Post(url string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
}
