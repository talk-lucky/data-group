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
