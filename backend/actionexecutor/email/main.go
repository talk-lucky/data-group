package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"runtime"
	"strings"
	"text/template" // For simple text templating

	"github.com/nats-io/nats.go"
)

// EmailTemplateContent defines the structure for email template details.
type EmailTemplateContent struct {
	SubjectTemplate       string `json:"subjectTemplate"`
	BodyTemplate          string `json:"bodyTemplate"`
	BodyType              string `json:"bodyType"` // "text/plain" or "text/html"
	ToRecipientsTemplate  string `json:"toRecipientsTemplate"`
	CcRecipientsTemplate  string `json:"ccRecipientsTemplate,omitempty"`
	BccRecipientsTemplate string `json:"bccRecipientsTemplate,omitempty"`
	FromEmail             string `json:"fromEmail,omitempty"` // Can be overridden by global config
}

// TaskMessage mirrors the structure from orchestration service.
type TaskMessage struct {
	TaskID            string                 `json:"task_id"`
	WorkflowID        string                 `json:"workflow_id"`
	ActionTemplateID  string                 `json:"action_template_id"`
	ActionType        string                 `json:"action_type"` // Should be "EMAIL" for this executor
	TemplateContent   string                 `json:"template_content"`
	ActionParams      map[string]interface{} `json:"action_params"`
	EntityInstanceID  string                 `json:"entity_instance_id,omitempty"`
	EntityInstance    map[string]interface{} `json:"entity_instance,omitempty"`
}

// TemplateData is used for passing data to Go templates.
type TemplateData struct {
	Entity map[string]interface{}
	Params map[string]interface{}
}

var (
	natsURL          string
	smtpHost         string
	smtpPort         string
	smtpUser         string
	smtpPass         string
	defaultFromEmail string
)

func loadConfig() {
	natsURL = os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222" // Default for local dev
		log.Printf("NATS_URL not set, using default: %s", natsURL)
	}
	smtpHost = os.Getenv("SMTP_HOST")
	smtpPort = os.Getenv("SMTP_PORT")
	smtpUser = os.Getenv("SMTP_USER")
	smtpPass = os.Getenv("SMTP_PASS")
	defaultFromEmail = os.Getenv("DEFAULT_FROM_EMAIL")

	if smtpHost == "" || smtpPort == "" {
		log.Println("Warning: SMTP_HOST or SMTP_PORT not configured. Email sending will be simulated.")
	}
	if defaultFromEmail == "" {
		defaultFromEmail = "noreply@example.com"
		log.Printf("DEFAULT_FROM_EMAIL not set, using default: %s", defaultFromEmail)
	}
}

func main() {
	loadConfig()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS server: %s", natsURL)

	// Subscribe to the email action subject
	// Using a queue group "email-executor-group" for load balancing if multiple instances are run
	_, err = nc.QueueSubscribe("actions.email", "email-executor-group", handleEmailTask)
	if err != nil {
		log.Fatalf("Error subscribing to NATS subject 'actions.email': %v", err)
	}
	log.Println("Subscribed to NATS subject 'actions.email' with queue group 'email-executor-group'")

	// Keep the service running
	log.Println("Email Executor Service is running. Waiting for messages...")
	runtime.Goexit() // Keeps main goroutine alive until all other goroutines exit. More robust than select{}.
}

func handleEmailTask(msg *nats.Msg) {
	log.Printf("Received task on subject '%s'", msg.Subject)

	var task TaskMessage
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("Error unmarshalling TaskMessage for TaskID %s: %v. Message data: %s", task.TaskID, err, string(msg.Data))
		// Consider sending to a dead-letter queue or logging more permanently
		return
	}
	log.Printf("Processing TaskID: %s, WorkflowID: %s, ActionType: %s", task.TaskID, task.WorkflowID, task.ActionType)

	if strings.ToUpper(task.ActionType) != "EMAIL" {
		log.Printf("TaskID %s: ActionType is '%s', not 'EMAIL'. Skipping.", task.TaskID, task.ActionType)
		return
	}

	var emailContent EmailTemplateContent
	if err := json.Unmarshal([]byte(task.TemplateContent), &emailContent); err != nil {
		log.Printf("TaskID %s: Error unmarshalling TemplateContent: %v. Content: %s", task.TaskID, err, task.TemplateContent)
		return
	}

	templateData := TemplateData{
		Entity: task.EntityInstance,
		Params: task.ActionParams,
	}

	// Apply templates
	subject, err := applyTemplate("subject", emailContent.SubjectTemplate, templateData)
	if err != nil {
		log.Printf("TaskID %s: Error applying subject template: %v", task.TaskID, err)
		return
	}
	body, err := applyTemplate("body", emailContent.BodyTemplate, templateData)
	if err != nil {
		log.Printf("TaskID %s: Error applying body template: %v", task.TaskID, err)
		return
	}
	toRecipientsStr, err := applyTemplate("to", emailContent.ToRecipientsTemplate, templateData)
	if err != nil {
		log.Printf("TaskID %s: Error applying ToRecipients template: %v", task.TaskID, err)
		return
	}
	
	var ccRecipientsStr, bccRecipientsStr string
	if emailContent.CcRecipientsTemplate != "" {
		ccRecipientsStr, err = applyTemplate("cc", emailContent.CcRecipientsTemplate, templateData)
		if err != nil { log.Printf("TaskID %s: Error applying CcRecipients template: %v. Proceeding without CC.", task.TaskID, err); ccRecipientsStr = "" }
	}
	if emailContent.BccRecipientsTemplate != "" {
		bccRecipientsStr, err = applyTemplate("bcc", emailContent.BccRecipientsTemplate, templateData)
		if err != nil { log.Printf("TaskID %s: Error applying BccRecipients template: %v. Proceeding without BCC.", task.TaskID, err); bccRecipientsStr = "" }
	}


	fromEmail := defaultFromEmail
	if emailContent.FromEmail != "" {
		fromEmail = emailContent.FromEmail
	}
	
	toRecipients := parseRecipientList(toRecipientsStr)
	if len(toRecipients) == 0 {
		log.Printf("TaskID %s: No valid 'To' recipients after template processing. Skipping email.", task.TaskID)
		return
	}
	ccRecipients := parseRecipientList(ccRecipientsStr)
	bccRecipients := parseRecipientList(bccRecipientsStr)
	
	allRecipients := append(toRecipients, ccRecipients...)
	allRecipients = append(allRecipients, bccRecipients...)


	// Construct email message
	// For simplicity, using a basic structure. Could be enhanced with MIME types, etc.
	var emailMessage strings.Builder
	emailMessage.WriteString(fmt.Sprintf("From: %s\r\n", fromEmail))
	emailMessage.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toRecipients, ", ")))
	if len(ccRecipients) > 0 {
		emailMessage.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccRecipients, ", ")))
	}
	emailMessage.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	
	contentType := "text/plain; charset=UTF-8"
	if strings.ToLower(emailContent.BodyType) == "text/html" {
		contentType = "text/html; charset=UTF-8"
	}
	emailMessage.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	emailMessage.WriteString("\r\n") // Empty line before body
	emailMessage.WriteString(body)


	if smtpHost == "" || smtpPort == "" {
		log.Printf("SIMULATING EMAIL SEND for TaskID %s:\n---BEGIN EMAIL---\nTo: %s\nCc: %s\nBcc: %s\nFrom: %s\nSubject: %s\nBodyType: %s\n\n%s\n---END EMAIL---",
			task.TaskID, strings.Join(toRecipients, ", "), strings.Join(ccRecipients, ", "), strings.Join(bccRecipients, ", "), fromEmail, subject, contentType, body)
		return
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	smtpAddr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	err = smtp.SendMail(smtpAddr, auth, fromEmail, allRecipients, []byte(emailMessage.String()))
	if err != nil {
		log.Printf("TaskID %s: Error sending email via SMTP: %v", task.TaskID, err)
		// Implement retry logic or dead-lettering if necessary
		return
	}

	log.Printf("TaskID %s: Email successfully sent to %s (CC: %s, BCC: %s) via %s", task.TaskID, strings.Join(toRecipients, ", "), strings.Join(ccRecipients, ", "), strings.Join(bccRecipients, ", "), smtpAddr)
}

func applyTemplate(templateName string, templateStr string, data TemplateData) (string, error) {
	if templateStr == "" {
		return "", nil // Handle empty template string gracefully
	}
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("error parsing template '%s': %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template '%s': %w", templateName, err)
	}
	return buf.String(), nil
}

// parseRecipientList parses a comma or semicolon separated list of emails.
func parseRecipientList(recipientsStr string) []string {
    if recipientsStr == "" {
        return []string{}
    }
    // Replace semicolons with commas, then split by comma
    normalizedStr := strings.ReplaceAll(recipientsStr, ";", ",")
    recipients := strings.Split(normalizedStr, ",")
    
    var validRecipients []string
    for _, r := range recipients {
        trimmed := strings.TrimSpace(r)
        if trimmed != "" { // Basic check, could add more validation
            validRecipients = append(validRecipients, trimmed)
        }
    }
    return validRecipients
}

```
