package models

import (
	"time"

	"github.com/google/uuid"
)

// ValidDataTypes defines the allowed data types for attributes.
var ValidDataTypes = map[string]bool{
	"STRING":   true,
	"TEXT":     true, // For longer text
	"INTEGER":  true,
	"FLOAT":    true,
	"BOOLEAN":  true,
	"DATETIME": true,
	// "RELATIONSHIP": true, // Future consideration
}

// EntityDefinition represents the structure for an entity definition.
// @Description EntityDefinition represents the structure for an entity definition.
type EntityDefinition struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name        string    `json:"name" binding:"required,min=1,max=255" gorm:"type:varchar(255);not null;unique"`
	Description string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Attributes  []AttributeDefinition `json:"attributes,omitempty" gorm:"foreignKey:EntityID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// AttributeDefinition represents the structure for an attribute definition.
// @Description AttributeDefinition represents the structure for an attribute definition.
type AttributeDefinition struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	EntityID      uuid.UUID `json:"entity_id" binding:"required" gorm:"type:uuid;not null;uniqueIndex:idx_entity_attr_name"` // Part of composite unique index
	Name          string    `json:"name" binding:"required,min=1,max=255" gorm:"type:varchar(255);not null;uniqueIndex:idx_entity_attr_name"` // Part of composite unique index
	DataType      string    `json:"data_type" binding:"required,oneof=STRING TEXT INTEGER FLOAT BOOLEAN DATETIME" gorm:"type:varchar(50);not null"`
	Description   string    `json:"description,omitempty" gorm:"type:text"`
	IsFilterable  bool      `json:"is_filterable" gorm:"default:false"`
	IsPII         bool      `json:"is_pii" gorm:"default:false"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	// Entity        EntityDefinition `json:"-" gorm:"foreignKey:EntityID"` // Belongs to Entity (optional, for eager loading if needed)
}

// CreateEntityRequest defines the request payload for creating an entity.
type CreateEntityRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description,omitempty" binding:"max=1000"`
}

// UpdateEntityRequest defines the request payload for updating an entity.
type UpdateEntityRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// CreateAttributeRequest defines the request payload for creating an attribute.
type CreateAttributeRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=255"`
	DataType      string `json:"data_type" binding:"required,oneof=STRING TEXT INTEGER FLOAT BOOLEAN DATETIME"`
	Description   string `json:"description,omitempty" binding:"max=1000"`
	IsFilterable  *bool  `json:"is_filterable,omitempty"` // Pointer to distinguish between false and not provided
	IsPII         *bool  `json:"is_pii,omitempty"`        // Pointer to distinguish between false and not provided
}

// UpdateAttributeRequest defines the request payload for updating an attribute.
type UpdateAttributeRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	DataType      *string `json:"data_type,omitempty" binding:"omitempty,oneof=STRING TEXT INTEGER FLOAT BOOLEAN DATETIME"`
	Description   *string `json:"description,omitempty" binding:"omitempty,max=1000"`
	IsFilterable  *bool   `json:"is_filterable,omitempty"`
	IsPII         *bool   `json:"is_pii,omitempty"`
}
