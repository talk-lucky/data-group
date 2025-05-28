package handlers

import (
	"github.com/gin-gonic/gin"
	"metadata-service/internal/models" // Import the models package to use APIError
)

// RespondWithError sends a standardized JSON error response.
// It logs the error internally for server-side tracking.
func RespondWithError(c *gin.Context, httpStatus int, appErrorCode string, message string, details interface{}) {
	// Log the error for server-side observability.
	// In a real application, you might use a more structured logger.
	// log.Printf("Error response: HTTPStatus=%d, AppErrorCode=%s, Message=%s, Details=%v", httpStatus, appErrorCode, message, details)

	errResp := models.APIError{
		Code:    appErrorCode,
		Message: message,
		Details: details,
	}
	c.JSON(httpStatus, errResp)
}

// RespondWithSuccess sends a standardized JSON success response.
// This can be used for GET (single resource), PUT, PATCH, DELETE (if returning content).
// For POST (Create), typically a 201 with the created resource is directly returned.
// For list responses, a different structure (PaginatedResponse) will be used.
func RespondWithSuccess(c *gin.Context, httpStatus int, data interface{}) {
	if data != nil {
		c.JSON(httpStatus, data)
	} else {
		// For 204 No Content, send no body
		c.Status(httpStatus)
	}
}
