package metadata

import "time"

// EntityDefinition represents the structure for a metadata entity.
type EntityDefinition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// --- Bulk Operation Structs for Entities ---

// EntityCreateData holds the data for creating a single entity within a bulk request.
type EntityCreateData struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
}

// BulkCreateEntitiesRequest defines the structure for a bulk entity creation request.
type BulkCreateEntitiesRequest struct {
	// Entities is a list of entities to be created.
	Entities []EntityCreateData `json:"entities" binding:"required,dive"`
}

// EntityUpdateData holds the data for updating a single entity within a bulk request.
// ID is required to identify the entity. Other fields are optional for partial updates.
type EntityUpdateData struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name,omitempty"`        // omitempty means if not provided, it won't be included in JSON, allowing for partial updates
	Description string `json:"description,omitempty"` // omitempty allows for partial updates
}

// BulkUpdateEntitiesRequest defines the structure for a bulk entity update request.
type BulkUpdateEntitiesRequest struct {
	// Entities is a list of entities to be updated.
	Entities []EntityUpdateData `json:"entities" binding:"required,dive"`
}

// BulkDeleteEntitiesRequest defines the structure for a bulk entity deletion request.
type BulkDeleteEntitiesRequest struct {
	// EntityIDs is a list of entity IDs to be deleted.
	EntityIDs []string `json:"entity_ids" binding:"required,dive"`
}

// BulkOperationResultItem represents the result of a single operation within a bulk request.
type BulkOperationResultItem struct {
	// ID is the identifier of the item that was processed.
	// It might be empty if the operation (e.g., creation) failed before an ID was assigned or known.
	ID string `json:"id,omitempty"`
	// Success indicates whether the operation on this item was successful.
	Success bool `json:"success"`
	// Error provides a message if the operation on this item failed.
	Error string `json:"error,omitempty"`
	// Entity holds the created/updated entity if the operation was successful and applicable.
	Entity *EntityDefinition `json:"entity,omitempty"`
}

// BulkOperationResponse defines the structure for the response of a bulk operation.
// It contains a list of results for each item processed in the bulk request.
type BulkOperationResponse struct {
	// Results is a list detailing the outcome for each individual item in the bulk operation.
	Results []BulkOperationResultItem `json:"results"`
}

// APIError represents a standard error response format for the API.
type APIError struct {
	// Code is the HTTP status code or a custom error code.
	Code int `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
}

// ListResponse represents a generic structure for API responses that return a list of items.
// It includes the data itself and a total count for pagination purposes.
type ListResponse struct {
	// Data holds the list of items. It's an interface{} to be flexible for various data types.
	Data interface{} `json:"data"`
	// Total is the total number of items available, which might be more than the items returned in a single response.
	Total int64 `json:"total"`
}


// RelationshipType defines the nature of the connection between two entities.
type RelationshipType string

const (
	OneToOne  RelationshipType = "ONE_TO_ONE"
	OneToMany RelationshipType = "ONE_TO_MANY"
	ManyToOne RelationshipType = "MANY_TO_ONE"
	// ManyToMany might be complex to implement directly without an intermediary table
	// and might be out of scope for initial implementation.
)

// EntityRelationshipDefinition defines how two entities (and specific attributes within them) are related.
type EntityRelationshipDefinition struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"` // User-friendly name for the relationship, e.g., "UserOrders"
	Description       string           `json:"description,omitempty"`
	SourceEntityID    string           `json:"source_entity_id"`    // ID of the source EntityDefinition
	SourceAttributeID string           `json:"source_attribute_id"` // ID of the source AttributeDefinition (e.g., foreign key)
	TargetEntityID    string           `json:"target_entity_id"`    // ID of the target EntityDefinition
	TargetAttributeID string           `json:"target_attribute_id"` // ID of the target AttributeDefinition (e.g., primary key)
	RelationshipType  RelationshipType `json:"relationship_type"`   // e.g., "ONE_TO_ONE", "ONE_TO_MANY"
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// ScheduleDefinition defines the structure for a scheduled task.
type ScheduleDefinition struct {
	ID                string    `json:"id"`
	Name              string    `json:"name" binding:"required"`
	Description       string    `json:"description,omitempty"`
	CronExpression    string    `json:"cron_expression" binding:"required"`
	TaskType          string    `json:"task_type" binding:"required"` // e.g., "ingest_data_source", "calculate_group", "trigger_workflow"
	TaskParameters    string    `json:"task_parameters" binding:"required"` // JSON string, e.g., {"source_id": "uuid"}
	IsEnabled         bool      `json:"is_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WorkflowDefinition defines the structure for a workflow.
type WorkflowDefinition struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	TriggerType         string    `json:"trigger_type"` // e.g., "on_group_update", "manual"
	TriggerConfig       string    `json:"trigger_config"` // e.g., {"group_id": "uuid"} as JSON string
	ActionSequenceJSON  string    `json:"action_sequence_json"` // JSON string for action sequence
	IsEnabled           bool      `json:"is_enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ActionTemplate defines the structure for an action template.
type ActionTemplate struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	ActionType      string    `json:"action_type"` // e.g., "webhook", "email"
	TemplateContent string    `json:"template_content"` // JSON string for template
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AttributeDefinition represents the structure for a metadata attribute.
type AttributeDefinition struct {
	ID           string    `json:"id"`
	EntityID     string    `json:"entity_id"` // Foreign key to EntityDefinition
	Name         string    `json:"name"`
	DataType     string    `json:"data_type"`
	Description  string    `json:"description"`
	IsFilterable bool      `json:"is_filterable"`
	IsPii        bool      `json:"is_pii"`
	IsIndexed    bool      `json:"is_indexed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DataSourceConfig represents the configuration for a data source.
type DataSourceConfig struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`               // e.g., "PostgreSQL", "MySQL", "CSV", "API"
	ConnectionDetails string    `json:"connection_details"` // JSON string for connection parameters
	EntityID          string    `json:"entity_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DataSourceFieldMapping represents the mapping between a data source field and a metadata attribute.
type DataSourceFieldMapping struct {
	ID                 string    `json:"id"`
	SourceID           string    `json:"source_id"` // Foreign key to DataSourceConfig
	SourceFieldName    string    `json:"source_field_name"`
	EntityID           string    `json:"entity_id"`                     // Foreign key to EntityDefinition (helps UI to pick attributes)
	AttributeID        string    `json:"attribute_id"`                  // Foreign key to AttributeDefinition
	TransformationRule string    `json:"transformation_rule,omitempty"` // Optional, e.g., "lowercase", "trim"
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// GroupDefinition defines the structure for a group of entities based on rules.
type GroupDefinition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	EntityID    string    `json:"entity_id"` // Foreign key to EntityDefinition
	RulesJSON   string    `json:"rules_json"` // JSON string to store grouping rules
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
