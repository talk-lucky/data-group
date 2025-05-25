package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
)

// --- Client Interfaces ---

// MetadataServiceClient defines methods to interact with the Metadata Service.
type MetadataServiceClient interface {
	ListEnabledSchedules() ([]ScheduleDefinition, error)
}

// IngestionServiceClient defines methods to interact with the Ingestion Service.
type IngestionServiceClient interface {
	TriggerIngestion(sourceID string) error
}

// --- HTTP Client Implementations ---

// HTTPMetadataServiceClient implements MetadataServiceClient.
type HTTPMetadataServiceClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPMetadataServiceClient creates a new HTTP client for the Metadata Service.
func NewHTTPMetadataServiceClient(baseURL string) *HTTPMetadataServiceClient {
	return &HTTPMetadataServiceClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ListEnabledSchedules fetches all schedules from the metadata service and filters for enabled ones.
func (c *HTTPMetadataServiceClient) ListEnabledSchedules() ([]ScheduleDefinition, error) {
	url := fmt.Sprintf("%s/api/v1/schedules", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to metadata service for schedules: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call metadata service at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned non-OK status %d for schedules at %s", resp.StatusCode, url)
	}

	var allSchedules []ScheduleDefinition
	if err := json.NewDecoder(resp.Body).Decode(&allSchedules); err != nil {
		return nil, fmt.Errorf("failed to decode response from metadata service for schedules: %w", err)
	}

	var enabledSchedules []ScheduleDefinition
	for _, schedule := range allSchedules {
		if schedule.IsEnabled {
			enabledSchedules = append(enabledSchedules, schedule)
		}
	}
	log.Printf("Fetched %d total schedules, %d are enabled.", len(allSchedules), len(enabledSchedules))
	return enabledSchedules, nil
}


// HTTPIngestionServiceClient implements IngestionServiceClient.
type HTTPIngestionServiceClient struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewHTTPIngestionServiceClient creates a new HTTP client for the Ingestion Service.
func NewHTTPIngestionServiceClient(baseURL string) *HTTPIngestionServiceClient {
	return &HTTPIngestionServiceClient{
		BaseURL:    baseURL,
		HttpClient: &http.Client{Timeout: 30 * time.Second}, // Increased timeout for trigger actions
	}
}

// TriggerIngestion makes a POST request to the Ingestion Service to trigger ingestion for a source.
func (c *HTTPIngestionServiceClient) TriggerIngestion(sourceID string) error {
	url := fmt.Sprintf("%s/api/v1/ingest/trigger/%s", c.BaseURL, sourceID) 
	req, err := http.NewRequest("POST", url, nil) 
	if err != nil {
		return fmt.Errorf("failed to create POST request to ingestion service for sourceID %s: %w", sourceID, err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call ingestion service at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted { 
		// Attempt to read body for more detailed error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ingestion service returned non-OK/Accepted status %d for triggering sourceID %s at %s. Response: %s", resp.StatusCode, sourceID, url, string(bodyBytes))
	}

	log.Printf("Successfully triggered ingestion for sourceID: %s. Status: %d", sourceID, resp.StatusCode)
	return nil
}


// --- Data Models ---
type ScheduleDefinition struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	CronExpression    string    `json:"cron_expression"`
	TaskType          string    `json:"task_type"` 
	TaskParameters    string    `json:"task_parameters"` 
	IsEnabled         bool      `json:"is_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type IngestDataSourceTaskParams struct {
	SourceID string `json:"source_id"`
}

// --- SchedulerService ---
type SchedulerService struct {
	cronRunner          *cron.Cron
	metadataClient      MetadataServiceClient
	ingestionClient     IngestionServiceClient
}

func NewSchedulerService(
	metaClient MetadataServiceClient,
	ingestClient IngestionServiceClient,
) *SchedulerService {
	log.Println("Initializing Scheduler Service...")
	return &SchedulerService{
		metadataClient:      metaClient,
		ingestionClient:     ingestClient,
		cronRunner: cron.New( // Initialize cronRunner here
			cron.WithSeconds(), // Use seconds field in cron expressions
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger), 
				cron.Recover(cron.DefaultLogger),          
			),
		),
	}
}

// runIngestionTask is called by a cron job to trigger data ingestion.
func (s *SchedulerService) runIngestionTask(scheduleID, scheduleName, sourceID string) {
	log.Printf("Executing scheduled task from Schedule ID: %s (%s) - Triggering ingestion for Source ID: %s", scheduleID, scheduleName, sourceID)
	err := s.ingestionClient.TriggerIngestion(sourceID)
	if err != nil {
		log.Printf("Error triggering ingestion for Source ID %s (scheduled by %s - %s): %v", sourceID, scheduleID, scheduleName, err)
	} else {
		log.Printf("Successfully triggered ingestion for Source ID %s (scheduled by %s - %s)", sourceID, scheduleID, scheduleName)
	}
}

// Start initializes the cron runner, loads schedules, and starts the scheduling process.
func (s *SchedulerService) Start() error {
	log.Println("Scheduler Service Start() called.")

	if s.cronRunner == nil { // Should have been initialized in NewSchedulerService
		log.Println("Cron runner not initialized, initializing now.")
		s.cronRunner = cron.New(
			cron.WithSeconds(),
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger), 
				cron.Recover(cron.DefaultLogger),          
			),
		)
	}
	
	log.Println("Loading schedules...")
	schedules, err := s.metadataClient.ListEnabledSchedules()
	if err != nil {
	    return fmt.Errorf("failed to load enabled schedules from metadata service: %w", err)
	}

	log.Printf("Found %d enabled schedules to process.", len(schedules))
	for _, schedule := range schedules {
		if schedule.TaskType == "ingest_data_source" {
			log.Printf("Attempting to schedule task for: %s (ID: %s), Type: %s, Cron: '%s'", 
				schedule.Name, schedule.ID, schedule.TaskType, schedule.CronExpression)

			var params IngestDataSourceTaskParams
			if err := json.Unmarshal([]byte(schedule.TaskParameters), &params); err != nil {
				log.Printf("Error parsing TaskParameters for schedule ID %s (%s): %v. Skipping schedule.", schedule.ID, schedule.Name, err)
				continue
			}

			if params.SourceID == "" {
				log.Printf("SourceID is empty in TaskParameters for schedule ID %s (%s). Skipping schedule.", schedule.ID, schedule.Name)
				continue
			}

			// Capture schedule and params for the closure
			currentSchedule := schedule 
			currentParams := params 

			entryID, err := s.cronRunner.AddFunc(currentSchedule.CronExpression, func() {
				s.runIngestionTask(currentSchedule.ID, currentSchedule.Name, currentParams.SourceID)
			})
			if err != nil {
				log.Printf("Error adding job for schedule ID %s (%s) with cron '%s': %v", 
					currentSchedule.ID, currentSchedule.Name, currentSchedule.CronExpression, err)
			} else {
				log.Printf("Successfully added job for schedule ID %s (%s), EntryID: %d, Cron: '%s', Task: Ingest SourceID %s",
					currentSchedule.ID, currentSchedule.Name, entryID, currentSchedule.CronExpression, currentParams.SourceID)
			}
		} else {
			log.Printf("TaskType '%s' for schedule ID %s (%s) is not yet supported by the scheduler. Skipping.", 
				schedule.TaskType, schedule.ID, schedule.Name)
		}
	}

	log.Println("Starting cron runner...")
	s.cronRunner.Start() // This is non-blocking
	log.Println("Cron runner started. Scheduler Service is active and jobs are scheduled.")
	
	return nil
}

// Stop gracefully shuts down the cron runner.
func (s *SchedulerService) Stop() {
	log.Println("Scheduler Service Stop() called.")
	if s.cronRunner != nil {
		log.Println("Stopping cron runner... waiting for jobs to complete.")
		ctx := s.cronRunner.Stop() 
		select {
		case <-ctx.Done():
			log.Println("Cron runner stopped gracefully.")
		case <-time.After(15 * time.Second): 
			log.Println("Cron runner shutdown timed out.")
		}
	} else {
		log.Println("Cron runner was not initialized or already stopped.")
	}
}

// Note: emptyBody variable was removed as it was unused.
// The io.ReadAll for error body in HTTPIngestionServiceClient.TriggerIngestion was added.
// Cron runner initialization moved to NewSchedulerService for consistency.
// Added check for nil cronRunner in Start, though it should always be initialized by New.
