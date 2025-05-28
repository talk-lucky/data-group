package handlers

import (
	"errors" // Already present
	"net/http"
	"strings" // Will be removed or its usage reduced

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq" // Added for pq.Error
	"gorm.io/gorm"

	"log"       // Added for logging validated params
	"strconv"   // Added for string to int conversion
	"strings"   // Already present, used for ToLower

	"metadata-service/internal/database"
	"metadata-service/internal/models"
)

const (
	DefaultLimit          = 10
	MaxLimit              = 100
	DefaultSortOrder      = "asc"
	DefaultEntitySortBy   = "created_at" // Corresponds to GORM's auto-managed field
)

var AllowedEntitySortByFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
}

// CreateEntity godoc
// @Summary Create a new entity definition
// @Description Create a new entity definition with the provided name and description.
// @Tags entities
// @Accept  json
// @Produce  json
// @Param   entity_definition  body   models.CreateEntityRequest   true  "Entity Definition to create"
// @Success 201 {object} models.EntityDefinition "Successfully created entity definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error - see 'code' in response for specifics like VALIDATION_ERROR)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate name - see 'code' in response for specifics like DUPLICATE_NAME)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities [post]
func CreateEntity(c *gin.Context) {
	var req models.CreateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	db := database.GetDB()
	entity := models.EntityDefinition{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
	}

	if err := db.Create(&entity).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // PostgreSQL error code for unique_violation
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "Entity definition with this name already exists.", gin.H{"name": entity.Name})
				return
			}
		}
		// Fallback for other errors or if not a pq.Error
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to create entity definition.", nil)
		return
	}

	// Use RespondWithSuccess for consistency, though c.JSON is fine for 201.
	RespondWithSuccess(c, http.StatusCreated, entity)
}

// ListEntities godoc
// @Summary List all entity definitions
// @Description Get a list of all entity definitions.
// @Tags entities
// @Produce  json
// @Success 200 {array} models.EntityDefinition "Successfully retrieved list of entity definitions"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities [get]
func ListEntities(c *gin.Context) {
	// Get and validate pagination parameters
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid limit parameter: not a number.", gin.H{"limit": limitStr})
		return
	}
	if limit <= 0 {
		limit = DefaultLimit
	} else if limit > MaxLimit {
		limit = MaxLimit
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid offset parameter: not a number.", gin.H{"offset": offsetStr})
		return
	}
	if offset < 0 {
		offset = 0
	}

	// Get and validate sorting parameters
	sortBy := c.DefaultQuery("sort_by", DefaultEntitySortBy)
	if _, isValid := AllowedEntitySortByFields[sortBy]; !isValid {
		allowedFields := make([]string, 0, len(AllowedEntitySortByFields))
		for k := range AllowedEntitySortByFields {
			allowedFields = append(allowedFields, k)
		}
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_by field for entities.", gin.H{"field": sortBy, "allowed": allowedFields})
		return
	}
	// Assuming API sort_by field name matches DB column name directly for now
	// dbSortByColumn := sortBy 

	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", DefaultSortOrder))
	if sortOrder != "asc" && sortOrder != "desc" {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_order value. Must be 'asc' or 'desc'.", gin.H{"value": c.Query("sort_order")})
		return
	}

	log.Printf("ListEntities: Validated params: limit=%d, offset=%d, sortBy=%s, sortOrder=%s", limit, offset, sortBy, sortOrder)

	// Existing DB logic (to be modified in next step)
	db := database.GetDB()
	var entities []models.EntityDefinition
	// This DB query will be updated in Step 8 to use limit, offset, sortBy, sortOrder
	if err := db.Find(&entities).Error; err != nil { 
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to list entity definitions", nil)
		return
	}
	// This response will change in Step 9 to use PaginatedResponse
	RespondWithSuccess(c, http.StatusOK, entities)
}

// GetEntity godoc
// @Summary Get a specific entity definition by ID
// @Description Get detailed information about a specific entity definition using its UUID.
// @Tags entities
// @Produce  json
// @Param   id     path   string     true  "Entity Definition ID (UUID)"
// @Success 200 {object} models.EntityDefinition "Successfully retrieved entity definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities/{id} [get]
func GetEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid ID format for entity ID", gin.H{"id": idStr})
		return
	}

	db := database.GetDB()
	var entity models.EntityDefinition
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Entity definition not found", gin.H{"id": entityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to get entity definition", nil)
		}
		return
	}
	RespondWithSuccess(c, http.StatusOK, entity)
}

// UpdateEntity godoc
// @Summary Update an existing entity definition
// @Description Update an existing entity definition's name and/or description.
// @Tags entities
// @Accept  json
// @Produce  json
// @Param   id     path   string     true  "Entity Definition ID (UUID)"
// @Param   entity_definition  body   models.UpdateEntityRequest   true  "Entity Definition fields to update"
// @Success 200 {object} models.EntityDefinition "Successfully updated entity definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error, invalid ID format - see 'code' in response for specifics like VALIDATION_ERROR, INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate name - see 'code' in response for specifics like DUPLICATE_NAME)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities/{id} [put]
func UpdateEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid ID format for entity ID", gin.H{"id": idStr})
		return
	}

	var req models.UpdateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	db := database.GetDB()
	var entity models.EntityDefinition
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Entity definition not found", gin.H{"id": entityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to find entity definition for update", nil)
		}
		return
	}

	// Update fields if provided in the request
	if req.Name != nil {
		entity.Name = *req.Name
	}
	if req.Description != nil {
		entity.Description = *req.Description
	}

	if err := db.Save(&entity).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "Entity definition with this name already exists.", gin.H{"name": entity.Name})
				return
			}
		}
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to update entity definition.", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, entity)
}

// DeleteEntity godoc
// @Summary Delete an entity definition
// @Description Delete an entity definition by its UUID.
// @Tags entities
// @Param   id     path   string     true  "Entity Definition ID (UUID)"
// @Success 204 "Successfully deleted entity definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities/{id} [delete]
func DeleteEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid ID format for entity ID", gin.H{"id": idStr})
		return
	}

	db := database.GetDB()
	// Check if entity exists before trying to delete
	// Using a separate First call to provide a clear "NotFound" vs other errors.
	var entityCheck models.EntityDefinition
	if err := db.First(&entityCheck, "id = ?", entityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Entity definition not found", gin.H{"id": entityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to find entity definition for deletion", nil)
		}
		return
	}

	// Attempt to delete the entity
	if err := db.Delete(&models.EntityDefinition{}, "id = ?", entityID).Error; err != nil {
		// This could be due to foreign key constraints or other issues.
		// For now, a general internal server error. More specific checks could be added.
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to delete entity definition", nil)
		return
	}
	RespondWithSuccess(c, http.StatusNoContent, nil)
}
