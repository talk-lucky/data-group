package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	// Ensure _ "github.com/lib/pq" is in main.go if db is used, not directly here unless this becomes a main package
)

// --- Service Clients & Interfaces ---

// MetadataServiceAPIClient defines an interface for metadata service interactions.
type MetadataServiceAPIClient interface {
	GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error)
	GetActionTemplate(templateID string) (*ActionTemplate, error)
	ListWorkflows() ([]WorkflowDefinition, error)
}

// GroupingServiceAPIClient defines an interface for grouping service interactions.
type GroupingServiceAPIClient interface {
	GetGroupMembers(groupID string) (*GroupCalculationResult, error)
}

// NatsJetStreamPublisher defines an interface for NATS JetStream operations used by OrchestrationService.
// This allows for easier mocking in tests.
type NatsJetStreamPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
	StreamInfo(stream string, opts ...nats.StreamInfoOpt) (*nats.StreamInfo, error)
	AddStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	Subscribe(subject string, cb nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error)
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
		return fmt.Errorf("API at %s returned status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *HTTPMetadataClient) GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error) {
	var wd WorkflowDefinition
	url := fmt.Sprintf("%s/api/v1/workflows/%s", c.BaseURL, workflowID)
	err := c.fetchAPI(url, &wd)
	return &wd, err
}

func (c *HTTPMetadataClient) GetActionTemplate(templateID string) (*ActionTemplate, error) {
	var at ActionTemplate
	url := fmt.Sprintf("%s/api/v1/actiontemplates/%s", c.BaseURL, templateID)
	err := c.fetchAPI(url, &at)
	return &at, err
}

func (c *HTTPMetadataClient) ListWorkflows() ([]WorkflowDefinition, error) {
	var workflows []WorkflowDefinition
	url := fmt.Sprintf("%s/api/v1/workflows", c.BaseURL)
	err := c.fetchAPI(url, &workflows)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows from %s: %w", url, err)
	}
	return workflows, nil
}

// HTTPGroupingServiceClient implements GroupingServiceAPIClient.
type HTTPGroupingServiceClient struct {
	BaseURL    string
	HttpClient *http.Client
}

func NewHTTPGroupingServiceClient(baseURL string) *HTTPGroupingServiceClient {
	return &HTTPGroupingServiceClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPGroupingServiceClient) GetGroupMembers(groupID string) (*GroupCalculationResult, error) {
	var gr GroupCalculationResult
	url := fmt.Sprintf("%s/api/v1/groups/%s/results", c.BaseURL, groupID)
	err := c.fetchAPI(url, &gr)
	return &gr, err
}

// --- Orchestration Service ---
const (
	groupEventsSubject          = "GROUP.updated.>"
	groupEventsConsumerDurable  = "OrchestrationServiceGroupConsumer"
	actionTasksStreamName       = "ACTIONS"
	actionTasksSubjectQualifier = "actions"
)

type OrchestrationService struct {
	natsJS         NatsJetStreamPublisher // Changed type to interface
	natsSub        *nats.Subscription
	metadataClient MetadataServiceAPIClient
	groupingClient GroupingServiceAPIClient
	db             *sql.DB
}

// NewOrchestrationService creates a new OrchestrationService.
// The 'js' parameter is now of type NatsJetStreamPublisher.
func NewOrchestrationService(js NatsJetStreamPublisher, metaClient MetadataServiceAPIClient, groupClient GroupingServiceAPIClient, db *sql.DB) *OrchestrationService {
	s := &OrchestrationService{
		natsJS:         js, // Assign interface
		metadataClient: metaClient,
		groupingClient: groupClient,
		db:             db,
	}
	go s.subscribeToGroupEvents() // This will now use the interface's Subscribe method
	return s
}

func (s *OrchestrationService) subscribeToGroupEvents() {
	var err error
	// s.natsJS is now NatsJetStreamPublisher, so this calls the interface method
	s.natsSub, err = s.natsJS.Subscribe(
		groupEventsSubject,
		s.handleGroupUpdateMsg,
		nats.Durable(groupEventsConsumerDurable),
		nats.AckNone(),
	)
	if err != nil {
		log.Printf("Error: Failed to subscribe to NATS subject '%s': %v. Group update events will not be processed automatically.", groupEventsSubject, err)
	} else {
		log.Printf("Successfully subscribed to NATS subject '%s'", groupEventsSubject)
	}
}

func (s *OrchestrationService) handleGroupUpdateMsg(msg *nats.Msg) {
	subjectParts := strings.Split(msg.Subject, ".")
	if len(subjectParts) < 3 {
		log.Printf("Error: Received message on unexpected subject format: %s. Expected 'GROUP.updated.<groupID>'", msg.Subject)
		return
	}
	groupID := subjectParts[2]
	log.Printf("Received NATS event on subject: %s. Parsed groupID: %s. Data: %s", msg.Subject, groupID, string(msg.Data))
	s.TriggerWorkflowForGroupUpdate(groupID)
}

func (s *OrchestrationService) executeWorkflow(wf WorkflowDefinition, groupMembers *GroupCalculationResult, triggerContext string) error {
	log.Printf("Executing workflow: %s (ID: %s), TriggerContext: %s", wf.Name, wf.ID, triggerContext)

	var actionSequence []ActionStep
	if err := json.Unmarshal([]byte(wf.ActionSequenceJSON), &actionSequence); err != nil {
		return fmt.Errorf("failed to parse action_sequence_json for workflow %s: %w", wf.ID, err)
	}
	if len(actionSequence) == 0 {
		log.Printf("Workflow %s has an empty action sequence. Nothing to do.", wf.ID)
		return nil
	}
	log.Printf("Parsed %d action steps for workflow %s", len(actionSequence), wf.ID)

	for i, step := range actionSequence {
		log.Printf("Processing action step %d/%d for workflow %s: TemplateID %s", i+1, len(actionSequence), wf.ID, step.ActionTemplateID)
		actionTemplate, errAT := s.metadataClient.GetActionTemplate(step.ActionTemplateID)
		if errAT != nil {
			log.Printf("Error fetching action template %s for workflow %s: %v. Skipping step.", step.ActionTemplateID, wf.ID, errAT)
			continue 
		}

		var stepParams map[string]interface{}
		if errP := json.Unmarshal([]byte(step.ParametersJSON), &stepParams); errP != nil {
			log.Printf("Error parsing parameters_json for step %d (template %s), workflow %s: %v. Skipping step.", i+1, step.ActionTemplateID, wf.ID, errP)
			continue
		}

		if groupMembers != nil && len(groupMembers.MemberIDs) > 0 {
			log.Printf("Dispatching tasks for %d group members for workflow %s, step %d", len(groupMembers.MemberIDs), wf.ID, i+1)
			for _, entityInstanceID := range groupMembers.MemberIDs {
				entityData, errED := s.fetchEntityInstanceData(entityInstanceID)
				if errED != nil {
					log.Printf("Error fetching entity instance data for ID %s (step %d, workflow %s): %v. Skipping task for this entity.", entityInstanceID, i+1, wf.ID, errED)
					continue
				}
				taskMsg := TaskMessage{
					TaskID:           uuid.NewString(),
					WorkflowID:       wf.ID,
					ActionTemplateID: actionTemplate.ID,
					ActionType:       actionTemplate.ActionType,
					TemplateContent:  actionTemplate.TemplateContent,
					ActionParams:     stepParams,
					EntityInstanceID: entityInstanceID,
					EntityInstance:   entityData,
				}
				if errPub := s.publishTask(taskMsg); errPub != nil {
					log.Printf("Error publishing task for entity %s, step %d, workflow %s: %v", entityInstanceID, i+1, wf.ID, errPub)
				}
			}
		} else {
			log.Printf("No specific entity instances for step %d (template %s), workflow %s (Context: %s). Publishing one general task.", i+1, step.ActionTemplateID, wf.ID, triggerContext)
			taskMsg := TaskMessage{
				TaskID:           uuid.NewString(),
				WorkflowID:       wf.ID,
				ActionTemplateID: actionTemplate.ID,
				ActionType:       actionTemplate.ActionType,
				TemplateContent:  actionTemplate.TemplateContent,
				ActionParams:     stepParams,
			}
			if errPub := s.publishTask(taskMsg); errPub != nil {
				log.Printf("Error publishing general task for step %d, workflow %s: %v", i+1, wf.ID, errPub)
			}
		}
	}
	log.Printf("Successfully completed execution of workflow: %s (ID: %s), TriggerContext: %s", wf.Name, wf.ID, triggerContext)
	return nil
}

func (s *OrchestrationService) TriggerWorkflowForGroupUpdate(groupID string) {
	log.Printf("Processing group update event for groupID: %s. Looking for matching workflows.", groupID)

	workflows, err := s.metadataClient.ListWorkflows()
	if err != nil {
		log.Printf("Error listing workflows for group update trigger (groupID %s): %v", groupID, err)
		return
	}

	if len(workflows) == 0 {
		log.Printf("No workflows defined. Nothing to trigger for groupID: %s.", groupID)
		return
	}

	foundAndTriggered := 0
	for _, wf := range workflows {
		if wf.IsEnabled && wf.TriggerType == "on_group_update" {
			var triggerConf struct {
				GroupID string `json:"group_id"`
			}
			if errJ := json.Unmarshal([]byte(wf.TriggerConfig), &triggerConf); errJ != nil {
				log.Printf("Error parsing trigger_config for workflow %s (ID: %s) for group update: %v. Skipping.", wf.Name, wf.ID, errJ)
				continue
			}

			if triggerConf.GroupID == groupID {
				log.Printf("Matching workflow found for group update: %s (ID: %s). Fetching group members...", wf.Name, wf.ID)
				groupResults, errGR := s.groupingClient.GetGroupMembers(groupID)
				if errGR != nil {
					log.Printf("Error fetching group members for group %s (for workflow %s): %v. Skipping this workflow.", groupID, wf.ID, errGR)
					continue 
				}
				log.Printf("Executing workflow %s for group %s with %d members.", wf.Name, groupID, len(groupResults.MemberIDs))
				errExec := s.executeWorkflow(wf, groupResults, fmt.Sprintf("group_update_event: %s", groupID))
				if errExec != nil {
					log.Printf("Error executing workflow %s (ID: %s) for group update (groupID %s): %v", wf.Name, wf.ID, groupID, errExec)
				} else {
					foundAndTriggered++
				}
			}
		}
	}
	log.Printf("Finished processing group update for groupID: %s. Found and attempted to trigger %d matching workflows.", groupID, foundAndTriggered)
}

func (s *OrchestrationService) TriggerWorkflow(workflowID string) error {
	log.Printf("Manually triggering workflow: %s", workflowID)

	workflow, err := s.metadataClient.GetWorkflowDefinition(workflowID)
	if err != nil {
		return fmt.Errorf("failed to fetch workflow %s for manual trigger: %w", workflowID, err)
	}
	if !workflow.IsEnabled {
		log.Printf("Workflow %s (ID: %s) is disabled. Manual trigger skipped.", workflow.Name, workflowID)
		return nil
	}

	var groupMembers *GroupCalculationResult 
	triggerContext := fmt.Sprintf("manual_trigger, workflow_type: %s", workflow.TriggerType)

	if workflow.TriggerType == "on_group_update" {
		var triggerConf struct {
			GroupID string `json:"group_id"`
		}
		if err := json.Unmarshal([]byte(workflow.TriggerConfig), &triggerConf); err != nil {
			return fmt.Errorf("failed to parse trigger_config for manually triggered group workflow %s: %w", workflowID, err)
		}
		if triggerConf.GroupID == "" {
			return fmt.Errorf("group_id missing in trigger_config for manually triggered 'on_group_update' workflow %s", workflowID)
		}
		
		log.Printf("Manually triggering 'on_group_update' workflow %s for group_id: %s. Fetching members...", workflowID, triggerConf.GroupID)
		groupMembers, err = s.groupingClient.GetGroupMembers(triggerConf.GroupID)
		if err != nil {
			return fmt.Errorf("failed to fetch members for group %s (for manually triggered workflow %s): %w", triggerConf.GroupID, workflowID, err)
		}
		triggerContext = fmt.Sprintf("manual_trigger_for_group_workflow, group_id: %s", triggerConf.GroupID)
		log.Printf("Fetched %d members for group %s for manual trigger of workflow %s", len(groupMembers.MemberIDs), triggerConf.GroupID, workflowID)
	}
	
	return s.executeWorkflow(*workflow, groupMembers, triggerContext)
}

func (s *OrchestrationService) fetchEntityInstanceData(entityInstanceID string) (map[string]interface{}, error) {
	var jsonData []byte
	err := s.db.QueryRow("SELECT attributes FROM processed_entities WHERE id = $1", entityInstanceID).Scan(&jsonData)
	if err != nil {
		if err == sql.ErrNoRows { return nil, fmt.Errorf("no processed_entity found with id %s", entityInstanceID) }
		return nil, fmt.Errorf("failed to query processed_entity with id %s: %w", entityInstanceID, err)
	}
	if jsonData == nil || len(jsonData) == 0 {
		log.Printf("Warning: processed_entity with id %s has NULL or empty attributes.", entityInstanceID)
		return make(map[string]interface{}), nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil { return nil, fmt.Errorf("failed to unmarshal attributes for processed_entity %s: %w", entityInstanceID, err) }
	return data, nil
}

func (s *OrchestrationService) publishTask(taskMsg TaskMessage) error {
	subject := fmt.Sprintf("%s.%s", actionTasksSubjectQualifier, taskMsg.ActionType)
	payload, err := json.Marshal(taskMsg)
	if err != nil { return fmt.Errorf("failed to marshal TaskMessage (TaskID %s): %w", taskMsg.TaskID, err) }

	// s.natsJS is now NatsJetStreamPublisher, so these are interface calls
	_, err = s.natsJS.StreamInfo(actionTasksStreamName)
	if err != nil {
		log.Printf("Stream %s not found, attempting to create it for subject %s...", actionTasksStreamName, subject)
		_, createErr := s.natsJS.AddStream(&nats.StreamConfig{ Name: actionTasksStreamName, Subjects: []string{fmt.Sprintf("%s.>", actionTasksSubjectQualifier)}, Storage: nats.FileStorage })
		if createErr != nil { return fmt.Errorf("failed to create NATS stream %s: %w", actionTasksStreamName, createErr) }
		log.Printf("Successfully created NATS stream %s", actionTasksStreamName)
	}

	pubAck, err := s.natsJS.Publish(subject, payload)
	if err != nil { return fmt.Errorf("failed to publish TaskMessage (TaskID %s) to subject %s: %w", taskMsg.TaskID, subject, err) }
	log.Printf("Published TaskID %s to subject %s (Stream: %s, Sequence: %d)", taskMsg.TaskID, subject, pubAck.Stream, pubAck.Sequence)
	return nil
}

// --- Struct definitions ---
type WorkflowDefinition struct {ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description,omitempty"`; TriggerType string `json:"trigger_type"`; TriggerConfig string `json:"trigger_config"`; ActionSequenceJSON string `json:"action_sequence_json"`; IsEnabled bool `json:"is_enabled"`}
type ActionTemplate struct {ID string `json:"id"`; Name string `json:"name"`; ActionType string `json:"action_type"`; TemplateContent string `json:"template_content"`}
type ActionStep struct {ActionTemplateID string `json:"action_template_id"`; ParametersJSON string `json:"parameters_json"`}
type GroupCalculationResult struct {GroupID string `json:"group_id"`; MemberIDs []string `json:"member_ids"`; CalculatedAt time.Time `json:"calculated_at"`; MemberCount int `json:"member_count"`}
type TaskMessage struct {TaskID string `json:"task_id"`; WorkflowID string `json:"workflow_id"`; ActionTemplateID string `json:"action_template_id"`; ActionType string `json:"action_type"`; TemplateContent string `json:"template_content"`; ActionParams map[string]interface{} `json:"action_params"`; EntityInstanceID string `json:"entity_instance_id,omitempty"`; EntityInstance map[string]interface{} `json:"entity_instance,omitempty"`}
