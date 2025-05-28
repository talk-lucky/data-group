package metadata

import "time"

// EntityDefinition represents the structure for a metadata entity.
// It defines a type of object that can be tracked or managed, e.g., "User", "Product", "Order".
type EntityDefinition struct {
	// ID is the unique identifier for the entity definition (e.g., a UUID).
	ID string `json:"id"`
	// Name is the user-defined, human-readable name for the entity type (e.g., "Customer", "Server").
	// This field should be unique across all entity definitions.
	Name string `json:"name"`
	// Description provides an optional, more detailed explanation of the entity type's purpose or characteristics.
	Description string `json:"description,omitempty"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this entity definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this entity definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this entity definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
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
	// ManyToMany represents a many-to-many relationship between two entities.
	// This type typically implies an intermediary (join) table in a relational database context,
	// though the direct definition here focuses on the conceptual link.
	ManyToMany RelationshipType = "MANY_TO_MANY"
)

// EntityRelationshipDefinition defines how two entities (and specific attributes within them) are related.
// This describes the type of link (e.g., User 'has many' Orders) and the specific attributes that form this link.
type EntityRelationshipDefinition struct {
	// ID is the unique identifier for the entity relationship definition (e.g., a UUID).
	ID string `json:"id"`
	// Name provides a user-friendly name for the relationship (e.g., "UserOrders", "ProductCategory").
	// This name should be unique within the context of a source entity, or globally if preferred.
	Name string `json:"name"`
	// Description offers an optional, more detailed explanation of the relationship's purpose or nature.
	Description string `json:"description,omitempty"`
	// SourceEntityID is the ID of the EntityDefinition that is the source of the relationship.
	SourceEntityID string `json:"source_entity_id"`
	// SourceAttributeID is the ID of the AttributeDefinition (belonging to the SourceEntityID)
	// that acts as the source key for the relationship (e.g., a primary key like "User.ID").
	SourceAttributeID string `json:"source_attribute_id"`
	// TargetEntityID is the ID of the EntityDefinition that is the target of the relationship.
	TargetEntityID string `json:"target_entity_id"`
	// TargetAttributeID is the ID of the AttributeDefinition (belonging to the TargetEntityID)
	// that acts as the target key for the relationship (e.g., a foreign key like "Order.UserID").
	TargetAttributeID string `json:"target_attribute_id"`
	// RelationshipType specifies the cardinality and nature of the link (e.g., "ONE_TO_ONE", "ONE_TO_MANY", "MANY_TO_MANY").
	RelationshipType RelationshipType `json:"relationship_type"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this relationship definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this relationship definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this relationship definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// ScheduleDefinition defines the structure for a scheduled task or job within the system.
// These tasks could range from data ingestion to triggering workflows or performing calculations.
type ScheduleDefinition struct {
	// ID is the unique identifier for the schedule definition (e.g., a UUID).
	ID string `json:"id"`
	// Name is a user-defined, human-readable name for the schedule (e.g., "Daily Sales Data Ingestion", "Hourly Report Generation").
	// This field is typically required and may need to be unique.
	Name string `json:"name" binding:"required"`
	// Description provides an optional, more detailed explanation of what the scheduled task does.
	Description string `json:"description,omitempty"`
	// CronExpression is a standard cron string (e.g., "0 0 * * *") defining the schedule's frequency and timing. Required.
	CronExpression string `json:"cron_expression" binding:"required"`
	// TaskType specifies the kind of task to be executed (e.g., "ingest_data_source", "calculate_group", "trigger_workflow"). Required.
	// This helps the system route the execution to the appropriate handler.
	TaskType string `json:"task_type" binding:"required"`
	// TaskParameters is a JSON string containing specific parameters required by the TaskType.
	// Example: `{"source_id": "uuid-of-data-source"}` for an "ingest_data_source" task. Required.
	TaskParameters string `json:"task_parameters" binding:"required"`
	// IsEnabled indicates whether the schedule is currently active and should run at its defined times.
	IsEnabled bool `json:"is_enabled"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this schedule definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this schedule definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this schedule definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkflowDefinition defines the structure for an automated workflow.
// Workflows consist of a trigger and a sequence of actions to be performed.
type WorkflowDefinition struct {
	// ID is the unique identifier for the workflow definition (e.g., a UUID).
	ID string `json:"id"`
	// Name is a user-defined, human-readable name for the workflow (e.g., "New User Onboarding", "Order Fulfillment Process").
	Name string `json:"name"`
	// Description provides an optional, more detailed explanation of the workflow's purpose and steps.
	Description string `json:"description,omitempty"`
	// TriggerType specifies the event or condition that initiates the workflow (e.g., "on_entity_create", "on_group_update", "manual", "scheduled").
	TriggerType string `json:"trigger_type"`
	// TriggerConfig is a JSON string containing parameters specific to the TriggerType.
	// Example: `{"entity_id": "user-entity-uuid", "conditions": {"field": "status", "value": "active"}}` for an "on_entity_update" trigger.
	TriggerConfig string `json:"trigger_config"`
	// ActionSequenceJSON is a JSON string defining the sequence of actions to be executed when the workflow is triggered.
	// This would detail action types, parameters, and potentially control flow (e.g., conditional steps, parallel execution).
	ActionSequenceJSON string `json:"action_sequence_json"`
	// IsEnabled indicates whether the workflow is currently active and can be triggered.
	IsEnabled bool `json:"is_enabled"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this workflow definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this workflow definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this workflow definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// ActionTemplate defines a reusable template for actions that can be part of a workflow or executed independently.
type ActionTemplate struct {
	// ID is the unique identifier for the action template (e.g., a UUID).
	ID string `json:"id"`
	// Name is a user-defined, human-readable name for the action template (e.g., "Send Welcome Email", "Notify Slack Channel").
	Name string `json:"name"`
	// Description provides an optional, more detailed explanation of what the action does.
	Description string `json:"description,omitempty"`
	// ActionType specifies the kind of action this template represents (e.g., "webhook", "email", "update_entity").
	// This helps the system route the execution to the appropriate handler.
	ActionType string `json:"action_type"`
	// TemplateContent is a JSON string containing the configuration or content for the action.
	// Example: For an "email" action, this might include subject, body template, and recipient placeholders.
	// For a "webhook", it might include URL, headers, and payload template.
	TemplateContent string `json:"template_content"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this action template.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this action template was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this action template was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Attribute Data Type Constants ---

// BaseDataTypeName represents the fundamental kind of data an attribute holds.
type BaseDataTypeName string

const (
	// BaseTypeString indicates a sequence of characters.
	BaseTypeString BaseDataTypeName = "string"
	// BaseTypeInteger indicates whole numbers.
	BaseTypeInteger BaseDataTypeName = "integer"
	// BaseTypeFloat indicates floating-point numbers.
	BaseTypeFloat BaseDataTypeName = "float"
	// BaseTypeBoolean indicates true/false values.
	BaseTypeBoolean BaseDataTypeName = "boolean"
	// BaseTypeDateTime indicates a specific point in time.
	BaseTypeDateTime BaseDataTypeName = "datetime"
	// BaseTypeDate indicates a specific date.
	BaseTypeDate BaseDataTypeName = "date"
	// BaseTypeTime indicates a specific time of day.
	BaseTypeTime BaseDataTypeName = "time"
	// BaseTypeArray indicates an ordered list of items of a specified type.
	// Details about the item type are in DataTypeDetails.
	BaseTypeArray BaseDataTypeName = "array"
	// BaseTypeObject indicates a collection of key-value pairs, where values can be of different types.
	// Details about the object's schema are in DataTypeDetails.
	BaseTypeObject BaseDataTypeName = "object"
	// BaseTypeEnum indicates a value that must be one of a predefined set of allowed strings.
	// The allowed values are specified in DataTypeDetails.
	BaseTypeEnum BaseDataTypeName = "enum"
	// BaseTypeReference indicates a link to another entity instance (e.g., a foreign key).
	// Details about the referenced entity are in DataTypeDetails.
	BaseTypeReference BaseDataTypeName = "reference"
	// BaseTypeJSON indicates a field storing arbitrary JSON data.
	BaseTypeJSON BaseDataTypeName = "json"
)

// AttributeDefinition represents the structure for a metadata attribute.
// An attribute is a characteristic or property of an entity type, e.g., "User.email", "Product.price".
type AttributeDefinition struct {
	// ID is the unique identifier for the attribute definition (e.g., a UUID).
	ID string `json:"id"`
	// EntityID is the foreign key referencing the EntityDefinition to which this attribute belongs.
	EntityID string `json:"entity_id"`
	// Name is the user-defined, human-readable name for the attribute (e.g., "EmailAddress", "UnitPrice").
	// This name should be unique within its parent entity definition.
	Name string `json:"name"`
	// DataTypeName stores the base type of the attribute, e.g., "string", "integer", "array", "enum".
	// For complex types like "array", "object", or "enum", DataTypeDetails will provide more specifics.
	DataTypeName BaseDataTypeName `json:"data_type_name" gorm:"column:data_type_name"`
	// DataTypeDetails stores specifics for complex types. Examples:
	// For "enum": `{"values": ["value1", "value2"]}`
	// For "array": `{"item_type_name": "string"}` or `{"item_type_name": "object", "item_type_details": {"schema": {"nested_field": "integer"}}}`
	// For "object": `{"schema": {"field1": "string", "field2": "integer"}}` (Simplified schema representation)
	// For "reference": `{"referenced_entity_id": "entity-uuid-xxxxx"}`
	DataTypeDetails map[string]interface{} `json:"data_type_details,omitempty" gorm:"type:jsonb"`
	// Description provides an optional, more detailed explanation of the attribute's meaning or usage.
	Description string `json:"description,omitempty"`
	// IsFilterable indicates whether this attribute can be used in query filters.
	IsFilterable bool `json:"is_filterable"`
	// IsPii indicates whether this attribute contains Personally Identifiable Information (PII).
	// This can be used for data governance and privacy considerations.
	IsPii bool `json:"is_pii"`
	// IsIndexed indicates whether this attribute should be indexed in the underlying data store for faster lookups.
	IsIndexed bool `json:"is_indexed"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this attribute definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this attribute definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this attribute definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// DataSourceConfig represents the configuration for an external data source.
// This could be a database, a CSV file, an API endpoint, etc.
type DataSourceConfig struct {
	// ID is the unique identifier for the data source configuration (e.g., a UUID).
	ID string `json:"id"`
	// Name is a user-defined, human-readable name for the data source (e.g., "Production PostgreSQL DB", "Customer Shopify API").
	Name string `json:"name"`
	// Type specifies the kind of data source (e.g., "PostgreSQL", "MySQL", "CSV", "API").
	// This helps determine how to connect and interact with the source.
	Type string `json:"type"`
	// ConnectionDetails is a JSON string containing the parameters needed to connect to the data source
	// (e.g., host, port, username, password, API key, file path). Structure varies by Type.
	ConnectionDetails string `json:"connection_details"`
	// EntityID is an optional foreign key to an EntityDefinition.
	// This can be used if the data source is primarily associated with one main entity type,
	// though mappings might define connections to other entities as well.
	EntityID string `json:"entity_id,omitempty"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this data source configuration.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this data source configuration was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this data source configuration was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// DataSourceFieldMapping represents the mapping between a field in an external data source
// and a specific attribute of an entity definition in the metadata system.
type DataSourceFieldMapping struct {
	// ID is the unique identifier for the field mapping (e.g., a UUID).
	ID string `json:"id"`
	// SourceID is the foreign key referencing the DataSourceConfig to which this mapping belongs.
	SourceID string `json:"source_id"`
	// SourceFieldName is the name of the field as it appears in the external data source (e.g., column name, API response key).
	SourceFieldName string `json:"source_field_name"`
	// EntityID is the foreign key referencing the EntityDefinition that the source field maps to.
	// This helps in UI selections by scoping attributes to the relevant entity.
	EntityID string `json:"entity_id"`
	// AttributeID is the foreign key referencing the AttributeDefinition that the source field maps to.
	AttributeID string `json:"attribute_id"`
	// TransformationRule is an optional rule or expression defining how to transform the source field's data
	// before it's mapped to the attribute (e.g., "lowercase", "trim", "multiply_by_100").
	TransformationRule string `json:"transformation_rule,omitempty"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this field mapping.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this field mapping was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this field mapping was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// GroupDefinition defines the structure for a group of entities based on a set of rules or criteria.
// Groups can be used for various purposes, such as segmentation, policy application, or triggering workflows.
type GroupDefinition struct {
	// ID is the unique identifier for the group definition (e.g., a UUID).
	ID string `json:"id"`
	// Name is a user-defined, human-readable name for the group (e.g., "High Value Customers", "Test Servers").
	Name string `json:"name"`
	// EntityID is the foreign key referencing the EntityDefinition to which the group's rules apply.
	// This specifies the type of entities that can be part of this group.
	EntityID string `json:"entity_id"`
	// RulesJSON is a JSON string containing the rules or criteria that define membership in this group.
	// The structure of these rules would be specific to the rule engine implementation.
	// Example: `{"condition": "AND", "rules": [{"field": "order_count", "operator": ">", "value": 10}]}`
	RulesJSON string `json:"rules_json"`
	// Description provides an optional, more detailed explanation of the group's purpose or membership criteria.
	Description string `json:"description,omitempty"`
	// Metadata allows for storing arbitrary key-value pairs for user-defined extensions,
	// custom attributes, or annotations related to this group definition.
	Metadata map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	// CreatedAt records the timestamp (UTC) when this group definition was first created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt records the timestamp (UTC) when this group definition was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}
