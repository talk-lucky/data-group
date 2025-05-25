package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// --- Rule Structs ---
type RuleCondition struct {
	Type          string      `json:"type"` 
	AttributeID   string      `json:"attribute_id"`
	AttributeName string      `json:"attribute_name"` 
	Operator      string      `json:"operator"`
	Value         interface{} `json:"value"`       
	ValueType     string      `json:"value_type"`  
}
type RuleGroup struct {
	Type      string            `json:"type"` 
	Condition string            `json:"condition"` 
	Rules     []json.RawMessage `json:"rules"`   
}
type GenericRule struct {
	Type string `json:"type"`
}

// --- Metadata Service Client ---
type MetadataServiceAPIClient interface {
	GetGroupDefinition(groupID string) (*GroupDefinition, error)
	GetEntityDefinition(entityID string) (*EntityDefinition, error)
	GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error)
	ListWorkflows() ([]WorkflowDefinition, error)
}
type HTTPMetadataClient struct {
	BaseURL    string
	HttpClient *http.Client
}
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	return &HTTPMetadataClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}
func (c *HTTPMetadataClient) fetchMetadata(url string, target interface{}) error {
	resp, err := c.HttpClient.Get(url)
	if err != nil { return fmt.Errorf("failed to GET %s: %w", url, err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("metadata service returned non-OK status %d for %s", resp.StatusCode, url) }
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil { return fmt.Errorf("failed to decode response from %s: %w", url, err) }
	return nil
}
func (c *HTTPMetadataClient) GetGroupDefinition(groupID string) (*GroupDefinition, error) {
	var groupDef GroupDefinition
	url := fmt.Sprintf("%s/api/v1/group-definitions/%s", c.BaseURL, groupID) // Updated path
	err := c.fetchMetadata(url, &groupDef)
	return &groupDef, err
}
func (c *HTTPMetadataClient) GetEntityDefinition(entityID string) (*EntityDefinition, error) {
	var entityDef EntityDefinition
	url := fmt.Sprintf("%s/api/v1/entities/%s", c.BaseURL, entityID)
	err := c.fetchMetadata(url, &entityDef)
	return &entityDef, err
}
func (c *HTTPMetadataClient) GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error) {
	var attrDef AttributeDefinition
	url := fmt.Sprintf("%s/api/v1/entities/%s/attributes/%s", c.BaseURL, entityID, attributeID)
	err := c.fetchMetadata(url, &attrDef)
	return &attrDef, err
}
func (c *HTTPMetadataClient) ListWorkflows() ([]WorkflowDefinition, error) {
	var workflows []WorkflowDefinition
	url := fmt.Sprintf("%s/api/v1/workflows", c.BaseURL)
	err := c.fetchMetadata(url, &workflows)
	return workflows, err
}

// --- Orchestration Service Client ---
type OrchestrationServiceAPIClient interface {
	TriggerWorkflow(workflowID string) error
}
type HTTPOrchestrationClient struct {
	BaseURL    string
	HttpClient *http.Client
}
func NewHTTPOrchestrationClient(baseURL string) *HTTPOrchestrationClient {
	return &HTTPOrchestrationClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 15 * time.Second},
	}
}
func (c *HTTPOrchestrationClient) TriggerWorkflow(workflowID string) error {
	url := fmt.Sprintf("%s/api/v1/orchestration/trigger/workflow/%s", c.BaseURL, workflowID)
	resp, err := c.HttpClient.Post(url, "application/json", nil)
	if err != nil { return fmt.Errorf("failed POST request to %s: %w", url, err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("orchestration service at %s returned status %d for workflow trigger %s", url, resp.StatusCode, workflowID) }
	log.Printf("Successfully triggered workflow %s via orchestration service at %s", workflowID, url)
	return nil
}

// --- Grouping Service ---
func initSchema(db *sql.DB) error {
	schemaStatements := []string{
		`CREATE TABLE IF NOT EXISTS group_calculation_logs (
            group_definition_id TEXT PRIMARY KEY,
            entity_definition_id TEXT,
            calculated_at TIMESTAMPTZ,
            member_count INTEGER,
            status TEXT, 
            error_message TEXT 
        );`, // Removed NULLABLE from error_message for UPSERT clarity
		`CREATE INDEX IF NOT EXISTS idx_gcl_entity_def_id ON group_calculation_logs(entity_definition_id);`,
		`CREATE INDEX IF NOT EXISTS idx_gcl_status ON group_calculation_logs(status);`,
		`CREATE TABLE IF NOT EXISTS group_memberships (
            group_definition_id TEXT NOT NULL,
            processed_entity_instance_id UUID NOT NULL,
            PRIMARY KEY (group_definition_id, processed_entity_instance_id),
            FOREIGN KEY (group_definition_id) REFERENCES group_calculation_logs(group_definition_id) ON DELETE CASCADE
        );`,
		`CREATE INDEX IF NOT EXISTS idx_gm_processed_entity_instance_id ON group_memberships(processed_entity_instance_id);`,
	}
	for i, stmt := range schemaStatements {
		_, err := db.Exec(stmt)
		if err != nil { return fmt.Errorf("failed to execute schema statement #%d for grouping tables: %s\nError: %w", i+1, stmt, err) }
	}
	log.Println("Schema for 'group_calculation_logs' and 'group_memberships' tables initialized successfully.")
	return nil
}
type GroupingService struct {
	metadataClient      MetadataServiceAPIClient
	orchestrationClient OrchestrationServiceAPIClient
	db                  *sql.DB
}
func NewGroupingService(metaClient MetadataServiceAPIClient, orchClient OrchestrationServiceAPIClient, db *sql.DB) *GroupingService {
	if db == nil { log.Panicf("GroupingService requires a valid database connection, but received nil.") }
	if err := initSchema(db); err != nil { log.Panicf("Failed to initialize database schema for GroupingService: %v", err) }
	return &GroupingService{
		metadataClient:      metaClient,
		orchestrationClient: orchClient,
		db:                  db,
	}
}
func getAllAttributeIDsAndNamesRecursive(ruleGroup RuleGroup, attrInfoMap map[string]string) error {
	for _, rawRule := range ruleGroup.Rules {
		var genericRule GenericRule
		if err := json.Unmarshal(rawRule, &genericRule); err != nil { return fmt.Errorf("failed to unmarshal generic rule: %w", err) }
		switch genericRule.Type {
		case "condition":
			var condition RuleCondition
			if err := json.Unmarshal(rawRule, &condition); err != nil { return fmt.Errorf("failed to unmarshal rule condition: %w", err) }
			if condition.AttributeID == "" || condition.AttributeName == "" { return fmt.Errorf("condition found with empty AttributeID ('%s') or AttributeName ('%s')", condition.AttributeID, condition.AttributeName) }
			attrInfoMap[condition.AttributeID] = condition.AttributeName
		case "group":
			var nestedGroup RuleGroup
			if err := json.Unmarshal(rawRule, &nestedGroup); err != nil { return fmt.Errorf("failed to unmarshal nested rule group: %w", err) }
			if err := getAllAttributeIDsAndNamesRecursive(nestedGroup, attrInfoMap); err != nil { return err }
		default:
			return fmt.Errorf("unknown rule type: %s", genericRule.Type)
		}
	}
	return nil
}
func buildWhereClauseRecursive(ruleGroup RuleGroup, attributeDefsMap map[string]*AttributeDefinition, params *[]interface{}, paramCounter *int) (string, error) {
	var conditions []string
	for _, rawRule := range ruleGroup.Rules {
		var genericRule GenericRule
		if err := json.Unmarshal(rawRule, &genericRule); err != nil { return "", fmt.Errorf("failed to determine rule type: %w", err) }
		if genericRule.Type == "condition" {
			var ruleCond RuleCondition
			if err := json.Unmarshal(rawRule, &ruleCond); err != nil { return "", fmt.Errorf("failed to unmarshal condition: %w", err) }
			attrDef, ok := attributeDefsMap[ruleCond.AttributeID]
			if !ok { return "", fmt.Errorf("attribute definition not found for ID: %s (Name: %s)", ruleCond.AttributeID, ruleCond.AttributeName) }
			valueType := ruleCond.ValueType; if valueType == "" { valueType = attrDef.DataType }
			fieldAccessor := fmt.Sprintf("(attributes->>'%s')", ruleCond.AttributeName)
			var castSuffix string
			switch strings.ToLower(valueType) {
			case "integer", "long": castSuffix = "::bigint"
			case "float", "double", "decimal", "numeric": castSuffix = "::numeric"
			case "boolean": castSuffix = "::boolean"
			case "date", "datetime", "timestamp": castSuffix = "::timestamptz"
			case "string", "text", "char", "varchar": castSuffix = ""
			default: log.Printf("Warning: Unhandled ValueType '%s' for attribute '%s'. No cast will be applied.", valueType, ruleCond.AttributeName); castSuffix = ""
			}
			if strings.HasSuffix(strings.ToLower(ruleCond.Operator), "null") { castSuffix = "" }
			var conditionStr string; op := strings.ToLower(ruleCond.Operator)
			switch op {
			case "=", "!=", ">", "<", ">=", "<=":
				if ruleCond.Value == nil { return "", fmt.Errorf("operator '%s' requires a non-null value for attribute '%s'", op, ruleCond.AttributeName) }
				conditionStr = fmt.Sprintf("%s%s %s $%d", fieldAccessor, castSuffix, op, *paramCounter); *params = append(*params, ruleCond.Value); *paramCounter++
			case "like", "not like", "ilike", "not ilike":
				if ruleCond.Value == nil { return "", fmt.Errorf("operator '%s' requires a non-null value for attribute '%s'", op, ruleCond.AttributeName) }
				valStr, ok := ruleCond.Value.(string); if !ok { return "", fmt.Errorf("value for operator '%s' must be a string for attribute '%s', got %T", op, ruleCond.AttributeName, ruleCond.Value) }
				conditionStr = fmt.Sprintf("%s %s $%d", fieldAccessor, op, *paramCounter); *params = append(*params, valStr); *paramCounter++
			case "contains", "does_not_contain":
				if ruleCond.Value == nil { return "", fmt.Errorf("operator '%s' requires a non-null value for attribute '%s'", op, ruleCond.AttributeName) }
				valStr, ok := ruleCond.Value.(string); if !ok { return "", fmt.Errorf("value for operator '%s' must be a string for attribute '%s', got %T", op, ruleCond.AttributeName, ruleCond.Value) }
				sqlOp := "LIKE"; if op == "does_not_contain" { sqlOp = "NOT LIKE"}
				conditionStr = fmt.Sprintf("%s %s $%d", fieldAccessor, sqlOp, *paramCounter); *params = append(*params, "%"+valStr+"%"); *paramCounter++
			case "in", "not in":
				values, ok := ruleCond.Value.([]interface{}); if !ok || len(values) == 0 { if op == "in" { conditionStr = "FALSE" } else { conditionStr = "TRUE" }; log.Printf("Operator '%s' for attribute '%s' received empty or non-array value. Resulting in %s.", op, ruleCond.AttributeName, conditionStr); break }
				var placeholders []string; for _, v := range values { placeholders = append(placeholders, fmt.Sprintf("$%d", *paramCounter)); *params = append(*params, v); *paramCounter++ }
				conditionStr = fmt.Sprintf("%s%s %s (%s)", fieldAccessor, castSuffix, strings.ToUpper(op), strings.Join(placeholders, ", "))
			case "is null", "is_null": conditionStr = fmt.Sprintf("%s IS NULL", fieldAccessor)
			case "is not null", "is_not_null": conditionStr = fmt.Sprintf("%s IS NOT NULL", fieldAccessor)
			default: return "", fmt.Errorf("unsupported operator: %s", ruleCond.Operator)
			}
			conditions = append(conditions, conditionStr)
		} else if genericRule.Type == "group" {
			var nestedGroup RuleGroup
			if err := json.Unmarshal(rawRule, &nestedGroup); err != nil { return "", fmt.Errorf("failed to unmarshal nested group: %w", err) }
			nestedSQL, err := buildWhereClauseRecursive(nestedGroup, attributeDefsMap, params, paramCounter); if err != nil { return "", err }
			if nestedSQL != "" { conditions = append(conditions, fmt.Sprintf("(%s)", nestedSQL)) }
		} else { return "", fmt.Errorf("unknown rule type: '%s'", genericRule.Type) }
	}
	if len(conditions) == 0 { return "", nil }
	return strings.Join(conditions, " "+strings.ToUpper(ruleGroup.Condition)+" "), nil
}

func (s *GroupingService) upsertGroupCalculationLog(tx *sql.Tx, groupID, entityDefID, status string, memberCount int, errorMsg sql.NullString) error {
	query := `
        INSERT INTO group_calculation_logs (group_definition_id, entity_definition_id, calculated_at, member_count, status, error_message)
        VALUES ($1, $2, NOW(), $3, $4, $5)
        ON CONFLICT (group_definition_id) DO UPDATE SET
            entity_definition_id = EXCLUDED.entity_definition_id,
            calculated_at = EXCLUDED.calculated_at,
            member_count = EXCLUDED.member_count,
            status = EXCLUDED.status,
            error_message = EXCLUDED.error_message;
    `
	_, err := tx.Exec(query, groupID, entityDefID, memberCount, status, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to upsert group calculation log for group %s: %w", groupID, err)
	}
	return nil
}

func (s *GroupingService) CalculateGroup(groupID string) ([]string, error) {
	log.Printf("Calculating group for groupID: %s", groupID)

	groupDef, err := s.metadataClient.GetGroupDefinition(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group definition for ID %s: %w", groupID, err)
	}
	log.Printf("Fetched GroupDefinition: %s (EntityID: %s)", groupDef.Name, groupDef.EntityID)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction for group calculation: %w", err)
	}
	// Defer rollback, which will only act if Commit is not called
	defer tx.Rollback()

	// Log "CALCULATING" status
	var initialErrorMsg sql.NullString // Represents NULL for error_message
	if err := s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "CALCULATING", 0, initialErrorMsg); err != nil {
		// tx.Rollback() is handled by defer
		return nil, fmt.Errorf("failed to log calculation start for group %s: %w", groupID, err)
	}

	// Clear previous members for this group
	_, err = tx.Exec("DELETE FROM group_memberships WHERE group_definition_id = $1", groupDef.ID)
	if err != nil {
		// tx.Rollback() is handled by defer
		// Attempt to log this failure before returning
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: fmt.Sprintf("failed to clear previous members: %v", err), Valid: true})
		// Try to commit this failure status. If this commit fails, the defer tx.Rollback() will handle it.
		_ = tx.Commit() 
		return nil, fmt.Errorf("failed to clear previous members for group %s: %w", groupID, err)
	}

	// Parse RulesJSON, fetch attributes, build and execute query (existing logic)
	var topRuleGroup RuleGroup
	if err := json.Unmarshal([]byte(groupDef.RulesJSON), &topRuleGroup); err != nil { /* ... error handling ... */ return nil, fmt.Errorf("failed to unmarshal RulesJSON for group %s: %w", groupID, err) }
	if topRuleGroup.Type != "group" && topRuleGroup.Type != "" {
		var singleCondition RuleCondition
		if errJ := json.Unmarshal([]byte(groupDef.RulesJSON), &singleCondition); errJ == nil && singleCondition.Type == "condition" {
			topRuleGroup = RuleGroup{Type: "group", Condition: "AND", Rules: []json.RawMessage{json.RawMessage(groupDef.RulesJSON)}}
		} else {
			if topRuleGroup.Condition == "" { return nil, fmt.Errorf("top-level rule group for group %s is missing 'condition'", groupID) }
			return nil, fmt.Errorf("invalid RulesJSON structure for group %s", groupID)
		}
	} else if topRuleGroup.Type == "" { topRuleGroup.Type = "group"; if topRuleGroup.Condition == "" { topRuleGroup.Condition = "AND" } }

	attrInfoMap := make(map[string]string)
	if err := getAllAttributeIDsAndNamesRecursive(topRuleGroup, attrInfoMap); err != nil { /* ... error handling ... */ return nil, fmt.Errorf("failed to extract attributes: %w", err) }
	attributeDefsMap := make(map[string]*AttributeDefinition)
	if len(attrInfoMap) > 0 {
		for attrID := range attrInfoMap {
			attrDef, errA := s.metadataClient.GetAttributeDefinition(groupDef.EntityID, attrID)
			if errA != nil { /* ... error handling ... */ return nil, fmt.Errorf("failed to get attr def %s: %w", attrID, errA) }
			attributeDefsMap[attrID] = attrDef
		}
	}
	var params []interface{}; paramCounter := 1
	params = append(params, groupDef.EntityID); paramCounter++
	whereClause, err := buildWhereClauseRecursive(topRuleGroup, attributeDefsMap, &params, &paramCounter)
	if err != nil { /* ... error handling ... */ return nil, fmt.Errorf("failed to build WHERE clause: %w", err) }

	var finalQuery strings.Builder
	finalQuery.WriteString("SELECT id FROM processed_entities WHERE entity_definition_id = $1")
	if whereClause != "" { finalQuery.WriteString(fmt.Sprintf(" AND (%s)", whereClause)) }
	sqlQueryStr := finalQuery.String()
	log.Printf("Executing member query for group %s: %s with params %v", groupID, sqlQueryStr, params)

	rows, err := s.db.Query(sqlQueryStr, params...) // Query using s.db, not tx, for read-only part if possible, or use tx if required
	if err != nil {
		errMsg := fmt.Sprintf("failed to execute member query for group %s: %v. Query: %s, Params: %v", groupID, err, sqlQueryStr, params)
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit() // Commit failure status
		return nil, fmt.Errorf(errMsg)
	}
	defer rows.Close()

	var entityInstanceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { /* ... error handling ... */ return nil, fmt.Errorf("failed to scan member row: %w", err) }
		entityInstanceIDs = append(entityInstanceIDs, id)
	}
	if err = rows.Err(); err != nil {
		errMsg := fmt.Errorf("error iterating member rows for group %s: %w", groupID, err).Error()
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit() // Commit failure status
		return nil, fmt.Errorf(errMsg)
	}
	log.Printf("Found %d entity instances for group %s", len(entityInstanceIDs), groupID)

	// Store new members
	if len(entityInstanceIDs) > 0 {
		memberStmt, errM := tx.Prepare("INSERT INTO group_memberships (group_definition_id, processed_entity_instance_id) VALUES ($1, $2)")
		if errM != nil {
			errMsg := fmt.Errorf("failed to prepare member insert statement for group %s: %w", groupID, errM).Error()
			_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
			_ = tx.Commit()
			return nil, fmt.Errorf(errMsg)
		}
		defer memberStmt.Close()
		for _, instanceID := range entityInstanceIDs {
			_, errI := memberStmt.Exec(groupDef.ID, instanceID)
			if errI != nil {
				errMsg := fmt.Errorf("failed to insert member %s for group %s: %w", instanceID, groupID, errI).Error()
				_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", len(entityInstanceIDs), sql.NullString{String: errMsg, Valid: true})
				_ = tx.Commit()
				return nil, fmt.Errorf(errMsg)
			}
		}
	}

	// Log "COMPLETED" status
	if err := s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "COMPLETED", len(entityInstanceIDs), sql.NullString{}); err != nil {
		// tx.Rollback() handled by defer, but this is a critical state.
		// If logging completion fails, the overall operation is problematic.
		return nil, fmt.Errorf("failed to log calculation completion for group %s: %w", groupID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction for group %s calculation: %w", groupID, err)
	}

	log.Printf("Successfully calculated and stored results for groupID: %s", groupID)
	go s.triggerLinkedWorkflows(groupID) // Trigger workflows after successful commit

	return entityInstanceIDs, nil
}

// StoreGroupResults method is now removed as its logic is integrated into CalculateGroup.

func (s *GroupingService) triggerLinkedWorkflows(groupID string) {
	log.Printf("Checking for workflows linked to groupID: %s", groupID)
	workflows, err := s.metadataClient.ListWorkflows()
	if err != nil { log.Printf("Error listing workflows for trigger check (group %s): %v", groupID, err); return }
	if len(workflows) == 0 { log.Printf("No workflows found to check for triggers for group %s.", groupID); return }
	log.Printf("Found %d workflows. Checking for 'on_group_update' for group %s...", len(workflows), groupID)
	for _, wf := range workflows {
		if wf.TriggerType == "on_group_update" && wf.IsEnabled {
			var triggerConf struct{ GroupID string `json:"group_id"` }
			if errJ := json.Unmarshal([]byte(wf.TriggerConfig), &triggerConf); errJ != nil { log.Printf("Error parsing trigger_config for workflow %s (%s): %v. Skipping.", wf.Name, wf.ID, errJ); continue }
			if triggerConf.GroupID == groupID {
				log.Printf("Workflow %s (%s) linked to group %s. Triggering...", wf.Name, wf.ID, groupID)
				if errT := s.orchestrationClient.TriggerWorkflow(wf.ID); errT != nil { log.Printf("Error triggering workflow %s for group %s: %v", wf.ID, groupID, errT) } else { log.Printf("Successfully triggered workflow %s for group %s.", wf.ID, groupID) }
			}
		}
	}
	log.Printf("Finished checking linked workflows for groupID: %s", groupID)
}

func (s *GroupingService) GetGroupResults(groupID string) ([]string, time.Time, error) {
	log.Printf("Fetching results for groupID: %s", groupID)

	var calculatedAt time.Time
	var status string
	var memberCount int // Store member_count from log to potentially cross-check, though len(instanceIDs) is primary.

	logQuery := "SELECT calculated_at, status, member_count FROM group_calculation_logs WHERE group_definition_id = $1 ORDER BY calculated_at DESC LIMIT 1"
	err := s.db.QueryRow(logQuery, groupID).Scan(&calculatedAt, &status, &memberCount)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No calculation log found for groupID: %s", groupID)
			return []string{}, time.Time{}, nil // Or return an error indicating not found/not calculated
		}
		return nil, time.Time{}, fmt.Errorf("failed to query group_calculation_logs for group %s: %w", groupID, err)
	}

	if strings.ToUpper(status) != "COMPLETED" {
		log.Printf("Group %s last calculation status was '%s', not 'COMPLETED'. Returning empty results.", groupID, status)
		return []string{}, calculatedAt, fmt.Errorf("last calculation for group %s was not successful (status: %s)", groupID, status)
	}

	// If status is COMPLETED, fetch members
	rows, err := s.db.Query("SELECT processed_entity_instance_id FROM group_memberships WHERE group_definition_id = $1", groupID)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to query group_memberships for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var instanceIDs []string
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to scan row from group_memberships for group %s: %w", groupID, err)
		}
		instanceIDs = append(instanceIDs, instanceID)
	}
	if err = rows.Err(); err != nil {
		return nil, time.Time{}, fmt.Errorf("error iterating rows from group_memberships for group %s: %w", groupID, err)
	}

	// Cross-check member count (optional, for consistency)
	if memberCount != len(instanceIDs) {
		log.Printf("Warning: Mismatch in member count for group %s. Log says %d, actual members found %d.", groupID, memberCount, len(instanceIDs))
	}

	log.Printf("Retrieved %d members for groupID: %s, calculated at %s", len(instanceIDs), groupID, calculatedAt.Format(time.RFC3339))
	return instanceIDs, calculatedAt, nil
}

// --- Data model structs ---
type GroupDefinition struct { ID string `json:"id"`; Name string `json:"name"`; EntityID string `json:"entity_id"`; RulesJSON string `json:"rules_json"`; Description string `json:"description,omitempty"` }
type EntityDefinition struct { ID string `json:"id"`; Name string `json:"name"` }
type AttributeDefinition struct { ID string `json:"id"`; EntityID string `json:"entity_id"`; Name string `json:"name"`; DataType string `json:"data_type"` }
type WorkflowDefinition struct { ID string `json:"id"`; Name string `json:"name"`; TriggerType string `json:"trigger_type"`; TriggerConfig string `json:"trigger_config"`; IsEnabled bool `json:"is_enabled"` }
