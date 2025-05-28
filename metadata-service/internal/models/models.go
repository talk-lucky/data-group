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

// ValidRelationshipTypes defines the allowed types for entity relationships.
var ValidRelationshipTypes = map[string]bool{
	"ONE_TO_ONE":   true,
	"ONE_TO_MANY":  true,
	"MANY_TO_MANY": true,
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

// EntityRelationshipDefinition represents the structure for an entity relationship definition.
// @Description EntityRelationshipDefinition represents the structure for an entity relationship definition.
type EntityRelationshipDefinition struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	Name             string    `json:"name,omitempty" binding:"omitempty,max=255" gorm:"type:varchar(255)"`
	SourceEntityID   uuid.UUID `json:"source_entity_id" binding:"required" gorm:"type:uuid;not null"`
	TargetEntityID   uuid.UUID `json:"target_entity_id" binding:"required" gorm:"type:uuid;not null"`
	RelationshipType string    `json:"relationship_type" binding:"required,oneof=ONE_TO_ONE ONE_TO_MANY MANY_TO_MANY" gorm:"type:varchar(50);not null"`
	Description      string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	// SourceEntity     EntityDefinition `json:"-" gorm:"foreignKey:SourceEntityID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"` // RESTRICT to prevent deletion if relationships exist
	// TargetEntity     EntityDefinition `json:"-" gorm:"foreignKey:TargetEntityID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// CreateEntityRelationshipRequest defines the request payload for creating an entity relationship.
type CreateEntityRelationshipRequest struct {
	Name             string `json:"name,omitempty" binding:"omitempty,max=255"`
	SourceEntityID   string `json:"source_entity_id" binding:"required,uuid"`
	TargetEntityID   string `json:"target_entity_id" binding:"required,uuid"`
	RelationshipType string `json:"relationship_type" binding:"required,oneof=ONE_TO_ONE ONE_TO_MANY MANY_TO_MANY"`
	Description      string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateEntityRelationshipRequest defines the request payload for updating an entity relationship.
type UpdateEntityRelationshipRequest struct {
	Name             *string `json:"name,omitempty" binding:"omitempty,max=255"`
	RelationshipType *string `json:"relationship_type,omitempty" binding:"omitempty,oneof=ONE_TO_ONE ONE_TO_MANY MANY_TO_MANY"`
	Description      *string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// PaginatedResponse is a standardized envelope for responses that return a list of items with pagination.
// @Description PaginatedResponse provides a standard structure for APIs returning lists of data, including the data itself and pagination details.
type PaginatedResponse struct {
	Data   interface{} `json:"data"`             // Slice of the actual data items for the current page
	Total  int64       `json:"total"`            // Total number of records available for the query
	Limit  int         `json:"limit"`            // The number of items requested per page
	Offset int         `json:"offset"`           // The starting offset of the returned items
}
