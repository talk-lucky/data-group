package handlers

import (
	"errors" // Already present
	"net/http"
	"strings" // Usage will be reduced/removed

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq" // Added for pq.Error
	"gorm.io/gorm"

	"log"     // Added for logging
	"strconv" // Added for string to int conversion

	"metadata-service/internal/database"
	"metadata-service/internal/models" // Already present
)

// Constants for pagination and sorting for Relationships
const (
	DefaultRelationshipLimit       = 10
	MaxRelationshipLimit           = 100
	DefaultRelationshipSortOrder   = "asc"
	DefaultRelationshipSortBy      = "created_at"
)

// AllowedRelationshipSortByFields defines the fields by which a list of relationships can be sorted.
var AllowedRelationshipSortByFields = map[string]bool{
	"name":              true,
	"relationship_type": true,
	"created_at":        true,
	"updated_at":        true,
}

// CreateRelationship godoc
// @Summary Create a new entity relationship
// @Description Creates a new entity relationship definition between two existing entities.
// @Tags relationships
// @Accept  json
// @Produce  json
// @Param   relationship  body   models.CreateEntityRelationshipRequest  true  "Entity Relationship Creation Request"
// @Success 201 {object} models.EntityRelationshipDefinition "Successfully created entity relationship"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error, invalid ID format, invalid enum value - see 'code' in response for specifics like VALIDATION_ERROR, INVALID_ID_FORMAT, INVALID_ENUM_VALUE)"
// @Failure 404 {object} models.APIError "Not Found (e.g., source/target entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate name, circular dependency - see 'code' in response for specifics like DUPLICATE_NAME, CIRCULAR_DEPENDENCY)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /relationships [post]
func CreateRelationship(c *gin.Context) {
	db := database.GetDB()
	var req models.CreateEntityRelationshipRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	normalizedRelationshipType := strings.ToUpper(req.RelationshipType)
	if !models.ValidRelationshipTypes[normalizedRelationshipType] {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidEnumValue, "Invalid relationship_type. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY", gin.H{"relationship_type": req.RelationshipType})
		return
	}

	sourceEntityID, err := uuid.Parse(req.SourceEntityID)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid source_entity_id format", gin.H{"id": req.SourceEntityID, "error": err.Error()})
		return
	}

	targetEntityID, err := uuid.Parse(req.TargetEntityID)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid target_entity_id format", gin.H{"id": req.TargetEntityID, "error": err.Error()})
		return
	}

	// Validate that SourceEntityID exists
	var sourceEntity models.EntityDefinition
	if err := db.First(&sourceEntity, "id = ?", sourceEntityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Source entity not found", gin.H{"entity_id": sourceEntityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Database error checking source entity", nil)
		}
		return
	}

	// Validate that TargetEntityID exists
	var targetEntity models.EntityDefinition
	if err := db.First(&targetEntity, "id = ?", targetEntityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Target entity not found", gin.H{"entity_id": targetEntityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Database error checking target entity", nil)
		}
		return
	}

	// Check for circular dependency
	var inverseRelationshipCount int64
	if err := db.Model(&models.EntityRelationshipDefinition{}).
		Where("source_entity_id = ? AND target_entity_id = ?", targetEntityID, sourceEntityID).
		Count(&inverseRelationshipCount).Error; err != nil {
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Database error during circular dependency check", nil)
		return
	}

	if inverseRelationshipCount > 0 {
		details := gin.H{
			"proposed_source_id": sourceEntityID,
			"proposed_target_id": targetEntityID,
			"conflicting_relationship_direction": gin.H{
				"source_id": targetEntityID,
				"target_id": sourceEntityID,
			},
		}
		RespondWithError(c, http.StatusConflict, models.ErrorCodeCircularDependency, "A circular dependency would be created. An inverse relationship already exists.", details)
		return
	}

	relationship := models.EntityRelationshipDefinition{
		ID:               uuid.New(),
		Name:             req.Name,
		SourceEntityID:   sourceEntityID,
		TargetEntityID:   targetEntityID,
		RelationshipType: normalizedRelationshipType,
		Description:      req.Description,
	}

	if err := db.Create(&relationship).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // PostgreSQL error code for unique_violation
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "A relationship with the same name, source entity, and target entity already exists.", gin.H{"name": req.Name, "source_id": req.SourceEntityID, "target_id": req.TargetEntityID})
				return
			}
		}
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to create entity relationship.", nil)
		return
	}
	RespondWithSuccess(c, http.StatusCreated, relationship)
}

// ListRelationships godoc
// @Summary List all entity relationships
// @Description Retrieves a list of all entity relationships, optionally filtered by source entity, target entity, or relationship type.
// @Tags relationships
// @Produce  json
// @Param   source_entity_id query  string  false  "Filter by source entity ID (UUID)"
// @Param   target_entity_id query  string  false  "Filter by target entity ID (UUID)"
// @Param   type query  string  false  "Filter by relationship type (e.g., ONE_TO_ONE)"
// @Success 200 {array} models.EntityRelationshipDefinition "Successfully retrieved entity relationships"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid query parameter format for ID or type - see 'code' in response for specifics like INVALID_ID_FORMAT, INVALID_ENUM_VALUE)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /relationships [get]
func ListRelationships(c *gin.Context) {
	db := database.GetDB()
	var relationships []models.EntityRelationshipDefinition

	query := db.Model(&models.EntityRelationshipDefinition{})

	sourceEntityIDStr := c.Query("source_entity_id")
	// If source_entity_id is not in query, check path parameter "id" (for /entities/:id/relationships)
	if sourceEntityIDStr == "" {
		sourceEntityIDStr = c.Param("id")
	}

	if sourceEntityIDStr != "" {
		_, err := uuid.Parse(sourceEntityIDStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid source_entity_id query parameter format", gin.H{"source_entity_id": sourceEntityIDStr, "error": err.Error()})
			return
		}
		query = query.Where("source_entity_id = ?", sourceEntityIDStr) // Use string directly as GORM handles UUID conversion
	}

	if targetEntityIDStr := c.Query("target_entity_id"); targetEntityIDStr != "" {
		_, err := uuid.Parse(targetEntityIDStr)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid target_entity_id query parameter format", gin.H{"target_entity_id": targetEntityIDStr, "error": err.Error()})
			return
		}
		query = query.Where("target_entity_id = ?", targetEntityIDStr)
	}

	if relationshipTypeParam := c.Query("type"); relationshipTypeParam != "" {
		normalizedType := strings.ToUpper(relationshipTypeParam)
		if !models.ValidRelationshipTypes[normalizedType] {
			RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidEnumValue, "Invalid relationship_type filter. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY", gin.H{"relationship_type": relationshipTypeParam})
			return
		}
		query = query.Where("relationship_type = ?", normalizedType)
	}

	// Pagination and Sorting parameters
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultRelationshipLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid limit parameter: not a number.", gin.H{"limit": limitStr})
		return
	}
	if limit <= 0 {
		limit = DefaultRelationshipLimit
	} else if limit > MaxRelationshipLimit {
		limit = MaxRelationshipLimit
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

	sortBy := c.DefaultQuery("sort_by", DefaultRelationshipSortBy)
	if _, isValid := AllowedRelationshipSortByFields[sortBy]; !isValid {
		allowedFields := make([]string, 0, len(AllowedRelationshipSortByFields))
		for k := range AllowedRelationshipSortByFields {
			allowedFields = append(allowedFields, k)
		}
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_by field for relationships.", gin.H{"field": sortBy, "allowed": allowedFields})
		return
	}

	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", DefaultRelationshipSortOrder))
	if sortOrder != "asc" && sortOrder != "desc" {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_order value. Must be 'asc' or 'desc'.", gin.H{"value": c.Query("sort_order")})
		return
	}

	log.Printf("ListRelationships: Validated params: limit=%d, offset=%d, sortBy=%s, sortOrder=%s. Filters: sourceEntityID=%s, targetEntityID=%s, type=%s",
		limit, offset, sortBy, sortOrder,
		c.Query("source_entity_id"), c.Query("target_entity_id"), c.Query("type"))


	// This DB query will be updated in a later step
	if err := query.Find(&relationships).Error; err != nil {
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to retrieve relationships", nil)
		return
	}
	// This response will change in a later step
	RespondWithSuccess(c, http.StatusOK, relationships)
}

// GetRelationship godoc
// @Summary Get a specific entity relationship by ID
// @Description Retrieves the details of a specific entity relationship using its UUID.
// @Tags relationships
// @Produce  json
// @Param   id   path   string  true  "Entity Relationship ID (UUID)"
// @Success 200 {object} models.EntityRelationshipDefinition "Successfully retrieved entity relationship"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., relationship not found - see 'code' in response for specifics like RELATIONSHIP_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /relationships/{id} [get]
func GetRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")
	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid relationship ID format", gin.H{"id": idStr, "error": err.Error()})
		return
	}

	var relationship models.EntityRelationshipDefinition
	if err := db.First(&relationship, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeRelationshipNotFound, "Entity relationship not found", gin.H{"relationship_id": relationshipID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to retrieve relationship", nil)
		}
		return
	}
	RespondWithSuccess(c, http.StatusOK, relationship)
}

// UpdateRelationship godoc
// @Summary Update an existing entity relationship
// @Description Updates specific fields of an existing entity relationship. Source and Target entity IDs cannot be changed.
// @Tags relationships
// @Accept  json
// @Produce  json
// @Param   id   path   string  true  "Entity Relationship ID (UUID)"
// @Param   relationship  body   models.UpdateEntityRelationshipRequest  true  "Entity Relationship Update Request"
// @Success 200 {object} models.EntityRelationshipDefinition "Successfully updated entity relationship"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error, invalid ID format, invalid enum value - see 'code' in response for specifics like VALIDATION_ERROR, INVALID_ID_FORMAT, INVALID_ENUM_VALUE)"
// @Failure 404 {object} models.APIError "Not Found (e.g., relationship not found - see 'code' in response for specifics like RELATIONSHIP_NOT_FOUND)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate name - see 'code' in response for specifics like DUPLICATE_NAME)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /relationships/{id} [put]
func UpdateRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")
	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid relationship ID format", gin.H{"id": idStr, "error": err.Error()})
		return
	}

	var req models.UpdateEntityRelationshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	var existingRelationship models.EntityRelationshipDefinition
	if err := db.First(&existingRelationship, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeRelationshipNotFound, "Entity relationship not found", gin.H{"relationship_id": relationshipID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Database error finding relationship for update", nil)
		}
		return
	}

	// Update fields if provided in the request
	if req.Name != nil {
		existingRelationship.Name = *req.Name
	}
	if req.Description != nil {
		existingRelationship.Description = *req.Description
	}
	if req.RelationshipType != nil {
		normalizedType := strings.ToUpper(*req.RelationshipType)
		if !models.ValidRelationshipTypes[normalizedType] {
			RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidEnumValue, "Invalid relationship_type. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY", gin.H{"relationship_type": *req.RelationshipType})
			return
		}
		existingRelationship.RelationshipType = normalizedType
	}

	if err := db.Save(&existingRelationship).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "A relationship with the same name, source entity, and target entity already exists.", gin.H{"name": existingRelationship.Name, "source_id": existingRelationship.SourceEntityID, "target_id": existingRelationship.TargetEntityID})
				return
			}
		}
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to update entity relationship.", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, existingRelationship)
}

// DeleteRelationship godoc
// @Summary Delete an entity relationship by ID
// @Description Deletes a specific entity relationship using its UUID.
// @Tags relationships
// @Produce  json
// @Param   id   path   string  true  "Entity Relationship ID (UUID)"
// @Success 204 "Successfully deleted entity relationship (No Content)"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., relationship not found - see 'code' in response for specifics like RELATIONSHIP_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /relationships/{id} [delete]
// @Router /entities/{id}/relationships [get] 
// @Param   id path string false "Source Entity ID (UUID) - used when called via /entities/{id}/relationships"
func DeleteRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")
	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid relationship ID format", gin.H{"id": idStr, "error": err.Error()})
		return
	}

	// Check if the relationship exists before attempting to delete
	var relationshipCheck models.EntityRelationshipDefinition
	if err := db.First(&relationshipCheck, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeRelationshipNotFound, "Entity relationship not found", gin.H{"relationship_id": relationshipID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Database error finding relationship for deletion", nil)
		}
		return
	}

	if err := db.Delete(&models.EntityRelationshipDefinition{}, "id = ?", relationshipID).Error; err != nil {
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to delete entity relationship", nil)
		return
	}
	RespondWithSuccess(c, http.StatusNoContent, nil)
}
