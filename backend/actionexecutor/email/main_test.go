package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to capture log output for verification
func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(os.Stderr) // Reset to default
	return buf.String()
}

// Helper to create a nats.Msg for testing
func newTestNatsMsg(t *testing.T, data interface{}) *nats.Msg {
	t.Helper()
	jsonData, err := json.Marshal(data)
	require.NoError(t, err)
	return &nats.Msg{Data: jsonData, Subject: "actions.email"}
}

func TestApplyTemplate(t *testing.T) {
	t.Run("Valid template with entity and params", func(t *testing.T) {
		data := TemplateData{
			Entity: map[string]interface{}{"name": "John Doe", "id": "user123"},
			Params: map[string]interface{}{"code": "XYZ123", "amount": 100},
		}
		templateStr := "Hello {{ .Entity.name }} ({{ .Entity.id }}), your code is {{ .Params.code }} and amount is {{ .Params.amount }}."
		expected := "Hello John Doe (user123), your code is XYZ123 and amount is 100."
		result, err := applyTemplate("testValid", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Missing entity data", func(t *testing.T) {
		data := TemplateData{
			Entity: map[string]interface{}{"id": "user123"}, // Name is missing
			Params: map[string]interface{}{"code": "XYZ123"},
		}
		// Go's default behavior for missing keys is to render an empty string or <no value>
		// For this test, we'll check for an empty string where the name would be.
		templateStr := "Hello {{ .Entity.name }}, your code is {{ .Params.code }}."
		expected := "Hello <no value>, your code is XYZ123." // or "Hello , your code is XYZ123." if Option("missingkey=zero")
		result, err := applyTemplate("testMissingEntity", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Missing params data", func(t *testing.T) {
		data := TemplateData{
			Entity: map[string]interface{}{"name": "John Doe"},
			Params: map[string]interface{}{}, // Code is missing
		}
		templateStr := "Hello {{ .Entity.name }}, your code is {{ .Params.code }}."
		expected := "Hello John Doe, your code is <no value>."
		result, err := applyTemplate("testMissingParams", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Invalid template string", func(t *testing.T) {
		data := TemplateData{}
		templateStr := "Hello {{ .Entity.name" // Unclosed brace
		_, err := applyTemplate("testInvalidTemplate", templateStr, data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing template")
	})

	t.Run("Empty template string", func(t *testing.T) {
		data := TemplateData{}
		templateStr := ""
		result, err := applyTemplate("testEmptyTemplate", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("Template with complex data (nested map)", func(t *testing.T) {
		data := TemplateData{
			Entity: map[string]interface{}{
				"details": map[string]interface{}{"city": "New York", "country": "USA"},
			},
			Params: map[string]interface{}{"order_id": "order789"},
		}
		templateStr := "Order {{ .Params.order_id }} for city: {{ .Entity.details.city }}."
		expected := "Order order789 for city: New York."
		result, err := applyTemplate("testComplexData", templateStr, data)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestHandleEmailTask(t *testing.T) {
	// Ensure config is loaded, then override for tests
	originalSMTPHost := smtpHost
	originalSMTPPort := smtpPort
	originalDefaultFromEmail := defaultFromEmail
	
	loadConfig() // Load defaults first

	t.Cleanup(func() {
		smtpHost = originalSMTPHost
		smtpPort = originalSMTPPort
		defaultFromEmail = originalDefaultFromEmail
		log.SetOutput(os.Stderr) // Ensure log output is reset
	})

	// --- Test Cases ---
	t.Run("Valid Task in Simulation Mode", func(t *testing.T) {
		smtpHost = "" // Trigger simulation mode
		smtpPort = ""
		defaultFromEmail = "test@example.com"

		emailContent := EmailTemplateContent{
			SubjectTemplate:      "Welcome {{ .Entity.name }}!",
			BodyTemplate:         "Your code is {{ .Params.code }}. Entity ID: {{ .Entity.id }}",
			BodyType:             "text/plain",
			ToRecipientsTemplate: "{{ .Entity.email }}",
			FromEmail:            "override_from@example.com",
		}
		templateContentJSON, _ := json.Marshal(emailContent)

		task := TaskMessage{
			TaskID:           "task123",
			WorkflowID:       "wf456",
			ActionType:       "EMAIL",
			TemplateContent:  string(templateContentJSON),
			ActionParams:     map[string]interface{}{"code": "WELCOME100"},
			EntityInstanceID: "entity789",
			EntityInstance:   map[string]interface{}{"name": "Alice", "id": "entity789", "email": "alice@example.com"},
		}
		msg := newTestNatsMsg(t, task)

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})

		assert.Contains(t, logOutput, "SIMULATING EMAIL SEND for TaskID task123")
		assert.Contains(t, logOutput, "To: alice@example.com")
		assert.Contains(t, logOutput, "From: override_from@example.com") // FromEmail in template overrides default
		assert.Contains(t, logOutput, "Subject: Welcome Alice!")
		assert.Contains(t, logOutput, "BodyType: text/plain; charset=UTF-8") // Default body type
		assert.Contains(t, logOutput, "Your code is WELCOME100. Entity ID: entity789")
	})
	
	t.Run("Valid Task in Simulation Mode with HTML body", func(t *testing.T) {
		smtpHost = "" 
		smtpPort = ""
		defaultFromEmail = "test@example.com"

		emailContent := EmailTemplateContent{
			SubjectTemplate:      "HTML Email Test",
			BodyTemplate:         "<h1>Hello {{ .Entity.name }}</h1><p>Your code: {{ .Params.code }}</p>",
			BodyType:             "text/html",
			ToRecipientsTemplate: "bob@example.com",
		}
		templateContentJSON, _ := json.Marshal(emailContent)
		task := TaskMessage{
			TaskID: "taskHTML", ActionType: "EMAIL", TemplateContent: string(templateContentJSON),
			ActionParams: map[string]interface{}{"code": "HTML123"},
			EntityInstance: map[string]interface{}{"name": "Bob"},
		}
		msg := newTestNatsMsg(t, task)
		logOutput := captureOutput(func() { handleEmailTask(msg) })

		assert.Contains(t, logOutput, "SIMULATING EMAIL SEND for TaskID taskHTML")
		assert.Contains(t, logOutput, "Subject: HTML Email Test")
		assert.Contains(t, logOutput, "BodyType: text/html; charset=UTF-8")
		assert.Contains(t, logOutput, "<h1>Hello Bob</h1><p>Your code: HTML123</p>")
	})


	t.Run("Invalid TaskMessage JSON", func(t *testing.T) {
		rawMsgData := []byte(`{"task_id": "broken", "action_type": "EMAIL", "template_content": "not json"`) // Malformed JSON
		msg := &nats.Msg{Data: rawMsgData, Subject: "actions.email"}

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})
		assert.Contains(t, logOutput, "Error unmarshalling TaskMessage")
	})

	t.Run("Invalid TemplateContent JSON", func(t *testing.T) {
		task := TaskMessage{
			TaskID:          "taskInvalidContent",
			ActionType:      "EMAIL",
			TemplateContent: `{"subjectTemplate": "Valid Subject", "bodyTemplate": "Valid Body", "toRecipientsTemplate": "test@example.com"`, // Malformed
		}
		msg := newTestNatsMsg(t, task)

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})
		assert.Contains(t, logOutput, "TaskID taskInvalidContent: Error unmarshalling TemplateContent")
	})

	t.Run("Recipient Template Resolves to Empty", func(t *testing.T) {
		smtpHost = "" // Simulation mode
		smtpPort = ""
		emailContent := EmailTemplateContent{
			SubjectTemplate:      "Test Subject",
			BodyTemplate:         "Test Body",
			ToRecipientsTemplate: "{{ .Entity.missing_email_field }}", // This will resolve to empty or <no value>
		}
		templateContentJSON, _ := json.Marshal(emailContent)
		task := TaskMessage{
			TaskID:          "taskEmptyTo",
			ActionType:      "EMAIL",
			TemplateContent: string(templateContentJSON),
			EntityInstance:  map[string]interface{}{"name": "Test User"}, // No email field
		}
		msg := newTestNatsMsg(t, task)

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})
		assert.Contains(t, logOutput, "TaskID taskEmptyTo: No valid 'To' recipients after template processing. Skipping email.")
		assert.NotContains(t, logOutput, "SIMULATING EMAIL SEND for TaskID taskEmptyTo") // Should not attempt to send
	})
	
	t.Run("Recipient Template is empty string", func(t *testing.T) {
		smtpHost = "" 
		smtpPort = ""
		emailContent := EmailTemplateContent{
			SubjectTemplate:      "Test Subject",
			BodyTemplate:         "Test Body",
			ToRecipientsTemplate: "", // Empty string directly
		}
		templateContentJSON, _ := json.Marshal(emailContent)
		task := TaskMessage{
			TaskID: "taskEmptyStringTo", ActionType: "EMAIL", TemplateContent: string(templateContentJSON),
		}
		msg := newTestNatsMsg(t, task)
		logOutput := captureOutput(func() { handleEmailTask(msg) })
		assert.Contains(t, logOutput, "TaskID taskEmptyStringTo: No valid 'To' recipients after template processing. Skipping email.")
	})


	t.Run("Template Execution Error (Invalid Field in Template)", func(t *testing.T) {
		smtpHost = "" // Simulation mode
		smtpPort = ""
		emailContent := EmailTemplateContent{
			SubjectTemplate:      "Subject: {{ .Entity.name }}",
			BodyTemplate:         "Body: {{ .Params.non_existent_param }}", // This field won't exist
			ToRecipientsTemplate: "test@example.com",
		}
		templateContentJSON, _ := json.Marshal(emailContent)
		task := TaskMessage{
			TaskID:          "taskMissingField",
			ActionType:      "EMAIL",
			TemplateContent: string(templateContentJSON),
			EntityInstance:  map[string]interface{}{"name": "Test User"},
			ActionParams:    map[string]interface{}{"actual_param": "value"},
		}
		msg := newTestNatsMsg(t, task)

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})
		// Default Go template behavior renders <no value> for missing keys
		assert.Contains(t, logOutput, "SIMULATING EMAIL SEND for TaskID taskMissingField")
		assert.Contains(t, logOutput, "Subject: Test User")
		assert.Contains(t, logOutput, "Body: <no value>")
	})
	
	t.Run("Task with non-EMAIL action type", func(t *testing.T) {
		task := TaskMessage{
			TaskID:     "taskWrongType",
			ActionType: "SLACK", // Not EMAIL
		}
		msg := newTestNatsMsg(t, task)
		logOutput := captureOutput(func() { handleEmailTask(msg) })
		assert.Contains(t, logOutput, "TaskID taskWrongType: ActionType is 'SLACK', not 'EMAIL'. Skipping.")
		assert.NotContains(t, logOutput, "SIMULATING EMAIL SEND")
	})

	t.Run("ParseRecipientList handles various formats", func(t *testing.T) {
		assert.Equal(t, []string{"a@b.com"}, parseRecipientList("a@b.com"))
		assert.Equal(t, []string{"a@b.com", "c@d.com"}, parseRecipientList("a@b.com,c@d.com"))
		assert.Equal(t, []string{"a@b.com", "c@d.com"}, parseRecipientList("a@b.com;c@d.com"))
		assert.Equal(t, []string{"a@b.com", "c@d.com", "e@f.com"}, parseRecipientList("a@b.com, c@d.com; e@f.com"))
		assert.Equal(t, []string{}, parseRecipientList(""))
		assert.Equal(t, []string{}, parseRecipientList("   "))
		assert.Equal(t, []string{"a@b.com"}, parseRecipientList("  a@b.com  "))
	})
	
	t.Run("CC and BCC recipients are handled", func(t *testing.T) {
		smtpHost = "" // Simulation mode
		smtpPort = ""
		defaultFromEmail = "test@example.com"

		emailContent := EmailTemplateContent{
			SubjectTemplate:       "Test CC/BCC",
			BodyTemplate:          "Testing all recipients.",
			BodyType:              "text/plain",
			ToRecipientsTemplate:  "to@example.com",
			CcRecipientsTemplate:  "cc1@example.com, {{ .Entity.cc_email }}",
			BccRecipientsTemplate: "bcc@example.com",
			FromEmail:             "sender@example.com",
		}
		templateContentJSON, _ := json.Marshal(emailContent)

		task := TaskMessage{
			TaskID:           "taskCCBCC",
			ActionType:       "EMAIL",
			TemplateContent:  string(templateContentJSON),
			EntityInstance:   map[string]interface{}{"cc_email": "cc_entity@example.com"},
		}
		msg := newTestNatsMsg(t, task)

		logOutput := captureOutput(func() {
			handleEmailTask(msg)
		})

		assert.Contains(t, logOutput, "SIMULATING EMAIL SEND for TaskID taskCCBCC")
		assert.Contains(t, logOutput, "To: to@example.com")
		assert.Contains(t, logOutput, "Cc: cc1@example.com, cc_entity@example.com")
		assert.Contains(t, logOutput, "Bcc: bcc@example.com")
		assert.Contains(t, logOutput, "From: sender@example.com")
		assert.Contains(t, logOutput, "Subject: Test CC/BCC")
		assert.Contains(t, logOutput, "Body: Testing all recipients.")
	})
}

```
