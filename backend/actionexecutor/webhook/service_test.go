package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go" // Only for nats.Msg in TestHandleNATSMsg
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assuming models.go defines TaskMessage, WebhookTemplate, TemplateData in package main.
// If not, these would need to be defined here or imported.

// --- Tests for renderTemplate ---
func TestRenderTemplate(t *testing.T) {
	t.Run("Successful Rendering", func(t *testing.T) {
		templateStr := "Hello {{.EntityInstance.name}}, your code is {{.ActionParams.code}}."
		data := TemplateData{
			EntityInstance: map[string]interface{}{"name": "World"},
			ActionParams:   map[string]interface{}{"code": 123},
		}
		rendered, err := renderTemplate("testRender", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, "Hello World, your code is 123.", rendered)
	})

	t.Run("Missing EntityInstance Key", func(t *testing.T) {
		templateStr := "Name: {{.EntityInstance.name}}, City: {{.EntityInstance.city}}"
		data := TemplateData{
			EntityInstance: map[string]interface{}{"name": "Alice"}, // city is missing
			ActionParams:   map[string]interface{}{},
		}
		rendered, err := renderTemplate("testMissingEntityKey", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, "Name: Alice, City: <no value>", rendered)
	})

	t.Run("Missing ActionParams Key", func(t *testing.T) {
		templateStr := "Param1: {{.ActionParams.param1}}, Param2: {{.ActionParams.param2}}"
		data := TemplateData{
			EntityInstance: map[string]interface{}{},
			ActionParams:   map[string]interface{}{"param1": "value1"}, // param2 is missing
		}
		rendered, err := renderTemplate("testMissingActionParamKey", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, "Param1: value1, Param2: <no value>", rendered)
	})

	t.Run("Template Syntax Error", func(t *testing.T) {
		templateStr := "Hello {{.EntityInstance.name" // Missing closing braces
		data := TemplateData{}
		_, err := renderTemplate("testSyntaxError", templateStr, data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template parsing error")
	})

	t.Run("Nil Data Maps", func(t *testing.T) {
		templateStr := "Entity: {{.EntityInstance.id}}, Param: {{.ActionParams.code}}"
		data := TemplateData{EntityInstance: nil, ActionParams: nil}
		rendered, err := renderTemplate("testNilMaps", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, "Entity: <no value>, Param: <no value>", rendered)
	})
}

// --- Tests for processWebhookTask ---
func TestProcessWebhookTask(t *testing.T) {
	// Suppress log output from the service during tests
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	t.Run("Successful POST with JSON payload", func(t *testing.T) {
		var receivedMethod, receivedPath string
		var receivedHeaders http.Header
		var receivedBody map[string]interface{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			receivedHeaders = r.Header
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"ok"}`)
		}))
		defer server.Close()

		service := NewWebhookExecutorService() // Uses default http.Client

		templateContentJSON := fmt.Sprintf(`{
            "url_template": "%s/submit",
            "method": "POST",
            "headers_template": {"X-Custom-Header": "Value-{{.ActionParams.header_val}}", "Content-Type": "application/json"},
            "payload_template": "{\"id\": \"{{.EntityInstance.id}}\", \"name\": \"{{.ActionParams.customName}}\", \"fixed_val\": 100}"
        }`, server.URL)

		task := TaskMessage{
			TemplateContent: templateContentJSON,
			EntityInstance:  map[string]interface{}{"id": "entity123"},
			ActionParams:    map[string]interface{}{"customName": "Test User", "header_val": "Dynamic"},
		}

		err := service.processWebhookTask(task)
		require.NoError(t, err)

		assert.Equal(t, "POST", receivedMethod)
		assert.Equal(t, "/submit", receivedPath)
		assert.Equal(t, "Value-Dynamic", receivedHeaders.Get("X-Custom-Header"))
		assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))

		expectedBody := map[string]interface{}{"id": "entity123", "name": "Test User", "fixed_val": json.Number("100")}
		assert.Equal(t, expectedBody["id"], receivedBody["id"])
		assert.Equal(t, expectedBody["name"], receivedBody["name"])
		// JSON numbers are unmarshaled as float64 or json.Number. Here, it's json.Number.
		assert.Equal(t, expectedBody["fixed_val"], receivedBody["fixed_val"].(json.Number))
	})

	t.Run("Successful GET request with templated URL and headers", func(t *testing.T) {
		var receivedMethod, receivedPath string
		var receivedHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			receivedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		service := NewWebhookExecutorService()

		templateContentJSON := fmt.Sprintf(`{
            "url_template": "%s/resource/{{.EntityInstance.id}}?param={{.ActionParams.query_param}}",
            "method": "GET",
            "headers_template": {"Authorization": "Bearer {{.ActionParams.token}}"}
        }`, server.URL)
		task := TaskMessage{
			TemplateContent: templateContentJSON,
			EntityInstance:  map[string]interface{}{"id": "res456"},
			ActionParams:    map[string]interface{}{"query_param": "searchVal", "token": "secret123"},
		}

		err := service.processWebhookTask(task)
		require.NoError(t, err)

		assert.Equal(t, "GET", receivedMethod)
		assert.Equal(t, "/resource/res456", receivedPath) // Query params not part of r.URL.Path
		assert.Equal(t, "Bearer secret123", receivedHeaders.Get("Authorization"))
		// Check query param if needed: assert.Equal(t, "searchVal", server.URL.Query().Get("param")) - needs access to original request on server side
	})

	t.Run("Template Rendering Failure - Invalid TemplateContent JSON", func(t *testing.T) {
		service := NewWebhookExecutorService()
		task := TaskMessage{TemplateContent: `{"url_template": "{{.URL}}", "method": "GET"`} // Malformed JSON
		err := service.processWebhookTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal webhook template")
	})

	t.Run("Template Rendering Failure - Invalid URL Go Template", func(t *testing.T) {
		service := NewWebhookExecutorService()
		task := TaskMessage{TemplateContent: `{"url_template": "{{.EntityInstance.id", "method": "GET"}`}
		err := service.processWebhookTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to render URL from template")
	})
	
	t.Run("Template Rendering Failure - Invalid Payload Go Template", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); }))
		defer server.Close();
		service := NewWebhookExecutorService()
		task := TaskMessage{TemplateContent: fmt.Sprintf(`{"url_template": "%s", "method": "POST", "payload_template": "{\"key\": \"{{.Val\"}"}`, server.URL)}
		err := service.processWebhookTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to render payload from template")
	})


	t.Run("HTTP Request Failure - Target Server Error (500)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		service := NewWebhookExecutorService()

		templateContentJSON := fmt.Sprintf(`{"url_template": "%s", "method": "GET"}`, server.URL)
		task := TaskMessage{TemplateContent: templateContentJSON}

		// Current logic logs non-2xx status but does not return an error.
		err := service.processWebhookTask(task)
		require.NoError(t, err) 
		// To verify the log, we'd need a more complex logging setup or capture stdout.
		// For this test, we assume the log happens and the function doesn't error out.
	})

	t.Run("HTTP Request Failure - Network Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		targetURL := server.URL // Get URL before closing
		server.Close()          // Close server to simulate network error

		service := NewWebhookExecutorService()
		templateContentJSON := fmt.Sprintf(`{"url_template": "%s", "method": "GET"}`, targetURL)
		task := TaskMessage{TemplateContent: templateContentJSON}

		err := service.processWebhookTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute webhook request")
	})
	
	t.Run("Default Content-Type for POST/PUT with payload", func(t *testing.T) {
		var receivedContentType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		service := NewWebhookExecutorService()

		templateContentJSON := fmt.Sprintf(`{
            "url_template": "%s",
            "method": "POST",
            "payload_template": "{\"key\":\"value\"}" 
        }`, server.URL) // No "headers_template" or Content-Type
		task := TaskMessage{TemplateContent: templateContentJSON}

		err := service.processWebhookTask(task)
		require.NoError(t, err)
		assert.Equal(t, "application/json; charset=utf-8", receivedContentType)
	})

	t.Run("No default Content-Type for GET", func(t *testing.T) {
		var receivedContentType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		service := NewWebhookExecutorService()
		templateContentJSON := fmt.Sprintf(`{"url_template": "%s", "method": "GET"}`, server.URL)
		task := TaskMessage{TemplateContent: templateContentJSON}

		err := service.processWebhookTask(task)
		require.NoError(t, err)
		assert.Empty(t, receivedContentType)
	})
}

// --- Tests for NATS Message Handling (Simplified) ---
// This requires handleNATSMsg to be an exported method or a standalone function for direct testing.
// If handleNATSMsg is not exported, we test the logic it would call (processWebhookTask)
// by simulating the unmarshaling step.

func TestHandleNATSMsg_ProcessWebhookTaskIntegration(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	// This is a conceptual test of what handleNATSMsg would do.
	// It assumes handleNATSMsg unmarshals TaskMessage and calls processWebhookTask.
	// We directly test that integration path.
	service := NewWebhookExecutorService() // Real service, real http client

	t.Run("Successful message processing", func(t *testing.T) {
		var webhookCalled bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			webhookCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		templateContent := fmt.Sprintf(`{"url_template": "%s", "method": "GET"}`, server.URL)
		task := TaskMessage{
			TaskID:          "nats-task-1",
			TemplateContent: templateContent,
		}
		taskData, err := json.Marshal(task)
		require.NoError(t, err)

		// Simulate NATS message (actual nats.Msg not fully mocked here)
		// In a real test for handleNATSMsg, you'd pass a *nats.Msg
		// For this integration test of the logic *within* handleNATSMsg:
		var unmarshaledTask TaskMessage
		err = json.Unmarshal(taskData, &unmarshaledTask)
		require.NoError(t, err)
		
		err = service.processWebhookTask(unmarshaledTask)
		require.NoError(t, err)
		assert.True(t, webhookCalled, "Webhook should have been called")
	})

	t.Run("Malformed JSON in NATS message data", func(t *testing.T) {
		malformedTaskData := []byte(`{"task_id": "broken"`) // Invalid JSON

		var unmarshaledTask TaskMessage
		err := json.Unmarshal(malformedTaskData, &unmarshaledTask)
		// This error occurs before processWebhookTask would be called by handleNATSMsg
		require.Error(t, err) 
		// If handleNATSMsg were tested directly, we'd check if msg.Term() was called.
		// Here, we just show that the unmarshaling, which handleNATSMsg does first, would fail.
	})
}
