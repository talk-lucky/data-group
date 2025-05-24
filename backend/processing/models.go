package processing

// ProcessDataRequest defines the structure for the data processing request.
type ProcessDataRequest struct {
	SourceID       string                   `json:"source_id"`
	EntityTypeName string                   `json:"entity_type_name"`
	RawData        []map[string]interface{} `json:"raw_data"`
}

// DataSourceFieldMapping mirrors the structure in the metadata service.
type DataSourceFieldMapping struct {
	ID                 string `json:"id"`
	SourceID           string `json:"source_id"`
	SourceFieldName    string `json:"source_field_name"`
	EntityID           string `json:"entity_id"`
	AttributeID        string `json:"attribute_id"`
	TransformationRule string `json:"transformation_rule,omitempty"`
}

// AttributeDefinition mirrors the structure in the metadata service.
type AttributeDefinition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// DataType might be used later for type casting or validation
	DataType string `json:"data_type"`
}

// EntityDefinition mirrors the structure in the metadata service
type EntityDefinition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DataSourceConfig mirrors the structure in the metadata service for unmarshalling.
// Only include fields relevant to ingestion.
type DataSourceConfig struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	ConnectionDetails string `json:"connection_details"` // JSON string
	EntityID          string `json:"entity_id,omitempty"`
}
