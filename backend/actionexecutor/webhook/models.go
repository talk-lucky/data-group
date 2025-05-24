package main

// TaskMessage is the payload received from NATS.
// It must be identical to the one defined in the orchestration service.
type TaskMessage struct {
	TaskID            string                 `json:"task_id"`
	WorkflowID        string                 `json:"workflow_id"`
	ActionTemplateID  string                 `json:"action_template_id"`
	ActionType        string                 `json:"action_type"`
	TemplateContent   string                 `json:"template_content"`  // JSON string of WebhookTemplate or other types
	ActionParams      map[string]interface{} `json:"action_params"`     // User-defined params for this action step
	EntityInstanceID  string                 `json:"entity_instance_id"`
	EntityInstance    map[string]interface{} `json:"entity_instance"`   // Data from processed_entities
}

// WebhookTemplate defines the structure for webhook action configurations.
// It's parsed from TaskMessage.TemplateContent.
type WebhookTemplate struct {
	URLTemplate     string            `json:"url_template"`
	Method          string            `json:"method"` // e.g., "POST", "GET", "PUT"
	PayloadTemplate string            `json:"payload_template,omitempty"` // JSON string as a Go template
	HeadersTemplate map[string]string `json:"headers_template,omitempty"` // map of header names to header value templates
}

// TemplateData is the structure passed to Go templates for rendering.
// It combines EntityInstance data and ActionParams.
type TemplateData struct {
	EntityInstance map[string]interface{}
	ActionParams   map[string]interface{}
}
