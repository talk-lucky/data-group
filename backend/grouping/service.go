package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// --- Metadata Service Client ---

// MetadataServiceAPIClient defines the interface for an API client to fetch metadata.
type MetadataServiceAPIClient interface {
	GetGroupDefinition(groupID string) (*GroupDefinition, error)
	GetEntityDefinition(entityID string) (*EntityDefinition, error)
	GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error)
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

func (c *HTTPMetadataClient) fetchMetadata(url string, target interface{}) error {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error from metadata service
		return fmt.Errorf("metadata service returned non-OK status %d for %s", resp.StatusCode, url)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response from %s: %w", url, err)
	}
	return nil
}

// GetGroupDefinition fetches a GroupDefinition from the metadata service.
func (c *HTTPMetadataClient) GetGroupDefinition(groupID string) (*GroupDefinition, error) {
	var groupDef GroupDefinition
	url := fmt.Sprintf("%s/api/v1/groups/%s", c.BaseURL, groupID)
	err := c.fetchMetadata(url, &groupDef)
	if err != nil {
		return nil, err
	}
	return &groupDef, nil
}

// GetEntityDefinition fetches an EntityDefinition from the metadata service.
func (c *HTTPMetadataClient) GetEntityDefinition(entityID string) (*EntityDefinition, error) {
	var entityDef EntityDefinition
	url := fmt.Sprintf("%s/api/v1/entities/%s", c.BaseURL, entityID)
	err := c.fetchMetadata(url, &entityDef)
	if err != nil {
		return nil, err
	}
	return &entityDef, nil
}

// GetAttributeDefinition fetches an AttributeDefinition from the metadata service.
func (c *HTTPMetadataClient) GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error) {
	var attrDef AttributeDefinition
	// Corrected URL assuming attributes are nested under entities
	url := fmt.Sprintf("%s/api/v1/entities/%s/attributes/%s", c.BaseURL, entityID, attributeID)
	err := c.fetchMetadata(url, &attrDef)
	if err != nil {
		return nil, err
	}
	return &attrDef, nil
}

// --- Grouping Service ---

// GroupingService handles the logic for calculating groups.
type GroupingService struct {
	metadataClient MetadataServiceAPIClient
	db             *sql.DB
}

// NewGroupingService creates a new GroupingService.
func NewGroupingService(client MetadataServiceAPIClient, db *sql.DB) *GroupingService {
	return &GroupingService{
		metadataClient: client,
		db:             db,
	}
}

// CalculateGroup fetches group rules, queries processed data, and returns matching entity instance IDs.
func (s *GroupingService) CalculateGroup(groupID string) ([]string, error) {
	log.Printf("Calculating group for groupID: %s", groupID)

	// 1. Fetch GroupDefinition
	groupDef, err := s.metadataClient.GetGroupDefinition(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group definition for ID %s: %w", groupID, err)
	}
	log.Printf("Fetched GroupDefinition: %s (EntityID: %s)", groupDef.Name, groupDef.EntityID)

	// 2. Fetch EntityDefinition
	entityDef, err := s.metadataClient.GetEntityDefinition(groupDef.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity definition for ID %s: %w", groupDef.EntityID, err)
	}
	entityTypeName := entityDef.Name
	log.Printf("Fetched EntityDefinition: %s (Type Name: %s)", entityDef.Name, entityTypeName)

	// 3. Parse RulesJSON
	var ruleSet RuleSet
	if err := json.Unmarshal([]byte(groupDef.RulesJSON), &ruleSet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules JSON for group %s: %w", groupID, err)
	}
	if len(ruleSet.Conditions) == 0 {
		log.Printf("No conditions found in rules for group %s. Returning empty result.", groupID)
		return []string{}, nil
	}
	log.Printf("Parsed %d rule conditions for group %s. Logical Operator: %s", len(ruleSet.Conditions), groupID, ruleSet.LogicalOperator)


	// 4. Fetch AttributeDefinitions for all unique attribute IDs in conditions
	attributeDefinitions := make(map[string]*AttributeDefinition) // Map AttributeID to its definition
	for _, cond := range ruleSet.Conditions {
		if _, exists := attributeDefinitions[cond.AttributeID]; !exists {
			// Note: The frontend's GroupRuleBuilder uses attribute ID directly.
			// The backend metadata API for attributes is /entities/{entity_id}/attributes/{attribute_id}
			attrDef, err := s.metadataClient.GetAttributeDefinition(groupDef.EntityID, cond.AttributeID)
			if err != nil {
				return nil, fmt.Errorf("failed to get attribute definition for ID %s (Entity %s): %w", cond.AttributeID, groupDef.EntityID, err)
			}
			attributeDefinitions[cond.AttributeID] = attrDef
			log.Printf("Fetched AttributeDefinition: %s (ID: %s, Type: %s)", attrDef.Name, attrDef.ID, attrDef.DataType)
		}
	}
	
	// 5. Construct SQL Query
	var queryBuilder strings.Builder
	var queryParams []interface{}
	paramCounter := 1 // For $1, $2, etc.

	queryBuilder.WriteString(fmt.Sprintf("SELECT id FROM processed_entities WHERE entity_type_name = $%d", paramCounter))
	queryParams = append(queryParams, entityTypeName)
	paramCounter++

	// Assuming "AND" logical operator for now as per instructions
	if strings.ToUpper(ruleSet.LogicalOperator) != "AND" && ruleSet.LogicalOperator != "" {
		log.Printf("Warning: Unsupported logical operator '%s' for group %s. Defaulting to AND.", ruleSet.LogicalOperator, groupID)
		// For now, all conditions are ANDed. Future work could handle OR.
	}

	for _, cond := range ruleSet.Conditions {
		attrDef, ok := attributeDefinitions[cond.AttributeID]
		if !ok {
			return nil, fmt.Errorf("internal error: attribute definition for ID %s not found in fetched map", cond.AttributeID)
		}
		attributeName := attrDef.Name // Use the actual name from metadata for JSONB key

		// Start building the condition string
		queryBuilder.WriteString(" AND ")

		// Handle JSONB access and type casting
		fieldAccessor := fmt.Sprintf("data->>'%s'", attributeName)
		castType := ""
		switch strings.ToLower(attrDef.DataType) {
		case "integer", "long": // Assuming 'long' might be a custom type mapping to integer-like
			castType = "::bigint" // Or ::integer, depending on expected range
		case "float", "double", "decimal": // Assuming these map to numeric types
			castType = "::numeric" // Or ::float, ::double precision
		case "boolean":
			castType = "::boolean"
		case "datetime", "date", "timestamp":
			castType = "::timestamp" // Or ::date, depending on required precision and comparison
		// String types (string, text, char, varchar, etc.) usually don't need explicit cast for text operators
		}

		// Build condition based on operator
		switch strings.ToLower(cond.Operator) {
		case "equals":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) = $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "not_equals":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) != $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "greater_than":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) > $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "less_than":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) < $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "greater_than_or_equal_to": // Corrected from greater_than_or_equals
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) >= $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "less_than_or_equal_to": // Corrected from less_than_or_equals
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) <= $%d", fieldAccessor, castType, paramCounter))
			queryParams = append(queryParams, cond.Value)
			paramCounter++
		case "contains":
			if castType != "" && castType != "::boolean" { // Contains is typically for text, don't cast non-text
				queryBuilder.WriteString(fmt.Sprintf("%s LIKE $%d", fieldAccessor, paramCounter))
			} else {
				queryBuilder.WriteString(fmt.Sprintf("%s LIKE $%d", fieldAccessor, paramCounter))
			}
			queryParams = append(queryParams, fmt.Sprintf("%%%v%%", cond.Value)) // Add wildcards
			paramCounter++
		case "does_not_contain":
			if castType != "" && castType != "::boolean" {
				queryBuilder.WriteString(fmt.Sprintf("%s NOT LIKE $%d", fieldAccessor, paramCounter))
			} else {
				queryBuilder.WriteString(fmt.Sprintf("%s NOT LIKE $%d", fieldAccessor, paramCounter))
			}
			queryParams = append(queryParams, fmt.Sprintf("%%%v%%", cond.Value))
			paramCounter++
		case "is_true":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) IS TRUE", fieldAccessor, castType))
			// No parameter needed
		case "is_false":
			queryBuilder.WriteString(fmt.Sprintf("(%s%s) IS FALSE", fieldAccessor, castType))
			// No parameter needed
		case "is_null":
			// For JSONB, checking for null is (data->'key') IS NULL, not data->>'key' IS NULL
			// because data->>'key' converts JSON null to SQL NULL, which behaves as expected with IS NULL.
			// However, if a key is entirely missing, data->>'key' also results in SQL NULL.
			// This behavior is usually fine for "is_null".
			queryBuilder.WriteString(fmt.Sprintf("%s IS NULL", fieldAccessor))
			// No parameter needed
		case "is_not_null":
			queryBuilder.WriteString(fmt.Sprintf("%s IS NOT NULL", fieldAccessor))
			// No parameter needed
		default:
			return nil, fmt.Errorf("unsupported operator '%s' for attribute %s", cond.Operator, attributeName)
		}
	}

	finalQuery := queryBuilder.String()
	log.Printf("Constructed SQL query for group %s: %s", groupID, finalQuery)
	log.Printf("Query parameters: %v", queryParams)

	// 6. Execute Query
	rows, err := s.db.Query(finalQuery, queryParams...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query for group %s: %w. Query: %s, Params: %v", groupID, err, finalQuery, queryParams)
	}
	defer rows.Close()

	var entityInstanceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan row for group %s: %w", groupID, err)
		}
		entityInstanceIDs = append(entityInstanceIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows for group %s: %w", groupID, err)
	}

	log.Printf("Found %d entity instances for group %s", len(entityInstanceIDs), groupID)

	// Store the results
	err = s.StoreGroupResults(groupID, entityInstanceIDs)
	if err != nil {
		// Log the error, but still return the IDs found by CalculateGroup.
		// Depending on requirements, this could be a critical error.
		log.Printf("Error storing group calculation results for group %s: %v", groupID, err)
		return entityInstanceIDs, fmt.Errorf("failed to store group results after calculation: %w", err)
	}

	return entityInstanceIDs, nil
}

// StoreGroupResults saves the calculated instance IDs for a group.
// It first clears any existing results for the group and then inserts the new ones.
func (s *GroupingService) StoreGroupResults(groupID string, instanceIDs []string) error {
	log.Printf("Storing %d results for groupID: %s", len(instanceIDs), groupID)
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for storing group results: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// 1. Delete existing results for this group
	_, err = tx.Exec("DELETE FROM group_results WHERE group_id = $1", groupID)
	if err != nil {
		return fmt.Errorf("failed to delete existing group results for group %s: %w", groupID, err)
	}
	log.Printf("Deleted existing results for groupID: %s", groupID)

	// 2. Insert new results
	if len(instanceIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO group_results (group_id, entity_instance_id, calculated_at) VALUES ($1, $2, $3)")
		if err != nil {
			return fmt.Errorf("failed to prepare insert statement for group results: %w", err)
		}
		defer stmt.Close()

		calculatedAt := time.Now().UTC() // Use one timestamp for the whole batch

		for _, instanceID := range instanceIDs {
			_, err := stmt.Exec(groupID, instanceID, calculatedAt)
			if err != nil {
				// If one insert fails, the transaction will be rolled back.
				return fmt.Errorf("failed to insert group result (groupID: %s, instanceID: %s): %w", groupID, instanceID, err)
			}
		}
		log.Printf("Inserted %d new results for groupID: %s", len(instanceIDs), groupID)
	} else {
		log.Printf("No instance IDs provided for groupID: %s. No new results to insert.", groupID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for storing group results: %w", err)
	}

	log.Printf("Successfully stored results for groupID: %s", groupID)
	return nil
}

// GetGroupResults retrieves the stored instance IDs and calculation timestamp for a group.
func (s *GroupingService) GetGroupResults(groupID string) ([]string, time.Time, error) {
	log.Printf("Fetching results for groupID: %s", groupID)

	rows, err := s.db.Query("SELECT entity_instance_id, calculated_at FROM group_results WHERE group_id = $1 ORDER BY calculated_at DESC", groupID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to query group_results for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var instanceIDs []string
	var calculatedAt time.Time
	var firstRow = true

	for rows.Next() {
		var instanceID string
		var currentCalculatedAt time.Time
		if err := rows.Scan(&instanceID, &currentCalculatedAt); err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to scan row from group_results for group %s: %w", groupID, err)
		}
		instanceIDs = append(instanceIDs, instanceID)
		if firstRow {
			calculatedAt = currentCalculatedAt // Capture the timestamp from the first row
			firstRow = false
		}
	}

	if err = rows.Err(); err != nil {
		return nil, time.Time{}, fmt.Errorf("error iterating rows from group_results for group %s: %w", groupID, err)
	}

	if len(instanceIDs) == 0 {
		log.Printf("No results found in group_results for groupID: %s", groupID)
		// Return zero time and empty slice, not an error, to distinguish "not found" from query failure
		return []string{}, time.Time{}, nil
	}

	log.Printf("Retrieved %d results for groupID: %s, calculated around %s", len(instanceIDs), groupID, calculatedAt.Format(time.RFC3339))
	return instanceIDs, calculatedAt, nil
}
