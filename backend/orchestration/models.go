package main

import "time"

// TaskMessage is the payload published to NATS for an Action Executor.
type TaskMessage struct {
	TaskID           string                 `json:"task_id"`           // Unique ID for this specific task
	WorkflowID       string                 `json:"workflow_id"`       // ID of the workflow definition
	ActionTemplateID string                 `json:"action_template_id"`  // ID of the action template used
	ActionType       string                 `json:"action_type"`       // e.g., "webhook", "email"
	TemplateContent  string                 `json:"template_content"`  // Content of the ActionTemplate (e.g., webhook URL template, email body template)
	ActionParams     map[string]interface{} `json:"action_params"`     // User-defined params for this action step from WorkflowDefinition.ActionSequenceJSON
	EntityInstanceID string                 `json:"entity_instance_id"`// The ID of the entity from the group (UUID from processed_entities table)
	// Note: The Action Executor service will be responsible for fetching the full EntityInstance data
	// from the processed_entities table if needed, using the EntityInstanceID.
	// Alternatively, a small subset of EntityInstance data could be included here if commonly needed and small.
}

// --- Structs for Metadata Service Responses (minimal versions) ---

// WorkflowDefinition mirrors the structure from the metadata service.
type WorkflowDefinition struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	TriggerType        string `json:"trigger_type"`        // e.g., "on_group_update", "manual"
	TriggerConfig      string `json:"trigger_config"`      // e.g., {"group_id": "uuid"} as JSON string
	ActionSequenceJSON string `json:"action_sequence_json"`// JSON string for action sequence: [{"action_template_id": "id1", "parameters_json": "{}"}, ...]
	IsEnabled          bool   `json:"is_enabled"`
}

// ActionTemplate mirrors the structure from the metadata service.
type ActionTemplate struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ActionType      string `json:"action_type"`      // e.g., "webhook", "email"
	TemplateContent string `json:"template_content"` // JSON string for template
}

// GroupDefinition (not directly used by Orchestration Service in this simplified version,
// as group members are fetched from Grouping Service, but good for context).
// type GroupDefinition struct { ... }

// --- Structs for Grouping Service Responses ---

// GroupCalculationResult mirrors the expected response from the Grouping Service's
// /api/v1/groups/:group_id/results endpoint.
type GroupCalculationResult struct {
	GroupID         string    `json:"group_id"`
	MemberIDs       []string  `json:"member_ids"` // These are entity_instance_id from processed_entities
	CalculatedAt    time.Time `json:"calculated_at"`
	MemberCount     int       `json:"member_count"`
}

// ActionStep defines the structure for an individual action within a workflow's ActionSequenceJSON.
type ActionStep struct {
	ActionTemplateID string `json:"action_template_id"`
	ParametersJSON   string `json:"parameters_json"` // JSON string of parameters for this step
}
