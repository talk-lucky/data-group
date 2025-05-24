package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// --- Service Clients ---

// MetadataServiceAPIClient defines an interface for metadata service interactions.
type MetadataServiceAPIClient interface {
	GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error)
	GetActionTemplate(templateID string) (*ActionTemplate, error)
}

// GroupingServiceAPIClient defines an interface for grouping service interactions.
type GroupingServiceAPIClient interface {
	GetGroupMembers(groupID string) (*GroupCalculationResult, error)
}

// HTTPMetadataClient implements MetadataServiceAPIClient.
type HTTPMetadataClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPMetadataClient creates a new metadata client.
func NewHTTPMetadataClient(baseURL string) *HTTPMetadataClient {
	return &HTTPMetadataClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPMetadataClient) fetchAPI(url string, target interface{}) error {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error
		return fmt.Errorf("API at %s returned status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

// GetWorkflowDefinition fetches a workflow definition.
func (c *HTTPMetadataClient) GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error) {
	var wd WorkflowDefinition
	url := fmt.Sprintf("%s/api/v1/workflows/%s", c.BaseURL, workflowID)
	err := c.fetchAPI(url, &wd)
	return &wd, err
}

// GetActionTemplate fetches an action template.
func (c *HTTPMetadataClient) GetActionTemplate(templateID string) (*ActionTemplate, error) {
	var at ActionTemplate
	url := fmt.Sprintf("%s/api/v1/actiontemplates/%s", c.BaseURL, templateID)
	err := c.fetchAPI(url, &at)
	return &at, err
}

// HTTPGroupingServiceClient implements GroupingServiceAPIClient.
type HTTPGroupingServiceClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPGroupingServiceClient creates a new grouping client.
func NewHTTPGroupingServiceClient(baseURL string) *HTTPGroupingServiceClient {
	return &HTTPGroupingServiceClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetGroupMembers fetches group members.
func (c *HTTPGroupingServiceClient) GetGroupMembers(groupID string) (*GroupCalculationResult, error) {
	var gr GroupCalculationResult
	url := fmt.Sprintf("%s/api/v1/groups/%s/results", c.BaseURL, groupID)
	err := c.fetchAPI(url, &gr)
	return &gr, err
}

// --- Orchestration Service ---

// OrchestrationService handles workflow triggering and task publishing.
type OrchestrationService struct {
	natsJS         nats.JetStreamContext
	metadataClient MetadataServiceAPIClient
	groupingClient GroupingServiceAPIClient
	db             *sql.DB // Direct access to processed_entities table
}

// NewOrchestrationService creates a new OrchestrationService.
func NewOrchestrationService(js nats.JetStreamContext, metaClient MetadataServiceAPIClient, groupClient GroupingServiceAPIClient, db *sql.DB) *OrchestrationService {
	return &OrchestrationService{
		natsJS:         js,
		metadataClient: metaClient,
		groupingClient: groupClient,
		db:             db,
	}
}

// TriggerWorkflow fetches workflow details, group members, and publishes tasks to NATS.
func (s *OrchestrationService) TriggerWorkflow(workflowID string) error {
	log.Printf("Triggering workflow: %s", workflowID)

	// 1. Fetch WorkflowDefinition
	workflow, err := s.metadataClient.GetWorkflowDefinition(workflowID)
	if err != nil {
		return fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}
	if !workflow.IsEnabled {
		log.Printf("Workflow %s (%s) is disabled. Skipping.", workflow.Name, workflowID)
		return nil // Not an error, but workflow won't run
	}
	log.Printf("Fetched workflow: %s (Trigger: %s)", workflow.Name, workflow.TriggerType)

	var entityInstanceIDs []string
	var groupID string // To store parsed group_id

	// 2. Handle Trigger Type and Fetch Group Members if applicable
	if workflow.TriggerType == "on_group_update" {
		var triggerConf struct {
			GroupID string `json:"group_id"`
		}
		if err := json.Unmarshal([]byte(workflow.TriggerConfig), &triggerConf); err != nil {
			return fmt.Errorf("failed to parse trigger_config for workflow %s: %w", workflowID, err)
		}
		if triggerConf.GroupID == "" {
			return fmt.Errorf("group_id missing in trigger_config for on_group_update workflow %s", workflowID)
		}
		groupID = triggerConf.GroupID // Store for logging/context
		log.Printf("Workflow %s triggered by group update for group_id: %s. Fetching members...", workflowID, groupID)

		groupResults, err := s.groupingClient.GetGroupMembers(groupID)
		if err != nil {
			return fmt.Errorf("failed to fetch members for group %s (workflow %s): %w", groupID, workflowID, err)
		}
		entityInstanceIDs = groupResults.MemberIDs
		log.Printf("Found %d members for group %s", len(entityInstanceIDs), groupID)
		if len(entityInstanceIDs) == 0 {
			log.Printf("No members in group %s for workflow %s. No tasks will be dispatched.", groupID, workflowID)
			return nil
		}
	} else if workflow.TriggerType == "manual" {
		log.Printf("Workflow %s is manually triggered. No specific entity instances from group trigger.", workflowID)
		// For manual triggers, we might need a different mechanism to specify target entities,
		// or the actions might not be entity-specific.
		// For now, if no entities, actions will run once if not entity specific, or not at all if they expect an entity.
		// This example assumes actions might run once without specific entity context if entityInstanceIDs is empty.
	} else {
		return fmt.Errorf("unsupported trigger_type '%s' for workflow %s", workflow.TriggerType, workflowID)
	}

	// 3. Parse ActionSequenceJSON
	var actionSequence []ActionStep
	if err := json.Unmarshal([]byte(workflow.ActionSequenceJSON), &actionSequence); err != nil {
		return fmt.Errorf("failed to parse action_sequence_json for workflow %s: %w", workflowID, err)
	}
	if len(actionSequence) == 0 {
		log.Printf("Workflow %s has an empty action sequence. Nothing to do.", workflowID)
		return nil
	}
	log.Printf("Parsed %d action steps for workflow %s", len(actionSequence), workflowID)

	// 4. Process each action step
	for i, step := range actionSequence {
		log.Printf("Processing action step %d/%d for workflow %s: TemplateID %s", i+1, len(actionSequence), workflowID, step.ActionTemplateID)
		actionTemplate, err := s.metadataClient.GetActionTemplate(step.ActionTemplateID)
		if err != nil {
			log.Printf("Error fetching action template %s for workflow %s: %v. Skipping step.", step.ActionTemplateID, workflowID, err)
			continue // Or return error to fail the whole workflow
		}

		// Parse step-specific parameters
		var stepParams map[string]interface{}
		if err := json.Unmarshal([]byte(step.ParametersJSON), &stepParams); err != nil {
			log.Printf("Error parsing parameters_json for step %d (template %s), workflow %s: %v. Skipping step.", i+1, step.ActionTemplateID, workflowID, err)
			continue
		}

		// Dispatch tasks
		if len(entityInstanceIDs) > 0 {
			// Dispatch one task per entity instance
			for _, entityInstanceID := range entityInstanceIDs {
				entityData, err := s.fetchEntityInstanceData(entityInstanceID)
				if err != nil {
					log.Printf("Error fetching entity instance data for ID %s (step %d, workflow %s): %v. Skipping task for this entity.", entityInstanceID, i+1, workflowID, err)
					continue
				}

				taskMsg := TaskMessage{
					TaskID:           uuid.New().String(),
					WorkflowID:       workflowID,
					ActionTemplateID: actionTemplate.ID,
					ActionType:       actionTemplate.ActionType,
					TemplateContent:  actionTemplate.TemplateContent,
					ActionParams:     stepParams,
					EntityInstanceID: entityInstanceID, // Added for clarity
					EntityInstance:   entityData,
				}
				if err := s.publishTask(taskMsg); err != nil {
					log.Printf("Error publishing task for entity %s, step %d, workflow %s: %v", entityInstanceID, i+1, workflowID, err)
					// Decide if one failed publish should stop others
				}
			}
		} else {
			// No specific entities (e.g., manual trigger not targeting a group, or action not entity-specific)
			// Publish one task without entity instance data, or with general context if applicable
			log.Printf("No specific entity instances for step %d (template %s), workflow %s. Publishing one general task.", i+1, step.ActionTemplateID, workflowID)
			taskMsg := TaskMessage{
				TaskID:           uuid.New().String(),
				WorkflowID:       workflowID,
				ActionTemplateID: actionTemplate.ID,
				ActionType:       actionTemplate.ActionType,
				TemplateContent:  actionTemplate.TemplateContent,
				ActionParams:     stepParams,
				EntityInstanceID: "", // No specific entity
				EntityInstance:   nil,
			}
			if err := s.publishTask(taskMsg); err != nil {
				log.Printf("Error publishing general task for step %d, workflow %s: %v", i+1, workflowID, err)
			}
		}
	}

	log.Printf("Successfully processed and dispatched tasks for workflow: %s (triggered by group: %s)", workflowID, groupID)
	return nil
}

// fetchEntityInstanceData retrieves the 'data' field from the 'processed_entities' table.
func (s *OrchestrationService) fetchEntityInstanceData(entityInstanceID string) (map[string]interface{}, error) {
	var jsonData []byte // sql.NullString might be better if data can be legitimately NULL
	err := s.db.QueryRow("SELECT data FROM processed_entities WHERE id = $1", entityInstanceID).Scan(&jsonData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no processed_entity found with id %s", entityInstanceID)
		}
		return nil, fmt.Errorf("failed to query processed_entity with id %s: %w", entityInstanceID, err)
	}

	if jsonData == nil || len(jsonData) == 0 {
		// This case might happen if the 'data' column is NULL or empty JSON '{}'
		log.Printf("Warning: processed_entity with id %s has NULL or empty data.", entityInstanceID)
		return make(map[string]interface{}), nil // Return empty map instead of error
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for processed_entity %s: %w", entityInstanceID, err)
	}
	return data, nil
}


// publishTask marshals and publishes a TaskMessage to NATS JetStream.
func (s *OrchestrationService) publishTask(taskMsg TaskMessage) error {
	subject := fmt.Sprintf("actions.%s", taskMsg.ActionType) // e.g., actions.webhook, actions.email
	payload, err := json.Marshal(taskMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal TaskMessage (TaskID %s): %w", taskMsg.TaskID, err)
	}

	// Ensure stream exists (idempotent operation)
	// Stream name could be ACTIONS_STREAM, capturing subjects "actions.>"
	streamName := "ACTIONS" // Or more specific like "WORKFLOW_TASKS"
	_, err = s.natsJS.StreamInfo(streamName)
	if err != nil { // If stream doesn't exist, try to create it
		log.Printf("Stream %s not found, attempting to create it for subject %s...", streamName, subject)
		_, createErr := s.natsJS.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{fmt.Sprintf("actions.>")}, // Capture all action types
			Storage:  nats.FileStorage,      // Or MemoryStorage for non-persistent
		})
		if createErr != nil {
			return fmt.Errorf("failed to create NATS stream %s: %w", streamName, createErr)
		}
		log.Printf("Successfully created NATS stream %s", streamName)
	}


	pubAck, err := s.natsJS.Publish(subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish TaskMessage (TaskID %s) to subject %s: %w", taskMsg.TaskID, subject, err)
	}

	log.Printf("Published TaskID %s to subject %s (Stream: %s, Sequence: %d)",
		taskMsg.TaskID, subject, pubAck.Stream, pubAck.Sequence)
	return nil
}
