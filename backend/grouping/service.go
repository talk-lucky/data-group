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
// RuleCondition defines a single rule based on an attribute.
type RuleCondition struct {
	// Type field is implicit ("condition") and used for unmarshalling, not stored in this struct.
	AttributeID   string      `json:"attribute_id"`
	AttributeName string      `json:"attribute_name"`
	EntityID      string      `json:"entity_id"` // Added EntityID
	Operator      string      `json:"operator"`
	Value         interface{} `json:"value"`
	ValueType     string      `json:"value_type"`
}

// RuleGroup defines a logical grouping of rules or other rule groups.
type RuleGroup struct {
	Type            string            `json:"type"` // "group"
	EntityID        string            `json:"entity_id,omitempty"` // Primary entity for this group
	LogicalOperator string            `json:"logical_operator"`
	Rules           []json.RawMessage `json:"rules"`
}

// RelationshipGroupNode defines a rule based on a relationship to another entity.
type RelationshipGroupNode struct {
	Type               string          `json:"type"` // "relationship_group"
	RelationshipID     string          `json:"relationship_id"`
	RelatedEntityRules json.RawMessage `json:"related_entity_rules"` // Will be unmarshalled into a RuleGroup
	// Optional: JoinType string `json:"join_type,omitempty"` // e.g., EXISTS, NOT_EXISTS, AT_LEAST_ONE_MATCH
}

// RelatedAttributeCondition defines a condition on an attribute of a related entity.
type RelatedAttributeCondition struct {
	Type             string      `json:"type"` // "related_attribute_condition"
	RelationshipID   string      `json:"relationship_id"`   // ID of the EntityRelationshipDefinition
	AttributeID      string      `json:"attribute_id"`      // AttributeID on the TARGET entity of the relationship
	AttributeName    string      `json:"attribute_name"`    // Name of the attribute on the TARGET entity
	Operator         string      `json:"operator"`
	Value            interface{} `json:"value"`
	ValueType        string      `json:"value_type"`        // Data type of the value for casting, e.g., "STRING", "INTEGER"
}

// GenericRule is used to determine the type of a rule before full unmarshalling.
type GenericRule struct {
	Type string `json:"type"`
}

// RelationshipType defines the nature of the connection between two entities.
// Mirrored from metadata/models.go for use by the grouping service.
type RelationshipType string

const (
	OneToOne  RelationshipType = "ONE_TO_ONE"
	OneToMany RelationshipType = "ONE_TO_MANY"
	ManyToOne RelationshipType = "MANY_TO_ONE"
)

// EntityRelationshipDefinition defines how two entities are related.
// Mirrored from metadata/models.go for use by the grouping service.
type EntityRelationshipDefinition struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	SourceEntityID    string           `json:"source_entity_id"`
	SourceAttributeID string           `json:"source_attribute_id"`
	TargetEntityID    string           `json:"target_entity_id"`
	TargetAttributeID string           `json:"target_attribute_id"`
	RelationshipType  RelationshipType `json:"relationship_type"`
}


// --- Metadata Service Client ---
type MetadataServiceAPIClient interface {
	GetGroupDefinition(groupID string) (*GroupDefinition, error)
	GetEntityDefinition(entityID string) (*EntityDefinition, error)
	GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error)
	GetEntityRelationship(relationshipID string) (*EntityRelationshipDefinition, error) // Added
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
func (c *HTTPMetadataClient) GetEntityRelationship(relationshipID string) (*EntityRelationshipDefinition, error) {
	var relDef EntityRelationshipDefinition
	url := fmt.Sprintf("%s/api/v1/entity-relationships/%s", c.BaseURL, relationshipID)
	err := c.fetchMetadata(url, &relDef)
	return &relDef, err
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
// getAllAttributeIDsAndNamesRecursive extracts all unique AttributeIDs, their names, and associated EntityIDs
// from a rule group and its nested structures.
// It populates attrInfoMap with AttributeID as key and AttributeName as value.
// It also populates entityAttrMap with EntityID as key and a map of its AttributeIDs to AttributeNames.
// It needs metadataClient to fetch relationship definitions if a relationship_group is encountered.
// It also collects all relationship IDs found into relationshipIDsMap.
func getAllAttributeIDsAndNamesRecursive(
	ruleGroup RuleGroup,
	attrInfoMap map[string]string, // AttributeID -> AttributeName
	entityAttrMap map[string]map[string]string, // EntityID -> map[AttributeID]AttributeName
	relationshipIDsMap map[string]bool, // Set of RelationshipIDs encountered
	metadataClient MetadataServiceAPIClient, 
	currentEntityIDForGroup string, 
) error {
	// If ruleGroup has its own EntityID, it defines the context for its direct children (conditions).
	// If not, it inherits the context from its parent (currentEntityIDForGroup).
	contextEntityID := currentEntityIDForGroup
	if ruleGroup.EntityID != "" {
		contextEntityID = ruleGroup.EntityID
	}
	if contextEntityID == "" {
        return fmt.Errorf("cannot determine entity context for rule group processing: ruleGroup.EntityID is '%s' and currentEntityIDForGroup is '%s'", ruleGroup.EntityID, currentEntityIDForGroup)
    }


	for _, rawRule := range ruleGroup.Rules {
		var genericRule GenericRule
		if err := json.Unmarshal(rawRule, &genericRule); err != nil {
			return fmt.Errorf("failed to unmarshal generic rule: %w", err)
		}

		switch genericRule.Type {
		case "condition":
			var condition RuleCondition
			if err := json.Unmarshal(rawRule, &condition); err != nil {
				return fmt.Errorf("failed to unmarshal rule condition: %w", err)
			}
			if condition.AttributeID == "" || condition.AttributeName == "" {
				return fmt.Errorf("condition found with empty AttributeID ('%s') or AttributeName ('%s')", condition.AttributeID, condition.AttributeName)
			}
			
			// Condition's own EntityID takes precedence. If not set, it uses the group's contextEntityID.
			conditionOnEntityID := condition.EntityID
			if conditionOnEntityID == "" {
				conditionOnEntityID = contextEntityID
			}
			if conditionOnEntityID == "" { // Should be caught by contextEntityID check above, but as safeguard.
                 return fmt.Errorf("condition for attribute '%s' is missing EntityID and parent group context also has no EntityID", condition.AttributeName)
            }

			attrInfoMap[condition.AttributeID] = condition.AttributeName // Store globally for name lookup
			if _, ok := entityAttrMap[conditionOnEntityID]; !ok {
				entityAttrMap[conditionOnEntityID] = make(map[string]string)
			}
			entityAttrMap[conditionOnEntityID][condition.AttributeID] = condition.AttributeName

		case "group":
			var nestedGroup RuleGroup
			if err := json.Unmarshal(rawRule, &nestedGroup); err != nil {
				return fmt.Errorf("failed to unmarshal nested rule group: %w", err)
			}
			// Nested group inherits contextEntityID if it doesn't define its own.
			nestedGroupContextEntityID := nestedGroup.EntityID
			if nestedGroupContextEntityID == "" {
				nestedGroupContextEntityID = contextEntityID
			}
			if err := getAllAttributeIDsAndNamesRecursive(nestedGroup, attrInfoMap, entityAttrMap, relationshipIDsMap, metadataClient, nestedGroupContextEntityID); err != nil {
				return err
			}
		
		case "relationship_group":
			var relGroupNode RelationshipGroupNode
			if err := json.Unmarshal(rawRule, &relGroupNode); err != nil {
				return fmt.Errorf("failed to unmarshal relationship_group node: %w", err)
			}
			if metadataClient == nil { // Should be ensured by the caller (CalculateGroup)
				return fmt.Errorf("metadataClient is nil, cannot process relationship_group %s", relGroupNode.RelationshipID)
			}
			
			relationshipIDsMap[relGroupNode.RelationshipID] = true // Collect relationship ID

			// Temporarily fetch relationship here to know TargetEntityID for recursive call.
			// Later, these will be pre-fetched in CalculateGroup.
			relDef, err := metadataClient.GetEntityRelationship(relGroupNode.RelationshipID)
			if err != nil {
				return fmt.Errorf("failed to pre-fetch entity relationship %s for attribute collection: %w", relGroupNode.RelationshipID, err)
			}
			
			// Attributes for join condition also need to be collected.
			// Source attribute is on contextEntityID (or relDef.SourceEntityID if more precise)
			// Target attribute is on relDef.TargetEntityID
			attrInfoMap[relDef.SourceAttributeID] = "" // Name to be fetched later
			if _, ok := entityAttrMap[relDef.SourceEntityID]; !ok { entityAttrMap[relDef.SourceEntityID] = make(map[string]string) }
			entityAttrMap[relDef.SourceEntityID][relDef.SourceAttributeID] = ""

			attrInfoMap[relDef.TargetAttributeID] = "" // Name to be fetched later
			if _, ok := entityAttrMap[relDef.TargetEntityID]; !ok { entityAttrMap[relDef.TargetEntityID] = make(map[string]string) }
			entityAttrMap[relDef.TargetEntityID][relDef.TargetAttributeID] = ""
			
			var relatedRulesGroup RuleGroup
			if err := json.Unmarshal(relGroupNode.RelatedEntityRules, &relatedRulesGroup); err != nil {
				return fmt.Errorf("failed to unmarshal related_entity_rules for relationship %s: %w", relGroupNode.RelationshipID, err)
			}
			
			// The related_entity_rules operate on the TargetEntity of the relationship.
			// If relatedRulesGroup.EntityID is specified, it MUST match relDef.TargetEntityID.
			if relatedRulesGroup.EntityID != "" && relatedRulesGroup.EntityID != relDef.TargetEntityID {
				return fmt.Errorf("relationship_group for rel %s: related_entity_rules has entity_id %s which does not match relationship target_entity_id %s", relGroupNode.RelationshipID, relatedRulesGroup.EntityID, relDef.TargetEntityID)
			}
			// Pass relDef.TargetEntityID as the context for the related rules.
			if err := getAllAttributeIDsAndNamesRecursive(relatedRulesGroup, attrInfoMap, entityAttrMap, relationshipIDsMap, metadataClient, relDef.TargetEntityID); err != nil {
				return err
			}
		
		case "related_attribute_condition":
			var condition RelatedAttributeCondition
			if err := json.Unmarshal(rawRule, &condition); err != nil {
				return fmt.Errorf("failed to unmarshal related_attribute_condition: %w", err)
			}
			if metadataClient == nil {
				return fmt.Errorf("metadataClient is nil, cannot process related_attribute_condition for relationship %s", condition.RelationshipID)
			}

			relationshipIDsMap[condition.RelationshipID] = true // Collect relationship ID

			relDef, err := metadataClient.GetEntityRelationship(condition.RelationshipID)
			if err != nil {
				return fmt.Errorf("failed to fetch entity relationship %s for related_attribute_condition: %w", condition.RelationshipID, err)
			}

			// Attribute being conditioned on (belongs to target entity)
			targetEntityID := relDef.TargetEntityID
			if condition.AttributeID == "" || condition.AttributeName == "" {
                 return fmt.Errorf("related_attribute_condition for relationship '%s' found with empty AttributeID ('%s') or AttributeName ('%s')", condition.RelationshipID, condition.AttributeID, condition.AttributeName)
            }
			attrInfoMap[condition.AttributeID] = condition.AttributeName
			if _, ok := entityAttrMap[targetEntityID]; !ok {
				entityAttrMap[targetEntityID] = make(map[string]string)
			}
			entityAttrMap[targetEntityID][condition.AttributeID] = condition.AttributeName

			// Collect join attributes from the relationship definition itself
			// Source join attribute (belongs to source entity of relationship)
			sourceJoinEntityID := relDef.SourceEntityID
			if relDef.SourceAttributeID == "" {
				return fmt.Errorf("relationship definition %s is missing SourceAttributeID", relDef.ID)
			}
			attrInfoMap[relDef.SourceAttributeID] = "" // Name will be filled when attributeDefsMap is populated
			if _, ok := entityAttrMap[sourceJoinEntityID]; !ok {
				entityAttrMap[sourceJoinEntityID] = make(map[string]string)
			}
			entityAttrMap[sourceJoinEntityID][relDef.SourceAttributeID] = "" 

			// Target join attribute (belongs to target entity of relationship)
			if relDef.TargetAttributeID == "" {
				return fmt.Errorf("relationship definition %s is missing TargetAttributeID", relDef.ID)
			}
			attrInfoMap[relDef.TargetAttributeID] = "" // Name will be filled
			if _, ok := entityAttrMap[targetEntityID]; !ok { // Target entity map might have been created above
				entityAttrMap[targetEntityID] = make(map[string]string)
			}
			entityAttrMap[targetEntityID][relDef.TargetAttributeID] = "" 

		default:
			return fmt.Errorf("unknown rule type: '%s' in rule group for entity '%s'", genericRule.Type, contextEntityID)
		}
	}
	return nil
}


// buildWhereClauseRecursive constructs the SQL WHERE clause.
// - ruleGroup: The current group of rules to process.
// - attributeDefsMap: Pre-fetched map of AttributeID to AttributeDefinition for all relevant attributes.
// - relationshipDefsMap: Pre-fetched map of RelationshipID to EntityRelationshipDefinition for all relevant relationships.
// - params: Slice to append SQL parameters to.
// - paramCounter: Pointer to the current SQL parameter placeholder number (e.g., $1, $2).
// - currentTableAlias: The SQL alias for the processed_entities table relevant to this ruleGroup.
// - aliasGenerator: A function or struct method to generate new unique table aliases for subqueries/joins.
// - groupEntityID: The entity_id context for the current ruleGroup.
// - metadataClient: Used to fetch relationship definitions if not pre-fetched.
func buildWhereClauseRecursive(
	ruleGroup RuleGroup,
	attributeDefsMap map[string]*AttributeDefinition,
	relationshipDefsMap map[string]*EntityRelationshipDefinition, 
	params *[]interface{},
	paramCounter *int,
	currentTableAlias string, 
	aliasGenerator func() string, 
	groupEntityID string, // Entity context for this specific group.
	metadataClient MetadataServiceAPIClient, // Needed if relationships are not pre-fetched.
) (string, error) {
	var conditions []string

	if currentTableAlias == "" {
		return "", fmt.Errorf("buildWhereClauseRecursive called with empty currentTableAlias for group with EntityID '%s'", groupEntityID)
	}
	// The entity_id for conditions directly under this group.
	// If ruleGroup.EntityID is set, it defines the context. Otherwise, it inherits from groupEntityID.
	contextualEntityID := groupEntityID
	if ruleGroup.EntityID != "" {
		contextualEntityID = ruleGroup.EntityID
	}
    if contextualEntityID == "" {
        return "", fmt.Errorf("cannot determine entity context for rule group processing (ruleGroup.EntityID: '%s', groupEntityID: '%s')", ruleGroup.EntityID, groupEntityID)
    }


	for _, rawRule := range ruleGroup.Rules {
		var genericRule GenericRule
		if err := json.Unmarshal(rawRule, &genericRule); err != nil { return "", fmt.Errorf("failed to determine rule type from rawRule: %s, error: %w", string(rawRule), err) }

		if genericRule.Type == "condition" {
			var ruleCond RuleCondition
			if err := json.Unmarshal(rawRule, &ruleCond); err != nil { return "", fmt.Errorf("failed to unmarshal condition: %w", err) }
			
			// Condition's EntityID field takes precedence. If empty, use the group's contextualEntityID.
			conditionOnEntityID := ruleCond.EntityID
			if conditionOnEntityID == "" {
				conditionOnEntityID = contextualEntityID
			}
            if conditionOnEntityID != contextualEntityID {
                 // This condition is trying to query an attribute from an entity (conditionOnEntityID)
                 // that is different from the current group's context (contextualEntityID)
                 // without being wrapped in a relationship_group. This is not allowed.
                 return "", fmt.Errorf("condition for attribute '%s' (on EntityID '%s') found in a group context for EntityID '%s'. Cross-entity conditions must use 'relationship_group'.", ruleCond.AttributeName, conditionOnEntityID, contextualEntityID)
            }

			attrDef, ok := attributeDefsMap[ruleCond.AttributeID]
			if !ok {
				return "", fmt.Errorf("attribute definition not found for ID: '%s' (Name: '%s', EntityID: '%s')", ruleCond.AttributeID, ruleCond.AttributeName, conditionOnEntityID)
			}
			if attrDef.EntityID != conditionOnEntityID { // Sanity check
                 return "", fmt.Errorf("attribute definition mismatch: attrDef.EntityID ('%s') != conditionOnEntityID ('%s') for attribute '%s'", attrDef.EntityID, conditionOnEntityID, attrDef.Name)
            }

			valueType := ruleCond.ValueType
			if valueType == "" { valueType = attrDef.DataType } 

			fieldAccessor := fmt.Sprintf("(%s.attributes->>'%s')", currentTableAlias, ruleCond.AttributeName)
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
			default: return "", fmt.Errorf("unsupported operator: '%s' for attribute '%s'", ruleCond.Operator, ruleCond.AttributeName)
			}
			conditions = append(conditions, conditionStr)

		} else if genericRule.Type == "group" {
			var nestedRuleGroup RuleGroup
			if err := json.Unmarshal(rawRule, &nestedRuleGroup); err != nil { return "", fmt.Errorf("failed to unmarshal nested group: %w", err) }
			
			// Nested group inherits currentTableAlias and contextualEntityID if it doesn't specify its own EntityID.
			// If nestedRuleGroup.EntityID is different, it implies a change of entity context NOT via a relationship.
			// This should ideally be an error or handled very carefully. For now, assume simple nesting on same entity context.
			nestedGroupEntityID := nestedRuleGroup.EntityID
			if nestedGroupEntityID == "" {
				nestedGroupEntityID = contextualEntityID
			}
			if nestedGroupEntityID != contextualEntityID {
				return "", fmt.Errorf("nested group has EntityID '%s' different from parent context '%s' without 'relationship_group'", nestedGroupEntityID, contextualEntityID)
			}

			// Recursive call for a standard nested group. It operates on the same currentTableAlias.
			nestedSQL, err := buildWhereClauseRecursive(nestedRuleGroup, attributeDefsMap, relationshipDefsMap, params, paramCounter, currentTableAlias, aliasGenerator, nestedGroupEntityID, metadataClient)
			if err != nil { return "", err }
			if nestedSQL != "" { conditions = append(conditions, fmt.Sprintf("(%s)", nestedSQL)) }
		
		} else if genericRule.Type == "relationship_group" {
			var relGroupNode RelationshipGroupNode
			if err := json.Unmarshal(rawRule, &relGroupNode); err != nil { return "", fmt.Errorf("failed to unmarshal relationship_group node: %w", err) }

			relDef, ok := relationshipDefsMap[relGroupNode.RelationshipID]
			if !ok {
				// Attempt to fetch if not in map (should ideally be pre-fetched by CalculateGroup)
				fetchedRelDef, err := metadataClient.GetEntityRelationship(relGroupNode.RelationshipID)
				if err != nil {
					return "", fmt.Errorf("relationship definition %s not found and failed to fetch: %w", relGroupNode.RelationshipID, err)
				}
				relDef = fetchedRelDef
				relationshipDefsMap[relGroupNode.RelationshipID] = fetchedRelDef // Cache it
			}

			// Validate relationship is applicable
			if relDef.SourceEntityID != contextualEntityID { // Current group's entity must be the source of the relationship
				return "", fmt.Errorf("relationship %s (source: %s) cannot originate from group with entity context %s", relDef.ID, relDef.SourceEntityID, contextualEntityID)
			}

			sourceAttr, okSA := attributeDefsMap[relDef.SourceAttributeID]
			targetAttr, okTA := attributeDefsMap[relDef.TargetAttributeID]
			if !okSA || !okTA {
				return "", fmt.Errorf("source ('%s') or target ('%s') attribute definition for relationship %s not found in attributeDefsMap", relDef.SourceAttributeID, relDef.TargetAttributeID, relDef.ID)
			}

			relatedTableAlias := aliasGenerator() // Generate a new alias for the related entity table, e.g., "pe2"
			
			var relatedActualRuleGroup RuleGroup
			if err := json.Unmarshal(relGroupNode.RelatedEntityRules, &relatedActualRuleGroup); err != nil {
				return "", fmt.Errorf("failed to unmarshal related_entity_rules for relationship %s: %w", relGroupNode.RelationshipID, err)
			}
			// Ensure the related rules group is for the target entity of the relationship
			if relatedActualRuleGroup.EntityID != "" && relatedActualRuleGroup.EntityID != relDef.TargetEntityID {
                 return "", fmt.Errorf("related_entity_rules for relationship %s has EntityID '%s' which does not match relationship's TargetEntityID '%s'", relGroupNode.RelationshipID, relatedActualRuleGroup.EntityID, relDef.TargetEntityID)
            }


			// Recursively build WHERE clause for the related entity's rules
			// This clause will apply to 'relatedTableAlias'
			relatedWhereClause, err := buildWhereClauseRecursive(relatedActualRuleGroup, attributeDefsMap, relationshipDefsMap, params, paramCounter, relatedTableAlias, aliasGenerator, relDef.TargetEntityID, metadataClient)
			if err != nil { return "", fmt.Errorf("failed to build WHERE clause for related entity (rel: %s): %w", relDef.ID, err) }

			// Construct the EXISTS subquery
			// Example: AND EXISTS (SELECT 1 FROM processed_entities pe2 WHERE pe2.entity_definition_id = $N AND (pe1.attributes->>'fk_attr') = (pe2.attributes->>'pk_attr') AND (related_conditions_on_pe2))
			var subQuery strings.Builder
			subQuery.WriteString(fmt.Sprintf("EXISTS (SELECT 1 FROM processed_entities %s WHERE %s.entity_definition_id = $%d", relatedTableAlias, relatedTableAlias, *paramCounter))
			*params = append(*params, relDef.TargetEntityID) // Parameter for target_entity_id
			*paramCounter++
			
			// Join condition using attributes from the relationship
			// (currentTableAlias.attributes->>'SourceAttributeName') = (relatedTableAlias.attributes->>'TargetAttributeName')
			subQuery.WriteString(fmt.Sprintf(" AND (%s.attributes->>'%s') = (%s.attributes->>'%s')", currentTableAlias, sourceAttr.Name, relatedTableAlias, targetAttr.Name))

			if relatedWhereClause != "" {
				subQuery.WriteString(fmt.Sprintf(" AND (%s)", relatedWhereClause))
			}
			subQuery.WriteString(")")
			conditions = append(conditions, subQuery.String())

		} else if genericRule.Type == "related_attribute_condition" {
			var relCond RelatedAttributeCondition
			if err := json.Unmarshal(rawRule, &relCond); err != nil {
				return "", fmt.Errorf("failed to unmarshal related_attribute_condition: %w", err)
			}

			relDef, ok := relationshipDefsMap[relCond.RelationshipID]
			if !ok {
				return "", fmt.Errorf("relationship definition %s not found in pre-fetched map for RelatedAttributeCondition", relCond.RelationshipID)
			}

			if relDef.SourceEntityID != contextualEntityID {
				return "", fmt.Errorf("RelatedAttributeCondition: relationship %s (source: %s) cannot originate from group with entity context %s", relDef.ID, relDef.SourceEntityID, contextualEntityID)
			}

			sourceJoinAttrDef, okSA := attributeDefsMap[relDef.SourceAttributeID]
			if !okSA {
				return "", fmt.Errorf("RelatedAttributeCondition: source join attribute definition %s (for relationship %s) not found", relDef.SourceAttributeID, relDef.ID)
			}
			targetJoinAttrDef, okTA := attributeDefsMap[relDef.TargetAttributeID]
			if !okTA {
				return "", fmt.Errorf("RelatedAttributeCondition: target join attribute definition %s (for relationship %s) not found", relDef.TargetAttributeID, relDef.ID)
			}

			conditionedAttrDef, okCA := attributeDefsMap[relCond.AttributeID]
			if !okCA {
				return "", fmt.Errorf("RelatedAttributeCondition: conditioned attribute definition %s (Name: %s) on target entity %s not found", relCond.AttributeID, relCond.AttributeName, relDef.TargetEntityID)
			}
			if conditionedAttrDef.EntityID != relDef.TargetEntityID { // Sanity check
                 return "", fmt.Errorf("RelatedAttributeCondition: conditioned attribute %s (EntityID %s) does not belong to target entity %s of relationship %s", conditionedAttrDef.Name, conditionedAttrDef.EntityID, relDef.TargetEntityID, relDef.ID)
            }
			// Also check if provided AttributeName matches the definition, if AttributeName is part of relCond
            if relCond.AttributeName != "" && relCond.AttributeName != conditionedAttrDef.Name {
                log.Printf("Warning: RelatedAttributeCondition for AttrID '%s' has Name '%s', but definition has Name '%s'. Using definition's name.", relCond.AttributeID, relCond.AttributeName, conditionedAttrDef.Name)
                // Potentially return an error here if strict matching is required.
            }


			newRelatedTableAlias := aliasGenerator()
			var subQuery strings.Builder
			subQuery.WriteString(fmt.Sprintf("EXISTS (SELECT 1 FROM processed_entities %s WHERE ", newRelatedTableAlias))

			// Condition 1: Target Entity Type
			subQuery.WriteString(fmt.Sprintf("%s.entity_definition_id = $%d", newRelatedTableAlias, *paramCounter))
			*params = append(*params, relDef.TargetEntityID)
			*paramCounter++

			// Condition 2: Join Condition
			subQuery.WriteString(fmt.Sprintf(" AND (%s.attributes->>'%s') = (%s.attributes->>'%s')",
				currentTableAlias, sourceJoinAttrDef.Name,
				newRelatedTableAlias, targetJoinAttrDef.Name))

			// Condition 3: Actual Related Attribute Condition
			valueType := relCond.ValueType
			if valueType == "" { valueType = conditionedAttrDef.DataType }

			fieldAccessor := fmt.Sprintf("(%s.attributes->>'%s')", newRelatedTableAlias, conditionedAttrDef.Name) // Use conditionedAttrDef.Name for safety
			var castSuffix string
			switch strings.ToLower(valueType) {
			case "integer", "long": castSuffix = "::bigint"
			case "float", "double", "decimal", "numeric": castSuffix = "::numeric"
			case "boolean": castSuffix = "::boolean"
			case "date", "datetime", "timestamp": castSuffix = "::timestamptz"
			case "string", "text", "char", "varchar": castSuffix = ""
			default: log.Printf("Warning (RelatedAttributeCondition): Unhandled ValueType '%s' for attribute '%s'. No cast.", valueType, conditionedAttrDef.Name); castSuffix = ""
			}
			if strings.HasSuffix(strings.ToLower(relCond.Operator), "null") { castSuffix = "" }
			
			var relatedAttrCondStr string
			op := strings.ToLower(relCond.Operator)
			switch op {
			case "=", "!=", ">", "<", ">=", "<=":
				if relCond.Value == nil { return "", fmt.Errorf("RelatedAttributeCondition: operator '%s' requires non-null value for attribute '%s'", op, conditionedAttrDef.Name) }
				relatedAttrCondStr = fmt.Sprintf("%s%s %s $%d", fieldAccessor, castSuffix, op, *paramCounter)
				*params = append(*params, relCond.Value)
				*paramCounter++
			case "like", "not like", "ilike", "not ilike":
				if relCond.Value == nil { return "", fmt.Errorf("RelatedAttributeCondition: operator '%s' requires non-null value for attribute '%s'", op, conditionedAttrDef.Name) }
				valStr, okV := relCond.Value.(string); if !okV { return "", fmt.Errorf("RelatedAttributeCondition: value for operator '%s' must be string for attribute '%s', got %T", op, conditionedAttrDef.Name, relCond.Value) }
				relatedAttrCondStr = fmt.Sprintf("%s %s $%d", fieldAccessor, op, *paramCounter)
				*params = append(*params, valStr)
				*paramCounter++
			case "contains", "does_not_contain":
                 if relCond.Value == nil { return "", fmt.Errorf("RelatedAttributeCondition: operator '%s' requires non-null value for attribute '%s'", op, conditionedAttrDef.Name) }
                 valStr, okV := relCond.Value.(string); if !okV { return "", fmt.Errorf("RelatedAttributeCondition: value for operator '%s' must be string for attribute '%s', got %T", op, conditionedAttrDef.Name, relCond.Value) }
                 sqlOp := "LIKE"; if op == "does_not_contain" { sqlOp = "NOT LIKE"}
                 relatedAttrCondStr = fmt.Sprintf("%s %s $%d", fieldAccessor, sqlOp, *paramCounter); *params = append(*params, "%"+valStr+"%"); *paramCounter++
			case "in", "not in":
				values, okV := relCond.Value.([]interface{}); if !okV || len(values) == 0 { 
					if op == "in" { relatedAttrCondStr = "FALSE" } else { relatedAttrCondStr = "TRUE" }
					log.Printf("RelatedAttributeCondition: Operator '%s' for attribute '%s' received empty/non-array value. Resulting in %s.", op, conditionedAttrDef.Name, relatedAttrCondStr)
					break 
				}
				var placeholders []string
				for _, v := range values { 
					placeholders = append(placeholders, fmt.Sprintf("$%d", *paramCounter))
					*params = append(*params, v)
					*paramCounter++ 
				}
				relatedAttrCondStr = fmt.Sprintf("%s%s %s (%s)", fieldAccessor, castSuffix, strings.ToUpper(op), strings.Join(placeholders, ", "))
			case "is null", "is_null": relatedAttrCondStr = fmt.Sprintf("%s IS NULL", fieldAccessor)
			case "is not null", "is_not_null": relatedAttrCondStr = fmt.Sprintf("%s IS NOT NULL", fieldAccessor)
			default: return "", fmt.Errorf("RelatedAttributeCondition: unsupported operator: '%s' for attribute '%s'", relCond.Operator, conditionedAttrDef.Name)
			}
			
			if relatedAttrCondStr != "" {
				subQuery.WriteString(fmt.Sprintf(" AND (%s)", relatedAttrCondStr))
			}
			subQuery.WriteString(")")
			conditions = append(conditions, subQuery.String())

		} else { 
			return "", fmt.Errorf("unknown rule type: '%s' in group for EntityID '%s'", genericRule.Type, contextualEntityID)
		}
	}

	if len(conditions) == 0 { return "", nil } 
	
	logicalOp := " AND " 
	if ruleGroup.LogicalOperator != "" {
		logicalOp = " " + strings.ToUpper(ruleGroup.LogicalOperator) + " "
	}
	return strings.Join(conditions, logicalOp), nil
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

	// Parse RulesJSON
	var topRuleGroup RuleGroup
	if err := json.Unmarshal([]byte(groupDef.RulesJSON), &topRuleGroup); err != nil {
		errMsg := fmt.Sprintf("failed to unmarshal RulesJSON for group %s: %w. Content: %s", groupID, err, groupDef.RulesJSON)
		// Log and commit failure, then return
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit()
		return nil, fmt.Errorf(errMsg)
	}

	// Validate topRuleGroup structure
	if topRuleGroup.Type != "group" {
		// This specific handling for a single top-level condition might need review
		// if the standard is that RulesJSON root MUST be a group object.
		// For now, assuming this wrapping is a desired flexibility.
		var singleCond RuleCondition
		if errUnmarshalCond := json.Unmarshal([]byte(groupDef.RulesJSON), &singleCond); errUnmarshalCond == nil && singleCond.AttributeID != "" {
			log.Printf("Top-level rule for group %s is a single condition. Wrapping it in a default AND group.", groupID)
			topRuleGroup = RuleGroup{
				Type:            "group",
				EntityID:        groupDef.EntityID, // Inherit group's primary entity ID
				LogicalOperator: "AND",
				Rules:           []json.RawMessage{json.RawMessage(groupDef.RulesJSON)},
			}
		} else {
			errMsg := fmt.Sprintf("invalid RulesJSON for group %s: top-level 'type' must be 'group' or a single valid condition. Got: %s", groupID, groupDef.RulesJSON)
			_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
			_ = tx.Commit()
			return nil, fmt.Errorf(errMsg)
		}
	}
	
	// Ensure topRuleGroup has an EntityID, default to groupDef.EntityID if not specified in JSON
	if topRuleGroup.EntityID == "" {
		log.Printf("Top-level rule group for group %s is missing 'entity_id', defaulting to group's primary EntityID: %s", groupID, groupDef.EntityID)
		topRuleGroup.EntityID = groupDef.EntityID
	} else if topRuleGroup.EntityID != groupDef.EntityID {
		// This is a critical mismatch. The group definition's entity_id should be the source of truth for the primary entity.
		errMsg := fmt.Sprintf("mismatch: GroupDefinition.EntityID ('%s') and RulesJSON root EntityID ('%s') for group %s. Using GroupDefinition.EntityID.", groupDef.EntityID, topRuleGroup.EntityID, groupID)
		log.Println(errMsg) // Log this, but proceed with groupDef.EntityID as the primary.
		topRuleGroup.EntityID = groupDef.EntityID 
		// Potentially, this could be an error state:
		// _ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		// _ = tx.Commit()
		// return nil, fmt.Errorf(errMsg)
	}


	if topRuleGroup.LogicalOperator == "" && len(topRuleGroup.Rules) > 1 {
		errMsg := fmt.Sprintf("invalid RulesJSON for group %s: top-level 'logical_operator' is required when multiple rules exist. Rules: %s", groupID, groupDef.RulesJSON)
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit()
		return nil, fmt.Errorf(errMsg)
	} else if topRuleGroup.LogicalOperator == "" { // Covers len(rules) <= 1
		topRuleGroup.LogicalOperator = "AND" // Default for single or no rule
	}

	attrInfoMap := make(map[string]string)             // AttributeID -> AttributeName
	entityAttrMap := make(map[string]map[string]string) // EntityID -> map[AttributeID]AttributeName
	
	attrInfoMap := make(map[string]string)                 // AttributeID -> AttributeName
	entityAttrMap := make(map[string]map[string]string)     // EntityID -> map[AttributeID]AttributeName
	relationshipIDsMap := make(map[string]bool)             // Set of RelationshipIDs

	// Extract all attribute and entity information from the rule structure
	if err := getAllAttributeIDsAndNamesRecursive(topRuleGroup, attrInfoMap, entityAttrMap, relationshipIDsMap, s.metadataClient, topRuleGroup.EntityID); err != nil {
		errMsg := fmt.Sprintf("failed to extract attribute, entity, and relationship info from rules for group %s: %w. Rules: %s", groupID, err, groupDef.RulesJSON)
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit()
		return nil, fmt.Errorf(errMsg)
	}

	attributeDefsMap := make(map[string]*AttributeDefinition)
	// Fetch all unique AttributeDefinitions
	for entityIDCtx, attrsInEntity := range entityAttrMap {
		for attrID := range attrsInEntity {
			if _, ok := attributeDefsMap[attrID]; !ok { // Fetch only if not already fetched
				attrDef, errA := s.metadataClient.GetAttributeDefinition(entityIDCtx, attrID)
				if errA != nil {
					errMsg := fmt.Sprintf("failed to get attribute definition for ID %s (EntityID %s) for group %s: %w", attrID, entityIDCtx, groupID, errA)
					_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
					_ = tx.Commit()
					return nil, fmt.Errorf(errMsg)
				}
				attributeDefsMap[attrID] = attrDef
			}
		}
	}
	
	relationshipDefsMap := make(map[string]*EntityRelationshipDefinition)
	// Fetch all unique EntityRelationshipDefinitions
	for relID := range relationshipIDsMap {
		relDef, errR := s.metadataClient.GetEntityRelationship(relID)
		if errR != nil {
			errMsg := fmt.Sprintf("failed to get entity relationship definition for ID %s for group %s: %w", relID, groupID, errR)
			_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
			_ = tx.Commit()
			return nil, fmt.Errorf(errMsg)
		}
		relationshipDefsMap[relID] = relDef
	}


	// SQL Construction
	tableAliasCounter := 0
	generateAlias := func() string {
		tableAliasCounter++
		return fmt.Sprintf("pe%d", tableAliasCounter)
	}
	primaryTableAlias := generateAlias() // e.g., "pe1"

	var queryParams []interface{} // Renamed from whereClauseParams for clarity
	paramCounter := 1            // Renamed from whereClauseParamCounter

	// The primary entity for the query is topRuleGroup.EntityID
	// Pass s.metadataClient in case buildWhereClauseRecursive needs to fetch relationships not caught by getAll... (ideally shouldn't happen)
	whereClause, err := buildWhereClauseRecursive(topRuleGroup, attributeDefsMap, relationshipDefsMap, &queryParams, &paramCounter, primaryTableAlias, generateAlias, topRuleGroup.EntityID, s.metadataClient)
	if err != nil {
		errMsg := fmt.Sprintf("failed to build WHERE clause for group %s: %w. Rules: %s", groupID, err, groupDef.RulesJSON)
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit()
		return nil, fmt.Errorf(errMsg)
	}

	var finalQuery strings.Builder
	// Primary query selects from the aliased primary entity table
	finalQuery.WriteString(fmt.Sprintf("SELECT %s.id FROM processed_entities %s", primaryTableAlias, primaryTableAlias))
	
	// Start building the WHERE clause for the main query
	mainWhereConditions := []string{}
	finalParams := []interface{}{} // Parameters for the final composed query

	// 1. Add primary entity ID filter for the main table
	mainWhereConditions = append(mainWhereConditions, fmt.Sprintf("%s.entity_definition_id = $%d", primaryTableAlias, len(finalParams)+1))
	finalParams = append(finalParams, topRuleGroup.EntityID)
	
	// 2. Add the generated whereClause (which contains conditions and subqueries)
	if whereClause != "" {
		// The parameters for 'whereClause' are already in 'queryParams'.
		// We need to adjust their placeholder numbers ($N) to follow the primary entity_id param.
		// Or, more simply, pass finalParams directly to buildWhereClauseRecursive and let it append.
		// For now, let's re-evaluate the param strategy for buildWhereClauseRecursive.
		// Let's assume buildWhereClauseRecursive populates queryParams starting from $1.
		// We will prepend primary entity_id to queryParams before execution.

		// Simpler approach: buildWhereClauseRecursive uses its own paramCounter starting from 1.
		// The main entity_definition_id filter is added *after* the recursive call returns.
		// So, the main query params will be [topRuleGroup.EntityID, ...queryParams].
		// The $N placeholders in whereClause are already correct relative to queryParams.
		// The placeholder for topRuleGroup.EntityID needs to be managed.

		// Revised strategy:
		// queryParams will hold all parameters.
		// buildWhereClauseRecursive will use paramCounter to add its params.
		// The primary entity_id filter will also use paramCounter.

		// Let's reset and build finalParams and mainWhereConditions carefully.
		finalParams = []interface{}{} // Reset
		paramCounter = 1 // Reset for the whole query construction
		mainWhereConditions = []string{}
		
		// Add primary entity_id filter first
		mainWhereConditions = append(mainWhereConditions, fmt.Sprintf("%s.entity_definition_id = $%d", primaryTableAlias, paramCounter))
		finalParams = append(finalParams, topRuleGroup.EntityID)
		paramCounter++

		// Now, build the complex WHERE clause, passing the *current* finalParams and paramCounter
		// This means buildWhereClauseRecursive will append to finalParams and increment paramCounter.
		// This is a change from its previous independent parameter management.
		// Let's revert buildWhereClauseRecursive to manage its own params and paramCounter starting from 1.
		// Then we combine them. This is cleaner.

		// whereClauseParams and whereClauseParamCounter are used by buildWhereClauseRecursive.
		// Let's rename them back for clarity with the previous step's logic for buildWhereClauseRecursive.
		// whereClauseParams was already populated by buildWhereClauseRecursive.
		// The parameters for whereClause are already in queryParams.
		
		if whereClause != "" {
			mainWhereConditions = append(mainWhereConditions, fmt.Sprintf("(%s)", whereClause))
		}
	}
	
	if len(mainWhereConditions) > 0 {
		finalQuery.WriteString(" WHERE ")
		finalQuery.WriteString(strings.Join(mainWhereConditions, " AND "))
	}

	// Combine parameters: primary entity ID first, then those from the recursive WHERE clause.
	// This assumes buildWhereClauseRecursive populates `queryParams` with placeholders starting $1, $2...
	// And the main `entity_definition_id` filter also needs a placeholder.
	
	// Corrected parameter combination:
	// The main query's entity_id filter is $1.
	// Parameters for whereClause (from queryParams) should be shifted.
	// Example: If whereClause was "col = $1", it becomes "col = $2"
	// This is complex. A simpler way:
	// finalParams starts with topRuleGroup.EntityID. buildWhereClauseRecursive appends to it.
	// Let's modify buildWhereClauseRecursive to append to the passed 'params' and update 'paramCounter'.
	// This was the strategy in the previous diff for buildWhereClauseRecursive. I'll stick to that.

	// Re-instating the logic from previous diff for params:
	// finalParams will be built by adding topRuleGroup.EntityID, then whereClauseParams.
	// The $N placeholders in whereClause must be adjusted.
	// This is getting complicated. Let's simplify:
	// paramCounter is global for the whole query.
	// buildWhereClauseRecursive appends to `finalParams` and increments `paramCounter`.

	// Resetting finalParams and paramCounter for the entire query generation.
	finalParams = []interface{}{}
	paramCounter = 1 

	finalQuery = strings.Builder{} // Reset builder
	finalQuery.WriteString(fmt.Sprintf("SELECT %s.id FROM processed_entities %s", primaryTableAlias, primaryTableAlias))
	
	var actualWhereConditions []string
	// Primary entity filter
	actualWhereConditions = append(actualWhereConditions, fmt.Sprintf("%s.entity_definition_id = $%d", primaryTableAlias, paramCounter))
	finalParams = append(finalParams, topRuleGroup.EntityID)
	paramCounter++

	// Build the rest of the WHERE clause.
	// buildWhereClauseRecursive will now append to finalParams and use/update paramCounter.
	recursiveWhereClause, err := buildWhereClauseRecursive(topRuleGroup, attributeDefsMap, relationshipDefsMap, &finalParams, &paramCounter, primaryTableAlias, generateAlias, topRuleGroup.EntityID, s.metadataClient)
	if err != nil {
		errMsg := fmt.Sprintf("failed to build WHERE clause for group %s: %w. Rules: %s", groupID, err, groupDef.RulesJSON)
		_ = s.upsertGroupCalculationLog(tx, groupDef.ID, groupDef.EntityID, "FAILED", 0, sql.NullString{String: errMsg, Valid: true})
		_ = tx.Commit()
		return nil, fmt.Errorf(errMsg)
	}
	if recursiveWhereClause != "" {
		actualWhereConditions = append(actualWhereConditions, fmt.Sprintf("(%s)", recursiveWhereClause))
	}

	if len(actualWhereConditions) > 0 {
		finalQuery.WriteString(" WHERE " + strings.Join(actualWhereConditions, " AND "))
	}

	sqlQueryStr := finalQuery.String()
	log.Printf("Executing member query for group %s: %s with params %v", groupID, sqlQueryStr, finalParams)

	// Use tx for the query, as the entire operation should be atomic.
	rows, err := tx.Query(sqlQueryStr, finalParams...)
	if err != nil {
		errMsg := fmt.Sprintf("failed to execute member query for group %s: %v. Query: %s, Params: %v", groupID, err, sqlQueryStr, finalParams)
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
