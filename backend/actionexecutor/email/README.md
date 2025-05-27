# Email Executor Service

## Overview

The Email Executor service is responsible for sending emails as part of an automated workflow. It listens for tasks on the NATS subject `actions.email` and processes them based on the provided action template and entity data.

## Configuration

The service is configured using the following environment variables:

| Variable             | Description                                                                 | Default                  |
|----------------------|-----------------------------------------------------------------------------|--------------------------|
| `NATS_URL`           | URL of the NATS server.                                                     | `nats://nats:4222`       |
| `SMTP_HOST`          | Hostname or IP address of the SMTP server.                                  | (Must be configured)     |
| `SMTP_PORT`          | Port number for the SMTP server.                                            | `587`                    |
| `SMTP_USER`          | Username for SMTP authentication (if required).                             | `""` (empty)             |
| `SMTP_PASS`          | Password for SMTP authentication (if required).                             | `""` (empty)             |
| `DEFAULT_FROM_EMAIL` | Default "From" email address if not specified in the action template.     | `noreply@example.com`    |

**Note:** If `SMTP_HOST` is not configured, the service will run in simulation mode, logging the email content instead of sending it.

## Action Template for Email (`ActionType="email"`)

To use this service, you need to create an `ActionTemplate` in the metadata service with `ActionType` set to `"email"`. The `TemplateContent` field of this `ActionTemplate` should be a JSON string defining the email's structure and content.

### `TemplateContent` JSON Structure:

| Field                   | Type   | Required | Description                                                                                                |
|-------------------------|--------|----------|------------------------------------------------------------------------------------------------------------|
| `subjectTemplate`       | string | Yes      | Go template string for the email subject.                                                                  |
| `bodyTemplate`          | string | Yes      | Go template string for the email body.                                                                     |
| `bodyType`              | string | No       | Content type of the body. Either `"text/plain"` (default) or `"text/html"`.                                |
| `toRecipientsTemplate`  | string | Yes      | Go template string that resolves to one or more email addresses (comma-separated if multiple).             |
| `ccRecipientsTemplate`  | string | No       | Go template string for CC recipients.                                                                      |
| `bccRecipientsTemplate` | string | No       | Go template string for BCC recipients.                                                                     |
| `fromEmail`             | string | No       | Overrides the `DEFAULT_FROM_EMAIL` for this specific template.                                             |

### Example `TemplateContent`:

```json
{
    "subjectTemplate": "Notification for {{ .Entity.name }} (ID: {{ .Entity.id }})",
    "bodyTemplate": "Hello {{ .Entity.contact_person }},\n\nThis is an automated notification regarding {{ .Entity.name }}.\n\nWorkflow Parameter 'custom_message': {{ .Params.custom_message }}\n\nThank you.",
    "bodyType": "text/plain",
    "toRecipientsTemplate": "{{ .Entity.email_address }}",
    "ccRecipientsTemplate": "{{ .Params.cc_list }}",
    "fromEmail": "alerts@example.com"
}
```

## Templating

The `subjectTemplate`, `bodyTemplate`, and recipient template fields (`toRecipientsTemplate`, `ccRecipientsTemplate`, `bccRecipientsTemplate`) are processed using Go's `text/template` package.

You can access two main data contexts within your templates:

*   **`{{ .Entity }}`**: This map contains the attributes of the entity instance that triggered the action. For example, if your entity has an attribute `customer_name`, you can access it using `{{ .Entity.customer_name }}`.
*   **`{{ .Params }}`**: This map contains parameters passed from the specific workflow action step's `ParametersJSON`. For example, if `ParametersJSON` is `{"discount_code": "SAVE10"}`, you can access it using `{{ .Params.discount_code }}`.

## Example Workflow Action Step

Here's how an action step within a workflow's `ActionSequenceJSON` might use an email action template:

```json
{
    "action_template_id": "your_email_action_template_uuid", 
    "parameters_json": "{ \"custom_message\": \"Your recent order has been processed.\", \"cc_list\": \"manager@example.com\" }"
}
```
In this example, `parameters_json` provides values that can be accessed in the email templates via `{{ .Params.custom_message }}` and `{{ .Params.cc_list }}`.
