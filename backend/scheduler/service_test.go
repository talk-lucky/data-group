package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock MetadataServiceAPIClient ---
type MockMetadataServiceClient struct {
	mock.Mock
}

func (m *MockMetadataServiceClient) ListEnabledSchedules() ([]ScheduleDefinition, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ScheduleDefinition), args.Error(1)
}

// --- Mock IngestionServiceAPIClient ---
type MockIngestionServiceClient struct {
	mock.Mock
	LastTriggeredSourceID string
}

func (m *MockIngestionServiceClient) TriggerIngestion(sourceID string) error {
	m.LastTriggeredSourceID = sourceID
	args := m.Called(sourceID)
	return args.Error(0)
}

// Helper to create valid JSON string for TaskParameters
func makeTaskParams(t *testing.T, sourceID string) string {
	t.Helper()
	params := IngestDataSourceTaskParams{SourceID: sourceID}
	jsonBytes, err := json.Marshal(params)
	require.NoError(t, err)
	return string(jsonBytes)
}

// --- Tests for SchedulerService.Start() ---
func TestSchedulerService_Start(t *testing.T) {

	t.Run("Successful Loading and Adding of Ingestion Jobs", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		schedules := []ScheduleDefinition{
			{ID: "s1", Name: "Schedule 1", CronExpression: "@every 5s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src1"), IsEnabled: true},
			{ID: "s2", Name: "Schedule 2", CronExpression: "@every 6s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src2"), IsEnabled: true},
		}
		mockMetaClient.On("ListEnabledSchedules").Return(schedules, nil).Once()

		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start() // service.cronRunner is initialized in New and started in Start
		
		require.NoError(t, err)
		assert.Len(t, service.cronRunner.Entries(), 2)

		mockMetaClient.AssertExpectations(t)
		// Note: Verifying that the correct functions are scheduled is complex with robfig/cron.
		// We primarily verify by the number of entries and by testing runIngestionTask separately.
		service.Stop() // Stop the cron runner
	})

	t.Run("No Enabled Schedules", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		mockMetaClient.On("ListEnabledSchedules").Return([]ScheduleDefinition{}, nil).Once()

		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start()
		
		require.NoError(t, err)
		assert.Empty(t, service.cronRunner.Entries())
		mockMetaClient.AssertExpectations(t)
		service.Stop()
	})

	t.Run("Error Fetching Schedules", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)
		expectedErr := errors.New("metadata fetch error")

		mockMetaClient.On("ListEnabledSchedules").Return(nil, expectedErr).Once()

		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start()
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		mockMetaClient.AssertExpectations(t)
		// service.Stop() not strictly needed as cronRunner.Start() might not be reached
	})

	t.Run("Schedule with Invalid Cron Expression", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		schedules := []ScheduleDefinition{
			{ID: "s1", Name: "Valid Schedule", CronExpression: "@every 5s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src1"), IsEnabled: true},
			{ID: "s2", Name: "Invalid Cron", CronExpression: "not a cron", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src2"), IsEnabled: true},
		}
		mockMetaClient.On("ListEnabledSchedules").Return(schedules, nil).Once()

		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start() // Start logs errors for invalid cron but doesn't return error itself
		
		require.NoError(t, err)
		assert.Len(t, service.cronRunner.Entries(), 1, "Only the valid schedule should be added")
		mockMetaClient.AssertExpectations(t)
		service.Stop()
	})

	t.Run("Schedule with Invalid TaskParameters JSON", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		schedules := []ScheduleDefinition{
			{ID: "s1", Name: "Valid Schedule", CronExpression: "@every 5s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src1"), IsEnabled: true},
			{ID: "s2", Name: "Invalid Params", CronExpression: "@every 6s", TaskType: "ingest_data_source", TaskParameters: `{"source_id":}`, IsEnabled: true},
		}
		mockMetaClient.On("ListEnabledSchedules").Return(schedules, nil).Once()
		
		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start()

		require.NoError(t, err)
		assert.Len(t, service.cronRunner.Entries(), 1, "Only the valid schedule should be added")
		mockMetaClient.AssertExpectations(t)
		service.Stop()
	})

	t.Run("Schedule with Missing source_id in TaskParameters", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		schedules := []ScheduleDefinition{
			{ID: "s1", Name: "Valid Schedule", CronExpression: "@every 5s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src1"), IsEnabled: true},
			{ID: "s2", Name: "Missing SourceID", CronExpression: "@every 6s", TaskType: "ingest_data_source", TaskParameters: `{}`, IsEnabled: true},
		}
		mockMetaClient.On("ListEnabledSchedules").Return(schedules, nil).Once()

		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start()
		
		require.NoError(t, err)
		assert.Len(t, service.cronRunner.Entries(), 1, "Only the valid schedule should be added")
		mockMetaClient.AssertExpectations(t)
		service.Stop()
	})

	t.Run("Unsupported TaskType", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)

		schedules := []ScheduleDefinition{
			{ID: "s1", Name: "Valid Schedule", CronExpression: "@every 5s", TaskType: "ingest_data_source", TaskParameters: makeTaskParams(t, "src1"), IsEnabled: true},
			{ID: "s2", Name: "Unsupported Type", CronExpression: "@every 6s", TaskType: "unsupported_type", TaskParameters: `{}`, IsEnabled: true},
		}
		mockMetaClient.On("ListEnabledSchedules").Return(schedules, nil).Once()
		
		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		err := service.Start()

		require.NoError(t, err)
		assert.Len(t, service.cronRunner.Entries(), 1, "Only the ingest_data_source schedule should be added")
		mockMetaClient.AssertExpectations(t)
		service.Stop()
	})
}

// --- Tests for SchedulerService.runIngestionTask() ---
func TestSchedulerService_runIngestionTask(t *testing.T) {
	
	t.Run("Successful TriggerIngestion Call", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient) // Not used directly here, but NewSchedulerService needs it
		mockIngestClient := new(MockIngestionServiceClient)
		
		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		testSourceID := "sourceABC"

		mockIngestClient.On("TriggerIngestion", testSourceID).Return(nil).Once()

		service.runIngestionTask("schedule123", "Test Schedule", testSourceID)

		mockIngestClient.AssertExpectations(t)
		assert.Equal(t, testSourceID, mockIngestClient.LastTriggeredSourceID)
	})

	t.Run("IngestionServiceClient Returns Error", func(t *testing.T) {
		mockMetaClient := new(MockMetadataServiceClient)
		mockIngestClient := new(MockIngestionServiceClient)
		
		service := NewSchedulerService(mockMetaClient, mockIngestClient)
		testSourceID := "sourceXYZ"
		expectedErr := errors.New("ingestion service error")

		mockIngestClient.On("TriggerIngestion", testSourceID).Return(expectedErr).Once()

		// Since runIngestionTask only logs errors, we can't assert the error directly.
		// We're mainly ensuring the mock was called.
		// Log verification would require capturing log output, which is more involved.
		service.runIngestionTask("schedule456", "Failing Schedule", testSourceID)

		mockIngestClient.AssertExpectations(t)
		assert.Equal(t, testSourceID, mockIngestClient.LastTriggeredSourceID)
	})
}
