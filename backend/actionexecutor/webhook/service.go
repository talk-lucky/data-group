package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/nats-io/nats.go"
)

// WebhookExecutorService handles consuming and processing webhook tasks from NATS.
type WebhookExecutorService struct {
	natsJS     nats.JetStreamContext
	httpClient *http.Client
}

// NewWebhookExecutorService creates a new WebhookExecutorService.
func NewWebhookExecutorService(js nats.JetStreamContext) *WebhookExecutorService {
	return &WebhookExecutorService{
		natsJS:     js,
		httpClient: &http.Client{Timeout: 30 * time.Second}, // Configurable timeout
	}
}

// StartConsuming subscribes to NATS subjects and processes messages.
func (s *WebhookExecutorService) StartConsuming() error {
	subject := "actions.webhook"
	streamName := "ACTIONS" // Must match the stream name used by Orchestration service
	consumerName := "webhookExecutor"

	log.Printf("WebhookExecutorService starting to consume from subject '%s', stream '%s', consumer '%s'", subject, streamName, consumerName)

	// Ensure stream exists (idempotent)
	_, err := s.natsJS.StreamInfo(streamName)
	if err != nil {
		log.Printf("Stream %s not found, attempting to create it for subject %s...", streamName, "actions.>")
		_, createErr := s.natsJS.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"actions.>"}, // Capture all action types
			Storage:  nats.FileStorage,
		})
		if createErr != nil {
			return fmt.Errorf("failed to create NATS stream %s: %w", streamName, createErr)
		}
		log.Printf("Successfully created NATS stream %s", streamName)
	}


	// Durable Pull Consumer
	// Pull consumer is generally safer for at-least-once processing if message handlers can take time or fail.
	// For simplicity in this example, a Push consumer with Ack is used.
	// If using Pull:
	// sub, err := s.natsJS.PullSubscribe(subject, consumerName, nats.AckWait(30*time.Second))
	// if err != nil {
	//  return fmt.Errorf("failed to pull subscribe to subject %s: %w", subject, err)
	// }
	// for {
	//  msgs, err := sub.Fetch(1, nats.MaxWait(10*time.Second)) // Fetch 1 message, wait up to 10s
	//  if err != nil && err != nats.ErrTimeout {
	//      log.Printf("Error fetching message: %v", err)
	//      time.Sleep(5 * time.Second) // Wait before retrying
	//      continue
	//  }
	//  if err == nats.ErrTimeout {
	//      continue
	//  }
	//  if len(msgs) == 0 {
	//      continue
	//  }
	//  msg := msgs[0]
	//  // process logic... then msg.Ack()
	// }

	// Using a Push consumer for this example
	_, err = s.natsJS.Subscribe(subject, func(msg *nats.Msg) {
		log.Printf("Received task on subject %s (seq: %d)", msg.Subject, msg.Sequence)
		var task TaskMessage
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			log.Printf("Error unmarshalling TaskMessage (TaskID: potentially unknown, Seq: %d): %v. Message will be terminated.", msg.Sequence, err)
			// Terminate message if it's malformed and cannot be processed.
			// For other types of errors, consider Nak() to allow redelivery.
			if err := msg.Term(); err != nil {
				log.Printf("Error terminating malformed message (Seq: %d): %v", msg.Sequence, err)
			}
			return
		}

		log.Printf("Processing TaskID: %s, ActionType: %s, EntityInstanceID: %s", task.TaskID, task.ActionType, task.EntityInstanceID)

		if err := s.processWebhookTask(task); err != nil {
			log.Printf("Error processing webhook task (TaskID: %s): %v", task.TaskID, err)
			// Nack the message for potential redelivery if it's a retryable error
			// For now, we just log and Ack to prevent infinite loops on bad tasks.
			// Consider more sophisticated error handling (e.g., dead-letter queue).
			// if nackErr := msg.Nak(); nackErr != nil {
			//  log.Printf("Error Nacking message (TaskID: %s): %v", task.TaskID, nackErr)
			// }
		}

		// Acknowledge the message after processing (or attempting to process)
		if err := msg.Ack(); err != nil {
			log.Printf("Error Acknowledging message (TaskID: %s): %v", task.TaskID, err)
		}
		log.Printf("Finished processing and Acked TaskID: %s", task.TaskID)

	}, nats.Durable(consumerName), nats.AckWait(60*time.Second), nats.ManualAck()) // ManualAck and AckWait

	if err != nil {
		return fmt.Errorf("failed to subscribe to subject %s with durable consumer %s: %w", subject, consumerName, err)
	}

	log.Printf("Subscribed to subject '%s' with durable consumer '%s'. Waiting for tasks...", subject, consumerName)
	// Keep the main goroutine alive (e.g., select{} or http.ListenAndServe if it also had an API)
	select {} // Block forever
}

// processWebhookTask handles the execution of a single webhook task.
func (s *WebhookExecutorService) processWebhookTask(task TaskMessage) error {
	var webhookTmpl WebhookTemplate
	if err := json.Unmarshal([]byte(task.TemplateContent), &webhookTmpl); err != nil {
		return fmt.Errorf("failed to unmarshal TemplateContent for TaskID %s: %w", task.TaskID, err)
	}

	// Prepare data for templating
	templateCtx := TemplateData{
		EntityInstance: task.EntityInstance,
		ActionParams:   task.ActionParams,
	}

	// Render URL
	renderedURL, err := s.renderTemplate("url", webhookTmpl.URLTemplate, templateCtx)
	if err != nil {
		return fmt.Errorf("failed to render URL for TaskID %s: %w", task.TaskID, err)
	}

	// Render Headers
	renderedHeaders := make(http.Header)
	for key, valueTmpl := range webhookTmpl.HeadersTemplate {
		renderedValue, err := s.renderTemplate(fmt.Sprintf("header-%s", key), valueTmpl, templateCtx)
		if err != nil {
			log.Printf("Warning: Failed to render header '%s' for TaskID %s: %v. Skipping header.", key, task.TaskID, err)
			continue
		}
		renderedHeaders.Set(key, renderedValue)
	}
	// Default content type for POST/PUT if not specified and payload exists
	if (strings.ToUpper(webhookTmpl.Method) == "POST" || strings.ToUpper(webhookTmpl.Method) == "PUT") &&
		webhookTmpl.PayloadTemplate != "" && renderedHeaders.Get("Content-Type") == "" {
		renderedHeaders.Set("Content-Type", "application/json")
	}


	// Render Payload (if applicable)
	var reqBodyReader *bytes.Reader
	var renderedPayloadStr string // For logging
	if (strings.ToUpper(webhookTmpl.Method) == "POST" || strings.ToUpper(webhookTmpl.Method) == "PUT") && webhookTmpl.PayloadTemplate != "" {
		renderedPayload, err := s.renderTemplate("payload", webhookTmpl.PayloadTemplate, templateCtx)
		if err != nil {
			return fmt.Errorf("failed to render payload for TaskID %s: %w", task.TaskID, err)
		}
		reqBodyReader = bytes.NewReader([]byte(renderedPayload))
		renderedPayloadStr = renderedPayload // Store for logging
	} else {
		reqBodyReader = bytes.NewReader([]byte{}) // Empty body for GET/DELETE or if no template
	}

	// Make the HTTP request
	req, err := http.NewRequest(strings.ToUpper(webhookTmpl.Method), renderedURL, reqBodyReader)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for TaskID %s: %w", task.TaskID, err)
	}
	req.Header = renderedHeaders

	log.Printf("Executing Webhook TaskID %s: %s %s", task.TaskID, req.Method, req.URL)
	if renderedPayloadStr != "" {
		log.Printf(" TaskID %s - Payload: %s", task.TaskID, s.truncateString(renderedPayloadStr, 256))
	}
	if len(req.Header) > 0 {
		log.Printf(" TaskID %s - Headers: %v", task.TaskID, req.Header)
	}


	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed for TaskID %s to %s: %w", task.TaskID, renderedURL, err)
	}
	defer resp.Body.Close()

	respBodyBytes, _ :=ReadAll(resp.Body) // ReadAll helper to handle potential errors
	respBodyStr := string(respBodyBytes)

	log.Printf("Webhook TaskID %s completed. URL: %s, Status: %s, Response Body: %s",
		task.TaskID, renderedURL, resp.Status, s.truncateString(respBodyStr, 256))

	if resp.StatusCode >= 400 {
		// Log as an error but don't necessarily return error to NATS unless it's retryable
		// For now, we consider the HTTP call "processed" even if it's a 4xx/5xx.
		log.Printf("Error: Webhook TaskID %s received HTTP %s. Full Body: %s", task.TaskID, resp.Status, respBodyStr)
		// return fmt.Errorf("webhook request for TaskID %s returned HTTP %s", task.TaskID, resp.Status)
	}

	return nil
}

// renderTemplate executes a Go template with the given data.
func (s *WebhookExecutorService) renderTemplate(templateName, templateStr string, data TemplateData) (string, error) {
	if templateStr == "" {
		return "", nil
	}
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", templateName, err)
	}

	var rendered bytes.Buffer
	// To access nested map fields like {{.EntityInstance.field_name}} or {{.ActionParams.param_name}}
	// the `data` must be structured correctly.
	// The TemplateData struct already provides this structure.
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", templateName, err)
	}
	return rendered.String(), nil
}

// ReadAll is a helper to read all bytes from an io.Reader, useful for http.Response.Body.
func ReadAll(r *http.Response) ([]byte, error) {
    var buf bytes.Buffer
    if _, err := buf.ReadFrom(r.Body); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

// truncateString is a helper for logging.
func (s *WebhookExecutorService) truncateString(str string, num int) string {
    bn := len(str)
    if bn > num {
        if num > 3 {
            return str[0:num-3] + "..."
        }
        return str[0:num]
    }
    return str
}
