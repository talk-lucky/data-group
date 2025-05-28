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
	"metadata-service/internal/models"
)

// Constants for pagination and sorting for Attributes
const (
	DefaultAttributeLimit       = 10
	MaxAttributeLimit           = 100
	DefaultAttributeSortOrder   = "asc"
	DefaultAttributeSortBy      = "created_at"
)

// AllowedAttributeSortByFields defines the fields by which a list of attributes can be sorted.
var AllowedAttributeSortByFields = map[string]bool{
	"name":         true,
	"data_type":    true,
	"created_at":   true,
	"updated_at":   true,
}

// CreateAttribute godoc
// @Summary Create a new attribute for an entity
// @Description Create a new attribute definition for a specific entity.
// @Tags attributes
// @Accept  json
// @Produce  json
// @Param   entity_id      path   string     true  "Entity ID (UUID)"
// @Param   attribute_definition  body   models.CreateAttributeRequest   true  "Attribute Definition to create"
// @Success 201 {object} models.AttributeDefinition "Successfully created attribute definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error, invalid Entity ID format - see 'code' in response for specifics like VALIDATION_ERROR, INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., parent entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate attribute name for the entity - see 'code' in response for specifics like DUPLICATE_NAME)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities/{id}/attributes [post]
func CreateAttribute(c *gin.Context) {
	entityIDStr := c.Param("id")
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid Entity ID format", gin.H{"id": entityIDStr, "error": err.Error()})
		return
	}

	var req models.CreateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	db := database.GetDB()

	// Check if parent entity exists
	var parentEntity models.EntityDefinition
	if err := db.First(&parentEntity, "id = ?", entityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Parent entity definition not found", gin.H{"entity_id": entityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to check parent entity existence", nil)
		}
		return
	}

	attribute := models.AttributeDefinition{
		ID:           uuid.New(),
		EntityID:     entityID,
		Name:         req.Name,
		DataType:     req.DataType,
		Description:  req.Description,
		IsFilterable: false,
		IsPII:        false,
	}
	if req.IsFilterable != nil {
		attribute.IsFilterable = *req.IsFilterable
	}
	if req.IsPII != nil {
		attribute.IsPII = *req.IsPII
	}

	if err := db.Create(&attribute).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // PostgreSQL error code for unique_violation
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "Attribute with this name already exists for the entity.", gin.H{"name": attribute.Name, "entity_id": entityID})
				return
			}
		}
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to create attribute definition.", nil)
		return
	}
	RespondWithSuccess(c, http.StatusCreated, attribute)
}

// ListAttributes godoc
// @Summary List all attributes for a specific entity
// @Description Get a list of all attribute definitions associated with a given entity ID.
// @Tags attributes
// @Produce  json
// @Param   entity_id      path   string     true  "Entity ID (UUID)"
// @Success 200 {array} models.AttributeDefinition "Successfully retrieved list of attribute definitions"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid Entity ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., parent entity not found - see 'code' in response for specifics like ENTITY_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /entities/{id}/attributes [get]
func ListAttributes(c *gin.Context) {
	entityIDStr := c.Param("id")
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid Entity ID format", gin.H{"id": entityIDStr, "error": err.Error()})
		return
	}

	// Pagination and Sorting parameters
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultAttributeLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid limit parameter: not a number.", gin.H{"limit": limitStr})
		return
	}
	if limit <= 0 {
		limit = DefaultAttributeLimit
	} else if limit > MaxAttributeLimit {
		limit = MaxAttributeLimit
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

	sortBy := c.DefaultQuery("sort_by", DefaultAttributeSortBy)
	if _, isValid := AllowedAttributeSortByFields[sortBy]; !isValid {
		allowedFields := make([]string, 0, len(AllowedAttributeSortByFields))
		for k := range AllowedAttributeSortByFields {
			allowedFields = append(allowedFields, k)
		}
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_by field for attributes.", gin.H{"field": sortBy, "allowed": allowedFields})
		return
	}

	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", DefaultAttributeSortOrder))
	if sortOrder != "asc" && sortOrder != "desc" {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid sort_order value. Must be 'asc' or 'desc'.", gin.H{"value": c.Query("sort_order")})
		return
	}

	log.Printf("ListAttributes for EntityID %s: Validated params: limit=%d, offset=%d, sortBy=%s, sortOrder=%s", entityID, limit, offset, sortBy, sortOrder)

	db := database.GetDB()

	// Check if parent entity exists
	var parentEntity models.EntityDefinition
	if err := db.First(&parentEntity, "id = ?", entityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeEntityNotFound, "Parent entity definition not found", gin.H{"entity_id": entityID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to check parent entity existence", nil)
		}
		return
	}

	var attributes []models.AttributeDefinition
	// This DB query will be updated in a later step
	if err := db.Where("entity_id = ?", entityID).Find(&attributes).Error; err != nil {
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to list attribute definitions", nil)
		return
	}
	// This response will change in a later step
	RespondWithSuccess(c, http.StatusOK, attributes)
}

// GetAttribute godoc
// @Summary Get a specific attribute definition by ID
// @Description Get detailed information about a specific attribute definition using its UUID.
// @Tags attributes
// @Produce  json
// @Param   attribute_id     path   string     true  "Attribute Definition ID (UUID)"
// @Success 200 {object} models.AttributeDefinition "Successfully retrieved attribute definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid Attribute ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., attribute not found - see 'code' in response for specifics like ATTRIBUTE_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /attributes/{attribute_id} [get]
func GetAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid Attribute ID format", gin.H{"id": attributeIDStr, "error": err.Error()})
		return
	}

	db := database.GetDB()
	var attribute models.AttributeDefinition
	if err := db.First(&attribute, "id = ?", attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeAttributeNotFound, "Attribute definition not found", gin.H{"attribute_id": attributeID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to get attribute definition", nil)
		}
		return
	}
	RespondWithSuccess(c, http.StatusOK, attribute)
}

// UpdateAttribute godoc
// @Summary Update an existing attribute definition
// @Description Update an existing attribute definition's fields. EntityID cannot be changed.
// @Tags attributes
// @Accept  json
// @Produce  json
// @Param   attribute_id     path   string     true  "Attribute Definition ID (UUID)"
// @Param   attribute_definition  body   models.UpdateAttributeRequest   true  "Attribute Definition fields to update"
// @Success 200 {object} models.AttributeDefinition "Successfully updated attribute definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., validation error, invalid ID format, invalid DataType - see 'code' in response for specifics like VALIDATION_ERROR, INVALID_ID_FORMAT, INVALID_ENUM_VALUE)"
// @Failure 404 {object} models.APIError "Not Found (e.g., attribute not found - see 'code' in response for specifics like ATTRIBUTE_NOT_FOUND)"
// @Failure 409 {object} models.APIError "Conflict (e.g., duplicate attribute name for the entity - see 'code' in response for specifics like DUPLICATE_NAME)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /attributes/{attribute_id} [put]
func UpdateAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid Attribute ID format", gin.H{"id": attributeIDStr, "error": err.Error()})
		return
	}

	var req models.UpdateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid request payload", gin.H{"reason": err.Error()})
		return
	}

	db := database.GetDB()
	var attribute models.AttributeDefinition
	if err := db.First(&attribute, "id = ?", attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeAttributeNotFound, "Attribute definition not found", gin.H{"attribute_id": attributeID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to find attribute definition for update", nil)
		}
		return
	}

	// Update fields if provided in the request
	if req.Name != nil {
		attribute.Name = *req.Name
	}
	if req.DataType != nil {
		// Convert to uppercase for consistent validation against models.ValidDataTypes
		normalizedDataType := strings.ToUpper(*req.DataType)
		if !models.ValidDataTypes[normalizedDataType] {
			RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidEnumValue, "Invalid DataType specified", gin.H{"data_type": *req.DataType})
			return
		}
		attribute.DataType = normalizedDataType // Store the normalized uppercase version
	}
	if req.Description != nil {
		attribute.Description = *req.Description
	}
	if req.IsFilterable != nil {
		attribute.IsFilterable = *req.IsFilterable
	}
	if req.IsPII != nil {
		attribute.IsPII = *req.IsPII
	}

	if err := db.Save(&attribute).Error; err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				RespondWithError(c, http.StatusConflict, models.ErrorCodeDuplicateName, "Attribute with this name already exists for the entity.", gin.H{"name": attribute.Name, "entity_id": attribute.EntityID})
				return
			}
		}
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to update attribute definition.", nil)
		return
	}
	RespondWithSuccess(c, http.StatusOK, attribute)
}

// DeleteAttribute godoc
// @Summary Delete an attribute definition
// @Description Delete an attribute definition by its UUID.
// @Tags attributes
// @Param   attribute_id     path   string     true  "Attribute Definition ID (UUID)"
// @Success 204 "Successfully deleted attribute definition"
// @Failure 400 {object} models.APIError "Bad Request (e.g., invalid Attribute ID format - see 'code' in response for specifics like INVALID_ID_FORMAT)"
// @Failure 404 {object} models.APIError "Not Found (e.g., attribute not found - see 'code' in response for specifics like ATTRIBUTE_NOT_FOUND)"
// @Failure 500 {object} models.APIError "Internal Server Error (see 'code' in response for specifics like INTERNAL_SERVER_ERROR)"
// @Router /attributes/{attribute_id} [delete]
func DeleteAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, models.ErrorCodeInvalidIDFormat, "Invalid Attribute ID format", gin.H{"id": attributeIDStr, "error": err.Error()})
		return
	}

	db := database.GetDB()
	// Check if attribute exists before trying to delete
	var attributeCheck models.AttributeDefinition
	if err := db.First(&attributeCheck, "id = ?", attributeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondWithError(c, http.StatusNotFound, models.ErrorCodeAttributeNotFound, "Attribute definition not found", gin.H{"attribute_id": attributeID})
		} else {
			RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to find attribute definition for deletion", nil)
		}
		return
	}

	if err := db.Delete(&models.AttributeDefinition{}, "id = ?", attributeID).Error; err != nil {
		RespondWithError(c, http.StatusInternalServerError, models.ErrorCodeInternalServerError, "Failed to delete attribute definition", nil)
		return
	}
	RespondWithSuccess(c, http.StatusNoContent, nil)
}
