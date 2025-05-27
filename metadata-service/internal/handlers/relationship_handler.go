package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"metadata-service/internal/database"
	"metadata-service/internal/models"
)

// CreateRelationship godoc
// @Summary Create a new entity relationship
// @Description Creates a new entity relationship definition between two existing entities.
// @Tags relationships
// @Accept  json
// @Produce  json
// @Param   relationship  body   models.CreateEntityRelationshipRequest  true  "Entity Relationship Creation Request"
// @Success 201 {object} models.EntityRelationshipDefinition "Successfully created entity relationship"
// @Failure 400 {object} gin.H "Invalid request payload or parameters"
// @Failure 404 {object} gin.H "Source or Target Entity not found"
// @Failure 422 {object} gin.H "Invalid relationship type or validation error"
// @Failure 500 {object} gin.H "Internal server error"
// @Router /relationships [post]
func CreateRelationship(c *gin.Context) {
	db := database.GetDB()
	var req models.CreateEntityRelationshipRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate RelationshipType
	if !models.ValidRelationshipTypes[strings.ToUpper(req.RelationshipType)] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid relationship_type. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY"})
		return
	}

	sourceEntityID, err := uuid.Parse(req.SourceEntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_entity_id format: " + err.Error()})
		return
	}

	targetEntityID, err := uuid.Parse(req.TargetEntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_entity_id format: " + err.Error()})
		return
	}

	// Validate that SourceEntityID exists
	var sourceEntity models.EntityDefinition
	if err := db.First(&sourceEntity, "id = ?", sourceEntityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source entity not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking source entity: " + err.Error()})
		return
	}

	// Validate that TargetEntityID exists
	var targetEntity models.EntityDefinition
	if err := db.First(&targetEntity, "id = ?", targetEntityID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target entity not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking target entity: " + err.Error()})
		return
	}
	
	// Prevent self-referential relationships if that's a business rule (optional, not specified but good practice for some cases)
	// if sourceEntityID == targetEntityID {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Source and target entity cannot be the same"})
	// 	return
	// }


	relationship := models.EntityRelationshipDefinition{
		ID:               uuid.New(),
		Name:             req.Name,
		SourceEntityID:   sourceEntityID,
		TargetEntityID:   targetEntityID,
		RelationshipType: strings.ToUpper(req.RelationshipType),
		Description:      req.Description,
	}

	if err := db.Create(&relationship).Error; err != nil {
		// Check for unique constraint violation (name, source_entity_id, target_entity_id)
		// Note: GORM might not return a specific error type for this, driver dependent.
		// A common way is to check the error message string.
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "A relationship with the same name, source entity, and target entity already exists."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create entity relationship: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, relationship)
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
// @Failure 400 {object} gin.H "Invalid query parameter format"
// @Failure 500 {object} gin.H "Internal server error"
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
		sourceEntityID, err := uuid.Parse(sourceEntityIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_entity_id format (from query or path): " + err.Error()})
			return
		}
		query = query.Where("source_entity_id = ?", sourceEntityID)
	}

	if targetEntityIDStr := c.Query("target_entity_id"); targetEntityIDStr != "" {
		targetEntityID, err := uuid.Parse(targetEntityIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_entity_id format: " + err.Error()})
			return
		}
		query = query.Where("target_entity_id = ?", targetEntityID)
	}

	if relationshipType := c.Query("type"); relationshipType != "" {
		normalizedType := strings.ToUpper(relationshipType)
		if !models.ValidRelationshipTypes[normalizedType] {
             c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship_type filter. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY"})
             return
        }
		query = query.Where("relationship_type = ?", normalizedType)
	}

	if err := query.Find(&relationships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve relationships: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, relationships)
}

// GetRelationship godoc
// @Summary Get a specific entity relationship by ID
// @Description Retrieves the details of a specific entity relationship using its UUID.
// @Tags relationships
// @Produce  json
// @Param   id   path   string  true  "Entity Relationship ID (UUID)"
// @Success 200 {object} models.EntityRelationshipDefinition "Successfully retrieved entity relationship"
// @Failure 400 {object} gin.H "Invalid relationship ID format"
// @Failure 404 {object} gin.H "Entity relationship not found"
// @Failure 500 {object} gin.H "Internal server error"
// @Router /relationships/{id} [get]
func GetRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")

	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship ID format: " + err.Error()})
		return
	}

	var relationship models.EntityRelationshipDefinition
	if err := db.First(&relationship, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity relationship not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve relationship: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, relationship)
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
// @Failure 400 {object} gin.H "Invalid request payload or ID format"
// @Failure 404 {object} gin.H "Entity relationship not found"
// @Failure 422 {object} gin.H "Invalid relationship type or validation error"
// @Failure 500 {object} gin.H "Internal server error"
// @Router /relationships/{id} [put]
func UpdateRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")

	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship ID format: " + err.Error()})
		return
	}

	var req models.UpdateEntityRelationshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	var existingRelationship models.EntityRelationshipDefinition
	if err := db.First(&existingRelationship, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity relationship not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error finding relationship: " + err.Error()})
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
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid relationship_type. Must be one of: ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY"})
			return
		}
		existingRelationship.RelationshipType = normalizedType
	}

	if err := db.Save(&existingRelationship).Error; err != nil {
		// Check for unique constraint violation (name, source_entity_id, target_entity_id)
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "A relationship with the same name, source entity, and target entity already exists."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update entity relationship: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingRelationship)
}

// DeleteRelationship godoc
// @Summary Delete an entity relationship by ID
// @Description Deletes a specific entity relationship using its UUID.
// @Tags relationships
// @Produce  json
// @Param   id   path   string  true  "Entity Relationship ID (UUID)"
// @Success 204 "Successfully deleted entity relationship (No Content)"
// @Failure 400 {object} gin.H "Invalid relationship ID format"
// @Failure 404 {object} gin.H "Entity relationship not found"
// @Failure 500 {object} gin.H "Internal server error"
// @Router /relationships/{id} [delete]
// @Router /entities/{id}/relationships [get]
// @Param   id path string false "Source Entity ID (UUID) - used when called via /entities/{id}/relationships"
func DeleteRelationship(c *gin.Context) {
	db := database.GetDB()
	idStr := c.Param("id")

	relationshipID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship ID format: " + err.Error()})
		return
	}

	// Check if the relationship exists before attempting to delete
	var relationship models.EntityRelationshipDefinition
	if err := db.First(&relationship, "id = ?", relationshipID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity relationship not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error finding relationship: " + err.Error()})
		return
	}

	if err := db.Delete(&models.EntityRelationshipDefinition{}, "id = ?", relationshipID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete entity relationship: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
