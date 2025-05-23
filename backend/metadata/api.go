package metadata

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// API provides handlers for the metadata service.
type API struct {
	store *Store
}

// NewAPI creates a new API handler with the given store.
func NewAPI(store *Store) *API {
	return &API{store: store}
}

// RegisterRoutes registers the metadata API routes with the given Gin router.
func (a *API) RegisterRoutes(router *gin.Engine) {
	entityRoutes := router.Group("/api/v1/entities")
	{
		entityRoutes.POST("/", a.createEntityHandler)
		entityRoutes.GET("/", a.listEntitiesHandler)
		entityRoutes.GET("/:entity_id", a.getEntityHandler)
		entityRoutes.PUT("/:entity_id", a.updateEntityHandler)
		entityRoutes.DELETE("/:entity_id", a.deleteEntityHandler)

		attributeRoutes := entityRoutes.Group("/:entity_id/attributes")
		{
			attributeRoutes.POST("/", a.createAttributeHandler)
			attributeRoutes.GET("/", a.listAttributesHandler)
			attributeRoutes.GET("/:attribute_id", a.getAttributeHandler)
			attributeRoutes.PUT("/:attribute_id", a.updateAttributeHandler)
			attributeRoutes.DELETE("/:attribute_id", a.deleteAttributeHandler)
		}
	}
}

// createEntityHandler handles requests to create a new entity.
func (a *API) createEntityHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	entity, err := a.store.CreateEntity(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create entity: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, entity)
}

// listEntitiesHandler handles requests to list all entities.
func (a *API) listEntitiesHandler(c *gin.Context) {
	entities := a.store.ListEntities()
	c.JSON(http.StatusOK, entities)
}

// getEntityHandler handles requests to get a specific entity by ID.
func (a *API) getEntityHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	entity, ok := a.store.GetEntity(entityID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

// updateEntityHandler handles requests to update a specific entity.
func (a *API) updateEntityHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	entity, err := a.store.UpdateEntity(entityID, req.Name, req.Description)
	if err != nil {
		// Check if error is due to not found
		if err.Error() == "entity with ID "+entityID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update entity: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, entity)
}

// deleteEntityHandler handles requests to delete a specific entity.
func (a *API) deleteEntityHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	err := a.store.DeleteEntity(entityID)
	if err != nil {
		// Check if error is due to not found
		if err.Error() == "entity with ID "+entityID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete entity: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// createAttributeHandler handles requests to create a new attribute for an entity.
func (a *API) createAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		DataType    string `json:"data_type" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	attribute, err := a.store.CreateAttribute(entityID, req.Name, req.DataType, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attribute: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, attribute)
}

// listAttributesHandler handles requests to list all attributes for an entity.
func (a *API) listAttributesHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	attributes, err := a.store.ListAttributes(entityID)
	if err != nil { // Should not happen if entity existence is checked
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list attributes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, attributes)
}

// getAttributeHandler handles requests to get a specific attribute by ID.
func (a *API) getAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	attributeID := c.Param("attribute_id")

	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	attribute, ok := a.store.GetAttribute(entityID, attributeID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attribute not found"})
		return
	}
	c.JSON(http.StatusOK, attribute)
}

// updateAttributeHandler handles requests to update a specific attribute.
func (a *API) updateAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	attributeID := c.Param("attribute_id")
	var req struct {
		Name        string `json:"name" binding:"required"`
		DataType    string `json:"data_type" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Check if entity exists first
	if _, ok := a.store.GetEntity(entityID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}
	
	attribute, err := a.store.UpdateAttribute(entityID, attributeID, req.Name, req.DataType, req.Description)
	if err != nil {
		// Check if error is due to attribute not found or other store issues
		if err.Error() == "attribute with ID "+attributeID+" not found for entity ID "+entityID || 
		   err.Error() == "no attributes found for entity ID "+entityID { // This second case might indicate entity exists but has no attribute map (shouldn't happen with current store)
			c.JSON(http.StatusNotFound, gin.H{"error": "Attribute not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attribute: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, attribute)
}

// deleteAttributeHandler handles requests to delete a specific attribute.
func (a *API) deleteAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	attributeID := c.Param("attribute_id")

	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	err := a.store.DeleteAttribute(entityID, attributeID)
	if err != nil {
		// Check if error is due to attribute not found or other store issues
		if err.Error() == "attribute with ID "+attributeID+" not found for entity ID "+entityID ||
		   err.Error() == "no attributes found for entity ID "+entityID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Attribute not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attribute: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
