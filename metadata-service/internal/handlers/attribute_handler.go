package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"metadata-service/internal/database"
	"metadata-service/internal/models"
)

// CreateAttribute godoc
// @Summary Create a new attribute for an entity
// @Description Create a new attribute definition for a specific entity.
// @Tags attributes
// @Accept  json
// @Produce  json
// @Param   entity_id      path   string     true  "Entity ID (UUID)"
// @Param   attribute_definition  body   models.CreateAttributeRequest   true  "Attribute Definition to create"
// @Success 201 {object} models.AttributeDefinition "Successfully created attribute definition"
// @Failure 400 {object} map[string]string "Invalid request payload or Entity ID format"
// @Failure 404 {object} map[string]string "Entity definition not found"
// @Failure 500 {object} map[string]string "Internal server error or unique constraint violation"
// @Router /entities/{id}/attributes [post]
func CreateAttribute(c *gin.Context) {
	entityIDStr := c.Param("id") // Changed from entity_id to id
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Entity ID format"})
		return
	}

	var req models.CreateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()

	// Check if parent entity exists
	var parentEntity models.EntityDefinition
	if err := db.First(&parentEntity, "id = ?", entityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent entity definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check parent entity: " + err.Error()})
		}
		return
	}

	attribute := models.AttributeDefinition{
		ID:           uuid.New(), // Generate new UUID for the attribute
		EntityID:     entityID,
		Name:         req.Name,
		DataType:     req.DataType,
		Description:  req.Description,
		IsFilterable: false, // Default, will be updated if provided
		IsPII:        false, // Default, will be updated if provided
	}
	if req.IsFilterable != nil {
		attribute.IsFilterable = *req.IsFilterable
	}
	if req.IsPII != nil {
		attribute.IsPII = *req.IsPII
	}

	if err := db.Create(&attribute).Error; err != nil {
		// Consider checking for unique constraint violation specifically
		// if db.Dialector.Name() == "postgres" && strings.Contains(err.Error(), "unique constraint") ...
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attribute definition: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, attribute)
}

// ListAttributes godoc
// @Summary List all attributes for a specific entity
// @Description Get a list of all attribute definitions associated with a given entity ID.
// @Tags attributes
// @Produce  json
// @Param   entity_id      path   string     true  "Entity ID (UUID)"
// @Success 200 {array} models.AttributeDefinition "Successfully retrieved list of attribute definitions"
// @Failure 400 {object} map[string]string "Invalid Entity ID format"
// @Failure 404 {object} map[string]string "Entity definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities/{id}/attributes [get]
func ListAttributes(c *gin.Context) {
	entityIDStr := c.Param("id") // Changed from entity_id to id
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Entity ID format"})
		return
	}

	db := database.GetDB()

	// Check if parent entity exists to provide a 404 if entity is not found
	var parentEntity models.EntityDefinition
	if err := db.First(&parentEntity, "id = ?", entityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check parent entity: " + err.Error()})
		}
		return
	}

	var attributes []models.AttributeDefinition
	if err := db.Where("entity_id = ?", entityID).Find(&attributes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list attribute definitions: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, attributes)
}

// GetAttribute godoc
// @Summary Get a specific attribute definition by ID
// @Description Get detailed information about a specific attribute definition using its UUID.
// @Tags attributes
// @Produce  json
// @Param   attribute_id     path   string     true  "Attribute Definition ID (UUID)"
// @Success 200 {object} models.AttributeDefinition "Successfully retrieved attribute definition"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Attribute definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /attributes/{attribute_id} [get]
func GetAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Attribute ID format"})
		return
	}

	db := database.GetDB()
	var attribute models.AttributeDefinition
	if err := db.First(&attribute, "id = ?", attributeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attribute definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attribute definition: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, attribute)
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
// @Failure 400 {object} map[string]string "Invalid request payload or ID format"
// @Failure 404 {object} map[string]string "Attribute definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /attributes/{attribute_id} [put]
func UpdateAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Attribute ID format"})
		return
	}

	var req models.UpdateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	var attribute models.AttributeDefinition
	if err := db.First(&attribute, "id = ?", attributeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attribute definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find attribute definition: " + err.Error()})
		}
		return
	}

	// Update fields if provided in the request
	if req.Name != nil {
		attribute.Name = *req.Name
	}
	if req.DataType != nil {
		// Additional validation for DataType if it's being changed
		if !models.ValidDataTypes[*req.DataType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DataType specified"})
			return
		}
		attribute.DataType = *req.DataType
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
	// EntityID should not be changed via this endpoint.

	if err := db.Save(&attribute).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attribute definition: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, attribute)
}

// DeleteAttribute godoc
// @Summary Delete an attribute definition
// @Description Delete an attribute definition by its UUID.
// @Tags attributes
// @Param   attribute_id     path   string     true  "Attribute Definition ID (UUID)"
// @Success 204 "Successfully deleted attribute definition"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Attribute definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /attributes/{attribute_id} [delete]
func DeleteAttribute(c *gin.Context) {
	attributeIDStr := c.Param("attribute_id")
	attributeID, err := uuid.Parse(attributeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Attribute ID format"})
		return
	}

	db := database.GetDB()
	var attribute models.AttributeDefinition
	// Check if attribute exists before trying to delete
	if err := db.First(&attribute, "id = ?", attributeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attribute definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find attribute definition: " + err.Error()})
		}
		return
	}

	if err := db.Delete(&models.AttributeDefinition{}, "id = ?", attributeID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attribute definition: " + err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
