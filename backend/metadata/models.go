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
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Name        string    `json:"name"`
	DataType    string    `json:"data_type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
