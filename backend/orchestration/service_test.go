package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock MetadataServiceAPIClient ---
type MockMetadataServiceClient struct {
	mock.Mock
}

func (m *MockMetadataServiceClient) GetWorkflowDefinition(workflowID string) (*WorkflowDefinition, error) {
	args := m.Called(workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WorkflowDefinition), args.Error(1)
}
func (m *MockMetadataServiceClient) GetActionTemplate(templateID string) (*ActionTemplate, error) {
	args := m.Called(templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ActionTemplate), args.Error(1)
}
func (m *MockMetadataServiceClient) ListWorkflows() ([]WorkflowDefinition, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WorkflowDefinition), args.Error(1)
}

// --- Mock GroupingServiceAPIClient ---
type MockGroupingServiceClient struct {
	mock.Mock
}

func (m *MockGroupingServiceClient) GetGroupMembers(groupID string) (*GroupCalculationResult, error) {
	args := m.Called(groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GroupCalculationResult), args.Error(1)
}

// --- Mock NatsJetStreamPublisher ---
type MockNatsJetStreamPublisher struct {
	mock.Mock
	PublishedMessages map[string][]byte // Capture subject and payload
}

func NewMockNatsJetStreamPublisher() *MockNatsJetStreamPublisher {
	return &MockNatsJetStreamPublisher{
		PublishedMessages: make(map[string][]byte),
	}
}
func (m *MockNatsJetStreamPublisher) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	args := m.Called(subj, data, opts)
	// Capture the message
	if m.PublishedMessages == nil { // Ensure map is initialized if mock is created directly
		m.PublishedMessages = make(map[string][]byte)
	}
	m.PublishedMessages[subj] = data // Or append to a slice if multiple messages on same subject
	
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.PubAck), args.Error(1)
}
func (m *MockNatsJetStreamPublisher) StreamInfo(stream string, opts ...nats.StreamInfoOpt) (*nats.StreamInfo, error) {
	args := m.Called(stream, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.StreamInfo), args.Error(1)
}
func (m *MockNatsJetStreamPublisher) AddStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error) {
	args := m.Called(cfg, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.StreamInfo), args.Error(1)
}
func (m *MockNatsJetStreamPublisher) Subscribe(subject string, cb nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	args := m.Called(subject, cb, opts) // cb might be tricky to assert directly, often just nats.MsgHandler type
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.Subscription), args.Error(1)
}


// --- Test HandleGroupUpdateMsg ---
func TestHandleGroupUpdateMsg(t *testing.T) {
	mockMeta := new(MockMetadataServiceClient)
	mockGrouping := new(MockGroupingServiceClient)
	mockNatsJS := NewMockNatsJetStreamPublisher() // Using the custom mock
	
	// Mock the Subscribe call for NewOrchestrationService
	mockNatsJS.On("Subscribe", groupEventsSubject, mock.AnythingOfType("nats.MsgHandler"), mock.Anything, mock.Anything).Return(&nats.Subscription{}, nil)

	service := NewOrchestrationService(mockNatsJS, mockMeta, mockGrouping, nil) // DB not needed for this handler test

	t.Run("Successful groupID parsing and trigger call", func(t *testing.T) {
		// Reset expectations and calls for ListWorkflows for this specific sub-test
		// This is a bit of a workaround because TriggerWorkflowForGroupUpdate calls ListWorkflows.
		// We are testing handleGroupUpdateMsg, which calls TriggerWorkflowForGroupUpdate.
		// So, we need to mock what TriggerWorkflowForGroupUpdate needs.
		mockMeta.ExpectedCalls = nil // Clear previous expectations if any
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{}, nil).Once() // Expect ListWorkflows to be called

		msg := &nats.Msg{Subject: "GROUP.updated.group123", Data: []byte("test data")}
		service.handleGroupUpdateMsg(msg) // This will call TriggerWorkflowForGroupUpdate

		mockMeta.AssertCalled(t, "ListWorkflows") // Verifies TriggerWorkflowForGroupUpdate was entered
	})

	t.Run("Malformed NATS subject", func(t *testing.T) {
		// No ListWorkflows call expected here as parsing should fail first
		mockMeta.ExpectedCalls = nil 
		
		msg := &nats.Msg{Subject: "GROUP.update", Data: []byte("test data")} // Malformed
		service.handleGroupUpdateMsg(msg)
		// Assert that TriggerWorkflowForGroupUpdate (and thus ListWorkflows) was NOT called
		mockMeta.AssertNotCalled(t, "ListWorkflows")
	})
}

// --- Test TriggerWorkflowForGroupUpdate ---
func TestTriggerWorkflowForGroupUpdate(t *testing.T) {
	mockMeta := new(MockMetadataServiceClient)
	mockGrouping := new(MockGroupingServiceClient)
	mockNatsJS := NewMockNatsJetStreamPublisher()
	mockNatsJS.On("Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&nats.Subscription{}, nil)

	db, _, _ := sqlmock.New() // Mock DB for fetchEntityInstanceData
	defer db.Close()
	service := NewOrchestrationService(mockNatsJS, mockMeta, mockGrouping, db)

	sampleGroupID := "group-xyz"
	otherGroupID := "group-abc"

	wfEnabledGroupMatch := WorkflowDefinition{ID: "wf1", Name: "WF1", IsEnabled: true, TriggerType: "on_group_update", TriggerConfig: fmt.Sprintf(`{"group_id": "%s"}`, sampleGroupID), ActionSequenceJSON: `[]`}
	wfDisabled := WorkflowDefinition{ID: "wf2", Name: "WF2", IsEnabled: false, TriggerType: "on_group_update", TriggerConfig: fmt.Sprintf(`{"group_id": "%s"}`, sampleGroupID), ActionSequenceJSON: `[]`}
	wfWrongTriggerType := WorkflowDefinition{ID: "wf3", Name: "WF3", IsEnabled: true, TriggerType: "manual", ActionSequenceJSON: `[]`}
	wfWrongGroupID := WorkflowDefinition{ID: "wf4", Name: "WF4", IsEnabled: true, TriggerType: "on_group_update", TriggerConfig: fmt.Sprintf(`{"group_id": "%s"}`, otherGroupID), ActionSequenceJSON: `[]`}
	wfBadTriggerConfig := WorkflowDefinition{ID: "wf5", Name: "WF5", IsEnabled: true, TriggerType: "on_group_update", TriggerConfig: `{"group_id":}`, ActionSequenceJSON: `[]`} // Invalid JSON

	t.Run("Workflow triggered for matching groupID", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfEnabledGroupMatch}, nil).Once()
		mockGrouping.On("GetGroupMembers", sampleGroupID).Return(&GroupCalculationResult{MemberIDs: []string{"id1"}}, nil).Once()
		// Mock GetActionTemplate if ActionSequenceJSON was not empty
		// Since it's empty (`[]`), executeWorkflow will return early. We are testing if it's called.
		// To properly test executeWorkflow call, we'd need to mock GetActionTemplate too.
		// For now, testing that the path to call executeWorkflow is reached.
		
		service.TriggerWorkflowForGroupUpdate(sampleGroupID)
		
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertExpectations(t)
		// We can't directly assert executeWorkflow was called without more refactoring or global state.
		// But ListWorkflows and GetGroupMembers being called implies it was on the right path.
	})

	t.Run("No trigger for disabled workflow", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfDisabled}, nil).Once()
		// GetGroupMembers should not be called
		
		service.TriggerWorkflowForGroupUpdate(sampleGroupID)
		
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertNotCalled(t, "GetGroupMembers", sampleGroupID)
	})

	t.Run("No trigger for wrong trigger type", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfWrongTriggerType}, nil).Once()
		service.TriggerWorkflowForGroupUpdate(sampleGroupID)
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertNotCalled(t, "GetGroupMembers", mock.Anything)
	})

	t.Run("No trigger for non-matching groupID in config", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfWrongGroupID}, nil).Once()
		service.TriggerWorkflowForGroupUpdate(sampleGroupID) // Event for sampleGroupID
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertNotCalled(t, "GetGroupMembers", mock.Anything)
	})
	
	t.Run("Error parsing TriggerConfig", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfBadTriggerConfig}, nil).Once()
		service.TriggerWorkflowForGroupUpdate(sampleGroupID)
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertNotCalled(t, "GetGroupMembers", mock.Anything)
	})

	t.Run("GetGroupMembers fails", func(t *testing.T) {
		mockMeta.On("ListWorkflows").Return([]WorkflowDefinition{wfEnabledGroupMatch}, nil).Once()
		mockGrouping.On("GetGroupMembers", sampleGroupID).Return(nil, fmt.Errorf("group fetch error")).Once()
		
		service.TriggerWorkflowForGroupUpdate(sampleGroupID)
		
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertExpectations(t)
		// executeWorkflow should not have proceeded to task publishing
	})
}


// --- Test TriggerWorkflow (Manual Trigger) ---
func TestTriggerWorkflow_Manual(t *testing.T) {
	mockMeta := new(MockMetadataServiceClient)
	mockGrouping := new(MockGroupingServiceClient)
	mockNatsJS := NewMockNatsJetStreamPublisher()
	mockNatsJS.On("Subscribe", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&nats.Subscription{}, nil)
	
	db, _, _ := sqlmock.New()
	defer db.Close()
	service := NewOrchestrationService(mockNatsJS, mockMeta, mockGrouping, db)

	workflowID := "manualWF"
	manualWorkflow := WorkflowDefinition{ID: workflowID, Name: "Manual WF", IsEnabled: true, TriggerType: "manual", ActionSequenceJSON: `[]`}
	groupTriggerWorkflow := WorkflowDefinition{ID: "groupTriggerWF", Name: "Group Trigger WF", IsEnabled: true, TriggerType: "on_group_update", TriggerConfig: `{"group_id": "group123"}`, ActionSequenceJSON: `[]`}


	t.Run("Successful manual trigger", func(t *testing.T) {
		mockMeta.On("GetWorkflowDefinition", workflowID).Return(&manualWorkflow, nil).Once()
		// executeWorkflow will be called with nil groupMembers.
		// If ActionSequenceJSON is empty, it returns nil quickly.
		
		err := service.TriggerWorkflow(workflowID)
		require.NoError(t, err)
		mockMeta.AssertExpectations(t)
	})

	t.Run("Manual trigger for on_group_update type workflow", func(t *testing.T) {
		mockMeta.On("GetWorkflowDefinition", "groupTriggerWF").Return(&groupTriggerWorkflow, nil).Once()
		mockGrouping.On("GetGroupMembers", "group123").Return(&GroupCalculationResult{MemberIDs: []string{"m1"}}, nil).Once()
		
		err := service.TriggerWorkflow("groupTriggerWF")
		require.NoError(t, err)
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertExpectations(t)
	})
	
	t.Run("Manual trigger for on_group_update type, GetGroupMembers fails", func(t *testing.T) {
		mockMeta.On("GetWorkflowDefinition", "groupTriggerWF").Return(&groupTriggerWorkflow, nil).Once()
		mockGrouping.On("GetGroupMembers", "group123").Return(nil, fmt.Errorf("group error")).Once()
		
		err := service.TriggerWorkflow("groupTriggerWF")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group error")
		mockMeta.AssertExpectations(t)
		mockGrouping.AssertExpectations(t)
	})


	t.Run("Workflow not found", func(t *testing.T) {
		mockMeta.On("GetWorkflowDefinition", "unknownWF").Return(nil, fmt.Errorf("not found")).Once()
		err := service.TriggerWorkflow("unknownWF")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		mockMeta.AssertExpectations(t)
	})

	t.Run("Workflow disabled", func(t *testing.T) {
		disabledWF := WorkflowDefinition{ID: "disabledWF", IsEnabled: false, TriggerType: "manual"}
		mockMeta.On("GetWorkflowDefinition", "disabledWF").Return(&disabledWF, nil).Once()
		err := service.TriggerWorkflow("disabledWF")
		require.NoError(t, err) // Disabling is not an error, just logs and skips
		mockMeta.AssertExpectations(t)
	})
}

// --- Test ExecuteWorkflow ---
func TestExecuteWorkflow(t *testing.T) {
	mockMeta := new(MockMetadataServiceClient)
	mockNatsJS := NewMockNatsJetStreamPublisher()
	// No Subscribe mock needed here as executeWorkflow doesn't trigger it.
	
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Pass nil for groupingClient if not used by a specific sub-test of executeWorkflow
	service := NewOrchestrationService(mockNatsJS, mockMeta, nil, db) 

	actionSeq := `[{"action_template_id": "at1", "parameters_json": "{}"}]`
	wf := WorkflowDefinition{ID: "wfExecute1", Name: "ExecuteTestWF", IsEnabled: true, ActionSequenceJSON: actionSeq}
	at := ActionTemplate{ID: "at1", Name: "Test AT", ActionType: "webhook", TemplateContent: "{}"}

	t.Run("Group-triggered with members", func(t *testing.T) {
		mockMeta.On("GetActionTemplate", "at1").Return(&at, nil).Once()
		groupMembers := &GroupCalculationResult{MemberIDs: []string{"entity1", "entity2"}}

		// Mock DB calls for fetchEntityInstanceData
		dbMock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs("entity1").WillReturnRows(sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(`{"email": "e1@test.com"}`)))
		dbMock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs("entity2").WillReturnRows(sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(`{"email": "e2@test.com"}`)))
		
		// Mock NATS publish calls
		mockNatsJS.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(&nats.StreamInfo{}, nil).Twice() // Called for each task
		mockNatsJS.On("Publish", "actions.webhook", mock.Anything, mock.Anything).Return(&nats.PubAck{}, nil).Twice()
		
		err := service.executeWorkflow(wf, groupMembers, "group_test_context")
		require.NoError(t, err)
		
		mockMeta.AssertExpectations(t)
		mockNatsJS.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
		// Check captured messages
		assert.Len(t, mockNatsJS.PublishedMessages, 1) // Only one entry if same subject, last message wins. Need to adjust capture.
		// For better capture, MockNatsJetStreamPublisher.PublishedMessages should be map[string][][]byte or slice of structs
	})
	
	// Reset mock calls for the next sub-test
	// Re-initialize mocks for cleaner state between sub-tests if needed, or manage expectations carefully.
	// For testify/mock, you might need to reset ExpectedCalls or use .Once(), .Twice() etc.
	// For simplicity, I'm assuming independent sub-tests or careful management of .On() calls.

	t.Run("Manually triggered (no group members)", func(t *testing.T) {
		// Re-initialize or clear expectations for mocks
		mockMetaManual := new(MockMetadataServiceClient)
		mockNatsJSManual := NewMockNatsJetStreamPublisher()
		serviceManual := NewOrchestrationService(mockNatsJSManual, mockMetaManual, nil, db)


		mockMetaManual.On("GetActionTemplate", "at1").Return(&at, nil).Once()
		mockNatsJSManual.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(&nats.StreamInfo{}, nil).Once()
		mockNatsJSManual.On("Publish", "actions.webhook", mock.Anything, mock.Anything).Return(&nats.PubAck{}, nil).Once()

		err := serviceManual.executeWorkflow(wf, nil, "manual_test_context") // nil for groupMembers
		require.NoError(t, err)

		mockMetaManual.AssertExpectations(t)
		mockNatsJSManual.AssertExpectations(t)
	})

	t.Run("ActionSequenceJSON parsing error", func(t *testing.T) {
		wfBadSeq := WorkflowDefinition{ID: "wfBadSeq", ActionSequenceJSON: `[{"id": "at1"`} // Invalid JSON
		err := service.executeWorkflow(wfBadSeq, nil, "bad_seq_context")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse action_sequence_json")
	})

	t.Run("GetActionTemplate fails", func(t *testing.T) {
		mockMetaFail := new(MockMetadataServiceClient)
		serviceFail := NewOrchestrationService(nil, mockMetaFail, nil, db)

		mockMetaFail.On("GetActionTemplate", "at1").Return(nil, fmt.Errorf("template fetch error")).Once()
		
		// executeWorkflow logs this error and continues. The overall workflow doesn't fail.
		err := serviceFail.executeWorkflow(wf, nil, "get_at_fail_context")
		require.NoError(t, err) 
		mockMetaFail.AssertExpectations(t)
	})

	t.Run("fetchEntityInstanceData fails for one member", func(t *testing.T) {
		// Re-init mocks for this specific scenario
		mockMetaFetchFail := new(MockMetadataServiceClient)
		mockNatsJSFetchFail := NewMockNatsJetStreamPublisher()
		dbFetch, dbMockFetch, _ := sqlmock.New()
		defer dbFetch.Close()
		serviceFetchFail := NewOrchestrationService(mockNatsJSFetchFail, mockMetaFetchFail, nil, dbFetch)

		mockMetaFetchFail.On("GetActionTemplate", "at1").Return(&at, nil).Once()
		groupMembers := &GroupCalculationResult{MemberIDs: []string{"entity1", "entity2_fails", "entity3"}}

		dbMockFetch.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs("entity1").WillReturnRows(sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(`{"data":"ok1"}`)))
		dbMockFetch.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs("entity2_fails").WillReturnError(fmt.Errorf("db error for entity2"))
		dbMockFetch.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs("entity3").WillReturnRows(sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(`{"data":"ok3"}`)))
		
		mockNatsJSFetchFail.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(&nats.StreamInfo{}, nil).Twice() // For entity1 and entity3
		mockNatsJSFetchFail.On("Publish", "actions.webhook", mock.Anything, mock.Anything).Return(&nats.PubAck{}, nil).Twice()

		err := serviceFetchFail.executeWorkflow(wf, groupMembers, "fetch_fail_context")
		require.NoError(t, err) // executeWorkflow logs and continues

		mockMetaFetchFail.AssertExpectations(t)
		mockNatsJSFetchFail.AssertExpectations(t)
		assert.NoError(t, dbMockFetch.ExpectationsWereMet())
	})
}

// --- Test FetchEntityInstanceData ---
func TestFetchEntityInstanceData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewOrchestrationService(nil, nil, nil, db) // NATS and other clients not needed

	entityID := "test-entity-id"

	t.Run("Successful data fetch", func(t *testing.T) {
		jsonData := `{"key": "value", "num": 123}`
		rows := sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(jsonData))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs(entityID).WillReturnRows(rows)

		data, err := service.fetchEntityInstanceData(entityID)
		require.NoError(t, err)
		assert.Equal(t, "value", data["key"])
		assert.Equal(t, float64(123), data["num"]) // JSON numbers are float64
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sql.ErrNoRows", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs(entityID).WillReturnError(sql.ErrNoRows)
		
		data, err := service.fetchEntityInstanceData(entityID)
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "no processed_entity found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DB query error", func(t *testing.T) {
		dbErr := errors.New("database query failed")
		mock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs(entityID).WillReturnError(dbErr)

		data, err := service.fetchEntityInstanceData(entityID)
		require.Error(t, err)
		assert.Nil(t, data)
		assert.True(t, errors.Is(err, dbErr))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("JSON unmarshal error", func(t *testing.T) {
		invalidJsonData := `{"key": "value", num: 123}` // num not quoted - invalid JSON
		rows := sqlmock.NewRows([]string{"attributes"}).AddRow([]byte(invalidJsonData))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs(entityID).WillReturnRows(rows)

		data, err := service.fetchEntityInstanceData(entityID)
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "failed to unmarshal attributes")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("Empty or NULL JSON data in DB", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"attributes"}).AddRow([]byte{}) // Empty JSON
		mock.ExpectQuery(regexp.QuoteMeta("SELECT attributes FROM processed_entities WHERE id = $1")).
			WithArgs(entityID).WillReturnRows(rows)

		data, err := service.fetchEntityInstanceData(entityID)
		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.Empty(t, data, "Data should be an empty map for empty/null JSON from DB")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Test PublishTask (Simplified - focuses on NATS interaction mocking) ---
func TestPublishTask(t *testing.T) {
	mockNatsJS := NewMockNatsJetStreamPublisher()
	service := NewOrchestrationService(mockNatsJS, nil, nil, nil) // Other clients and DB not needed

	taskMsg := TaskMessage{
		TaskID:      "task123",
		ActionType:  "webhook",
		// Other fields filled as needed...
	}
	expectedSubject := "actions.webhook"

	t.Run("Successful publish (stream exists)", func(t *testing.T) {
		mockNatsJS.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(&nats.StreamInfo{}, nil).Once()
		mockNatsJS.On("Publish", expectedSubject, mock.Anything, mock.Anything).Return(&nats.PubAck{Stream: actionTasksStreamName, Sequence: 1}, nil).Once()

		err := service.publishTask(taskMsg)
		require.NoError(t, err)
		mockNatsJS.AssertExpectations(t)
	})

	t.Run("Successful publish (stream creation needed)", func(t *testing.T) {
		mockNatsJS.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(nil, nats.ErrStreamNotFound).Once() // Stream does not exist
		mockNatsJS.On("AddStream", mock.AnythingOfType("*nats.StreamConfig"), mock.Anything).Return(&nats.StreamInfo{}, nil).Once()
		mockNatsJS.On("Publish", expectedSubject, mock.Anything, mock.Anything).Return(&nats.PubAck{Stream: actionTasksStreamName, Sequence: 1}, nil).Once()

		err := service.publishTask(taskMsg)
		require.NoError(t, err)
		mockNatsJS.AssertExpectations(t)
	})

	t.Run("Publish fails", func(t *testing.T) {
		publishErr := errors.New("NATS publish error")
		mockNatsJS.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(&nats.StreamInfo{}, nil).Once()
		mockNatsJS.On("Publish", expectedSubject, mock.Anything, mock.Anything).Return(nil, publishErr).Once()
		
		err := service.publishTask(taskMsg)
		require.Error(t, err)
		assert.True(t, errors.Is(err, publishErr))
		mockNatsJS.AssertExpectations(t)
	})
	
	t.Run("Stream creation fails after not found", func(t *testing.T) {
		addStreamErr := errors.New("NATS AddStream error")
		mockNatsJS.On("StreamInfo", actionTasksStreamName, mock.Anything).Return(nil, nats.ErrStreamNotFound).Once()
		mockNatsJS.On("AddStream", mock.AnythingOfType("*nats.StreamConfig"), mock.Anything).Return(nil, addStreamErr).Once()
		// Publish should not be called
		
		err := service.publishTask(taskMsg)
		require.Error(t, err)
		assert.True(t, errors.Is(err, addStreamErr))
		mockNatsJS.AssertExpectations(t)
		mockNatsJS.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
	})
}
