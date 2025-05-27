package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"metadata-service/internal/database"
	"metadata-service/internal/models"
)

// CreateEntity godoc
// @Summary Create a new entity definition
// @Description Create a new entity definition with the provided name and description.
// @Tags entities
// @Accept  json
// @Produce  json
// @Param   entity_definition  body   models.CreateEntityRequest   true  "Entity Definition to create"
// @Success 201 {object} models.EntityDefinition "Successfully created entity definition"
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities [post]
func CreateEntity(c *gin.Context) {
	var req models.CreateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	entity := models.EntityDefinition{
		ID:          uuid.New(), // Generate UUID before creation
		Name:        req.Name,
		Description: req.Description,
	}

	if err := db.Create(&entity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create entity definition: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entity)
}

// ListEntities godoc
// @Summary List all entity definitions
// @Description Get a list of all entity definitions.
// @Tags entities
// @Produce  json
// @Success 200 {array} models.EntityDefinition "Successfully retrieved list of entity definitions"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities [get]
func ListEntities(c *gin.Context) {
	db := database.GetDB()
	var entities []models.EntityDefinition
	if err := db.Find(&entities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list entity definitions: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// GetEntity godoc
// @Summary Get a specific entity definition by ID
// @Description Get detailed information about a specific entity definition using its UUID.
// @Tags entities
// @Produce  json
// @Param   id     path   string     true  "Entity Definition ID (UUID)"
// @Success 200 {object} models.EntityDefinition "Successfully retrieved entity definition"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Entity definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities/{id} [get]
func GetEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	db := database.GetDB()
	var entity models.EntityDefinition
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get entity definition: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, entity)
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
// @Failure 400 {object} map[string]string "Invalid request payload or ID format"
// @Failure 404 {object} map[string]string "Entity definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities/{id} [put]
func UpdateEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req models.UpdateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	var entity models.EntityDefinition
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find entity definition: " + err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update entity definition: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, entity)
}

// DeleteEntity godoc
// @Summary Delete an entity definition
// @Description Delete an entity definition by its UUID.
// @Tags entities
// @Param   id     path   string     true  "Entity Definition ID (UUID)"
// @Success 204 "Successfully deleted entity definition"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Entity definition not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /entities/{id} [delete]
func DeleteEntity(c *gin.Context) {
	idStr := c.Param("id")
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	db := database.GetDB()
	var entity models.EntityDefinition
	// Check if entity exists before trying to delete
	if err := db.First(&entity, "id = ?", entityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity definition not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find entity definition: " + err.Error()})
		}
		return
	}

	if err := db.Delete(&models.EntityDefinition{}, "id = ?", entityID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete entity definition: " + err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
