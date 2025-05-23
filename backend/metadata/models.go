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
