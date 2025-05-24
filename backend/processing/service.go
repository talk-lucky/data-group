package processing

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// MetadataServiceAPIClient defines the interface for an API client to fetch metadata.
type MetadataServiceAPIClient interface {
	GetDataSourceFieldMappings(sourceID string) ([]DataSourceFieldMapping, error)
	GetAttributeDefinition(attributeID string, entityID string) (*AttributeDefinition, error)
	GetDataSourceConfig(sourceID string) (*DataSourceConfig, error)
	// GetEntityDefinition(entityID string) (*EntityDefinition, error) // Not used by current ProcessAndStoreData
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
		return nil, fmt.Errorf("metadata service returned non-OK status %d for DataSourceConfig with source ID %s", resp.StatusCode, sourceID)
	}

	var config DataSourceConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode DataSourceConfig response from metadata service: %w", err)
	}
	return &config, nil
}

// initSchema creates the processed_entities table if it doesn't exist.
func initSchema(db *sql.DB) error {
	schema := `
    CREATE TABLE IF NOT EXISTS processed_entities (
        id UUID PRIMARY KEY,
        entity_definition_id TEXT, 
        entity_type_name TEXT NOT NULL,
        source_id TEXT,
        attributes JSONB,
        raw_record_identifier TEXT,
        processed_at TIMESTAMPTZ DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_processed_entities_entity_def_id ON processed_entities(entity_definition_id);
    CREATE INDEX IF NOT EXISTS idx_processed_entities_entity_type_name ON processed_entities(entity_type_name);
    CREATE INDEX IF NOT EXISTS idx_processed_entities_source_id ON processed_entities(source_id);
    `
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema initialization for processed_entities: %w", err)
	}
	log.Println("Schema for 'processed_entities' table initialized successfully.")
	return nil
}

// ProcessingService handles data processing and storage.
type ProcessingService struct {
	metadataClient MetadataServiceAPIClient
	db             *sql.DB
}

// NewProcessingService creates a new ProcessingService.
func NewProcessingService(client MetadataServiceAPIClient, db *sql.DB) *ProcessingService {
	if db != nil { // Allow nil DB for testing transformation logic separately
		if err := initSchema(db); err != nil {
			log.Panicf("Failed to initialize database schema for ProcessingService: %v", err)
		}
	}
	return &ProcessingService{
		metadataClient: client,
		db:             db,
	}
}

// convertToTargetType attempts to convert an interface{} value to a specified target data type.
func convertToTargetType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	strValue := fmt.Sprintf("%v", value)

	switch strings.ToLower(targetType) {
	case "string":
		return strValue, nil
	case "integer":
		if fVal, ok := value.(float64); ok {
			return int64(fVal), nil
		}
		if jnVal, ok := value.(json.Number); ok {
			iVal, err := jnVal.Int64()
			if err == nil {
				return iVal, nil
			}
		}
		return strconv.ParseInt(strValue, 10, 64)
	case "float":
		if jnVal, ok := value.(json.Number); ok {
			fVal, err := jnVal.Float64()
			if err == nil {
				return fVal, nil
			}
		}
		if iVal, ok := value.(int64); ok {
			return float64(iVal), nil
		}
		if iVal, ok := value.(int); ok {
			return float64(iVal), nil
		}
		return strconv.ParseFloat(strValue, 64)
	case "boolean":
		if bVal, ok := value.(bool); ok {
			return bVal, nil
		}
		if fVal, ok := value.(float64); ok {
			return fVal != 0, nil
		}
		if iVal, ok := value.(int64); ok {
			return iVal != 0, nil
		}
		if jnVal, ok := value.(json.Number); ok {
			iVal, errInt := jnVal.Int64()
			if errInt == nil {
				return iVal != 0, nil
			}
			fVal, errFloat := jnVal.Float64()
			if errFloat == nil {
				return fVal != 0, nil
			}
		}
		lowerStrValue := strings.ToLower(strValue)
		switch lowerStrValue {
		case "true", "t", "yes", "1":
			return true, nil
		case "false", "f", "no", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("cannot convert string '%s' to boolean", strValue)
		}
	case "date", "datetime":
		if tVal, ok := value.(time.Time); ok {
			return tVal, nil
		}
		formats := []string{
			time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05", "2006-01-02", "2006/01/02", "01/02/2006", "1/2/2006",
		}
		for _, format := range formats {
			t, err := time.Parse(format, strValue)
			if err == nil {
				return t, nil
			}
		}
		unixSec, errSec := strconv.ParseInt(strValue, 10, 64)
		if errSec == nil {
			return time.Unix(unixSec, 0).UTC(), nil // Return as UTC
		}
		unixFloat, errFloat := strconv.ParseFloat(strValue, 64)
		if errFloat == nil {
			sec, nsec := int64(unixFloat), int64((unixFloat-float64(int64(unixFloat)))*1e9)
			return time.Unix(sec, nsec).UTC(), nil // Return as UTC
		}
		return nil, fmt.Errorf("cannot parse date/datetime string '%s' with known formats", strValue)
	default:
		// For unsupported types, we previously returned an error.
		// However, the requirement for transformAndConvertRecord is to log and skip.
		// convertToTargetType should signal this by returning the original value and an error.
		return value, fmt.Errorf("unsupported target data type: %s", targetType)
	}
}

// transformAndConvertRecord processes a single raw record based on mappings and attribute definitions.
// It returns the processed data map and a raw record identifier string.
func (s *ProcessingService) transformAndConvertRecord(
	rawRecord map[string]interface{},
	mappings []DataSourceFieldMapping,
	attributeDefs map[string]*AttributeDefinition,
	recordIndex int,
	sourceID string,
) (map[string]interface{}, string) {
	processedRecordData := make(map[string]interface{})
	var rawRecordIdentifierValue string

	if idVal, idOk := rawRecord["id"]; idOk {
		rawRecordIdentifierValue = fmt.Sprintf("%v", idVal)
	} else if idVal, idOk := rawRecord["source_record_id"]; idOk {
		rawRecordIdentifierValue = fmt.Sprintf("%v", idVal)
	}

	for _, mapping := range mappings {
		rawValue, ok := rawRecord[mapping.SourceFieldName]
		if !ok {
			continue
		}

		targetAttrDef, ok := attributeDefs[mapping.AttributeID]
		if !ok {
			log.Printf("Attribute definition for ID %s not found in cache for record #%d. Skipping field.", mapping.AttributeID, recordIndex)
			continue
		}
		targetAttrName := targetAttrDef.Name
		targetDataType := targetAttrDef.DataType
		transformedValue := rawValue

		if mapping.TransformationRule != "" {
			switch strings.ToLower(mapping.TransformationRule) {
			case "lowercase":
				if strVal, isStr := transformedValue.(string); isStr {
					transformedValue = strings.ToLower(strVal)
				} else {
					log.Printf("Warning: 'lowercase' rule for field '%s' (type %T) for record #%d, sourceID '%s'. Value not a string.", mapping.SourceFieldName, transformedValue, recordIndex, sourceID)
				}
			case "trim":
				if strVal, isStr := transformedValue.(string); isStr {
					transformedValue = strings.TrimSpace(strVal)
				} else {
					log.Printf("Warning: 'trim' rule for field '%s' (type %T) for record #%d, sourceID '%s'. Value not a string.", mapping.SourceFieldName, transformedValue, recordIndex, sourceID)
				}
			default:
				log.Printf("Warning: Transformation rule '%s' for field '%s' (record #%d, sourceID '%s') is not implemented.", mapping.TransformationRule, mapping.SourceFieldName, recordIndex, sourceID)
			}
		}

		convertedValue, err := convertToTargetType(transformedValue, targetDataType)
		if err != nil {
			log.Printf("Could not convert value '%v' (original: '%v') for source field '%s' to target type '%s' for attribute '%s' (record #%d, sourceID '%s'). Skipping field. Error: %v",
				transformedValue, rawValue, mapping.SourceFieldName, targetDataType, targetAttrName, recordIndex, sourceID, err)
			continue
		}
		processedRecordData[targetAttrName] = convertedValue
	}
	return processedRecordData, rawRecordIdentifierValue
}

// ProcessAndStoreData processes raw data based on mappings and stores it.
func (s *ProcessingService) ProcessAndStoreData(sourceID string, entityTypeName string, rawData []map[string]interface{}) (int, error) {
	log.Printf("Processing data for sourceID: %s, entityTypeName: %s. Records received: %d", sourceID, entityTypeName, len(rawData))

	dsConfig, err := s.metadataClient.GetDataSourceConfig(sourceID)
	var entityDefinitionID string
	if err != nil {
		log.Printf("Warning: Failed to fetch DataSourceConfig for source %s: %v. entity_definition_id will be empty.", sourceID, err)
		// dsConfig remains nil or you can create an empty one: dsConfig = &DataSourceConfig{}
	} else if dsConfig != nil {
		entityDefinitionID = dsConfig.EntityID
	}


	mappings, err := s.metadataClient.GetDataSourceFieldMappings(sourceID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch field mappings for source %s: %w", sourceID, err)
	}
	if len(mappings) == 0 {
		log.Printf("No field mappings found for source %s. Skipping processing.", sourceID)
		return 0, nil
	}
	log.Printf("Fetched %d field mappings for source %s", len(mappings), sourceID)

	attributeDefs := make(map[string]*AttributeDefinition)
	for _, mapping := range mappings {
		if _, exists := attributeDefs[mapping.AttributeID]; !exists {
			attrDef, err := s.metadataClient.GetAttributeDefinition(mapping.AttributeID, mapping.EntityID)
			if err != nil {
				log.Printf("Warning: Failed to fetch attribute definition for ID %s (EntityID %s): %v. This attribute will be skipped for all records.", mapping.AttributeID, mapping.EntityID, err)
				continue
			}
			attributeDefs[mapping.AttributeID] = attrDef
			log.Printf("Fetched attribute definition for ID %s: Name '%s', Type '%s'", mapping.AttributeID, attrDef.Name, attrDef.DataType)
		}
	}
	if len(attributeDefs) == 0 && len(mappings) > 0 {
		return 0, fmt.Errorf("no valid attribute definitions could be fetched for the provided mappings for source %s", sourceID)
	}

	if s.db == nil {
		log.Println("Warning: ProcessingService.db is nil. Skipping database operations. This should only occur in specific test scenarios.")
		processedCountForLogicTest := 0
		for i, rawRecord := range rawData {
			processedRecord, _ := s.transformAndConvertRecord(rawRecord, mappings, attributeDefs, i+1, sourceID)
			if len(processedRecord) > 0 {
				processedCountForLogicTest++
			}
		}
		return processedCountForLogicTest, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO processed_entities (id, entity_definition_id, entity_type_name, source_id, attributes, raw_record_identifier, processed_at) VALUES ($1, $2, $3, $4, $5, $6, $7)")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement for processed_entities: %w", err)
	}
	defer stmt.Close()

	processedCount := 0
	for i, rawRecord := range rawData {
		processedRecordData, rawRecordIdentifierStr := s.transformAndConvertRecord(rawRecord, mappings, attributeDefs, i+1, sourceID)
		
		if len(processedRecordData) == 0 {
			log.Printf("Record #%d for source %s resulted in empty processed data after mapping and conversion. Skipping.", i+1, sourceID)
			continue
		}

		jsonData, err := json.Marshal(processedRecordData)
		if err != nil {
			log.Printf("Failed to marshal processed record #%d for source %s: %v. Skipping.", i+1, sourceID, err)
			continue
		}

		recordID := uuid.New()
		var dbEntityDefinitionID sql.NullString
		if entityDefinitionID != "" {
			dbEntityDefinitionID.String = entityDefinitionID
			dbEntityDefinitionID.Valid = true
		}
		var dbRawRecordIdentifier sql.NullString
		if rawRecordIdentifierStr != "" {
			dbRawRecordIdentifier.String = rawRecordIdentifierStr
			dbRawRecordIdentifier.Valid = true
		}

		_, err = stmt.Exec(recordID, dbEntityDefinitionID, entityTypeName, sourceID, jsonData, dbRawRecordIdentifier, time.Now().UTC())
		if err != nil {
			log.Printf("Failed to insert processed record #%d (ID: %s) for source %s: %v", i+1, recordID, sourceID, err)
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

// Simplified struct definitions for metadata types, assumed to be compatible with actual metadata service responses.
type DataSourceFieldMapping struct {
	ID                 string `json:"id"`
	SourceID           string `json:"source_id"`
	SourceFieldName    string `json:"source_field_name"`
	EntityID           string `json:"entity_id"` // Refers to EntityDefinition.ID
	AttributeID        string `json:"attribute_id"`
	TransformationRule string `json:"transformation_rule,omitempty"`
}

type AttributeDefinition struct {
	ID       string `json:"id"`
	EntityID string `json:"entity_id"`
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

type DataSourceConfig struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	ConnectionDetails string `json:"connection_details"`
	EntityID          string `json:"entity_id,omitempty"` // This is the EntityDefinition.ID
}

// EntityDefinition is not directly used in ProcessAndStoreData if EntityTypeName is passed in,
// but GetDataSourceConfig might provide EntityID that refers to an EntityDefinition.
// type EntityDefinition struct {
// 	ID   string `json:"id"`
// 	Name string `json:"name"`
// }
