package main

import "time"

// DEPRECATED: This struct is part of an older rule definition system.
// Use structs in grouping/service.go (RuleCondition) for new rule processing.
// RuleCondition defines a single condition within a rule set.
// The AttributeID here refers to the ID of an AttributeDefinition.
type RuleCondition struct {
	AttributeID string      `json:"attributeId"` // Changed from attribute_id to match frontend
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value,omitempty"` // Value might be omitted for unary operators like 'is_null'
}

// DEPRECATED: This struct is part of an older rule definition system.
// Use structs in grouping/service.go (RuleGroup) for new rule processing.
// RuleSet defines a collection of conditions and how they are logically combined.
type RuleSet struct {
	Conditions      []RuleCondition `json:"conditions"`
	LogicalOperator string          `json:"logical_operator"` // e.g., "AND", "OR"
}

// --- Structs for Metadata Service Responses ---
// These are minimal versions needed by the grouping service.

// GroupDefinition mirrors the structure from the metadata service.
type GroupDefinition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	EntityID    string    `json:"entity_id"`
	RulesJSON   string    `json:"rules_json"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EntityDefinition mirrors the structure from the metadata service.
type EntityDefinition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AttributeDefinition mirrors the structure from the metadata service.
type AttributeDefinition struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Name        string    `json:"name"`
	DataType    string    `json:"data_type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Fields like IsFilterable, IsPii, IsIndexed are not strictly needed for grouping logic itself
	// but DataType and Name are crucial.
}
