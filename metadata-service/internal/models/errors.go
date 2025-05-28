package models

// APIError represents a standardized error response format for the API.
// @Description APIError represents a standardized error response format, including an application-specific error code, a human-readable message, and optional details.
type APIError struct {
	Code    string      `json:"code"`              // Application-specific error code (e.g., "NOT_FOUND", "VALIDATION_ERROR")
	Message string      `json:"message"`           // Human-readable message describing the error
	Details interface{} `json:"details,omitempty"` // Optional field for additional error details (e.g., validation failures per field)
}

// Predefined application-specific error codes
const (
	// Generic Errors
	ErrorCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	ErrorCodeUnknown             = "UNKNOWN_ERROR"
	ErrorCodeRequestTimeout      = "REQUEST_TIMEOUT"
	ErrorCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"

	// Input Validation & Data Errors
	ErrorCodeValidation          = "VALIDATION_ERROR"       // General validation failure
	ErrorCodeInvalidJSON         = "INVALID_JSON"         // Malformed JSON payload
	ErrorCodeInvalidIDFormat     = "INVALID_ID_FORMAT"    // e.g., UUID format error
	ErrorCodeMissingRequiredField= "MISSING_REQUIRED_FIELD" 
	ErrorCodeValueOutOfRange     = "VALUE_OUT_OF_RANGE"
	ErrorCodeInvalidEnumValue    = "INVALID_ENUM_VALUE"   // For fields like DataType, RelationshipType

	// Resource Specific Errors
	ErrorCodeNotFound            = "NOT_FOUND"            // Generic resource not found
	ErrorCodeEntityNotFound      = "ENTITY_NOT_FOUND"
	ErrorCodeAttributeNotFound   = "ATTRIBUTE_NOT_FOUND"
	ErrorCodeRelationshipNotFound= "RELATIONSHIP_NOT_FOUND"
	ErrorCodeForeignKeyNotFound  = "FOREIGN_KEY_NOT_FOUND" // When a referenced entity (e.g. parent entity) does not exist

	// Business Logic / State Errors
	ErrorCodeConflict            = "CONFLICT_ERROR"       // e.g., unique constraint violation, circular dependency
	ErrorCodeDuplicateName       = "DUPLICATE_NAME"
	ErrorCodeCircularDependency  = "CIRCULAR_DEPENDENCY"  // Specifically for relationships

	// Add more codes as needed
)
