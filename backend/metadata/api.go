package metadata

import (
	"net/http"
	"strings"

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
	v1 := router.Group("/api/v1")

	// Entity Routes
	entityRoutes := v1.Group("/entities")
	{
		entityRoutes.POST("/", a.createEntityHandler)
		entityRoutes.GET("/", a.listEntitiesHandler)
		entityRoutes.GET("/:entity_id", a.getEntityHandler)
		entityRoutes.PUT("/:entity_id", a.updateEntityHandler)
		entityRoutes.DELETE("/:entity_id", a.deleteEntityHandler)

		// Attribute Routes (nested under entities)
		attributeRoutes := entityRoutes.Group("/:entity_id/attributes")
		{
			attributeRoutes.POST("/", a.createAttributeHandler)
			attributeRoutes.GET("/", a.listAttributesHandler)
			attributeRoutes.GET("/:attribute_id", a.getAttributeHandler)
			attributeRoutes.PUT("/:attribute_id", a.updateAttributeHandler)
			attributeRoutes.DELETE("/:attribute_id", a.deleteAttributeHandler)
		}
	}

	// Data Source Routes
	dataSourceRoutes := v1.Group("/datasources")
	{
		dataSourceRoutes.POST("/", a.createDataSourceHandler)
		dataSourceRoutes.GET("/", a.listDataSourcesHandler)
		dataSourceRoutes.GET("/:source_id", a.getDataSourceHandler)
		dataSourceRoutes.PUT("/:source_id", a.updateDataSourceHandler)
		dataSourceRoutes.DELETE("/:source_id", a.deleteDataSourceHandler)

		// Field Mapping Routes (nested under data sources)
		mappingRoutes := dataSourceRoutes.Group("/:source_id/mappings")
		{
			mappingRoutes.POST("/", a.createFieldMappingHandler)
			mappingRoutes.GET("/", a.listFieldMappingsHandler)
			mappingRoutes.GET("/:mapping_id", a.getFieldMappingHandler)
			mappingRoutes.PUT("/:mapping_id", a.updateFieldMappingHandler)
			mappingRoutes.DELETE("/:mapping_id", a.deleteFieldMappingHandler)
		}
	}
}

// --- Entity Handlers ---

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
		handleStoreError(c, err, "Entity")
		return
	}
	c.JSON(http.StatusOK, entity)
}

// deleteEntityHandler handles requests to delete a specific entity.
func (a *API) deleteEntityHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	err := a.store.DeleteEntity(entityID)
	if err != nil {
		handleStoreError(c, err, "Entity")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- Attribute Handlers ---

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
		handleStoreError(c, err, "Attribute")
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
		handleStoreError(c, err, "Attribute")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- DataSource Handlers ---

func (a *API) createDataSourceHandler(c *gin.Context) {
	var req DataSourceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""

	ds, err := a.store.CreateDataSource(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create data source: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ds)
}

func (a *API) listDataSourcesHandler(c *gin.Context) {
	sources, err := a.store.GetDataSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list data sources: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, sources)
}

func (a *API) getDataSourceHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	ds, err := a.store.GetDataSource(sourceID)
	if err != nil {
		handleStoreError(c, err, "Data Source")
		return
	}
	c.JSON(http.StatusOK, ds)
}

func (a *API) updateDataSourceHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	var req DataSourceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	ds, err := a.store.UpdateDataSource(sourceID, req)
	if err != nil {
		handleStoreError(c, err, "Data Source")
		return
	}
	c.JSON(http.StatusOK, ds)
}

func (a *API) deleteDataSourceHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	err := a.store.DeleteDataSource(sourceID)
	if err != nil {
		handleStoreError(c, err, "Data Source")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- FieldMapping Handlers ---

func (a *API) createFieldMappingHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	var req DataSourceFieldMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// Ensure SourceID from path matches payload, or set it from path
	if req.SourceID != "" && req.SourceID != sourceID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SourceID in path and payload do not match"})
		return
	}
	req.SourceID = sourceID
	req.ID = "" // ID is set by the store

	mapping, err := a.store.CreateFieldMapping(req)
	if err != nil {
		// More specific error handling for FK violations might be needed if store returns typed errors
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Failed to create field mapping: " + err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create field mapping: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, mapping)
}

func (a *API) listFieldMappingsHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	// Check if data source exists
	if _, err := a.store.GetDataSource(sourceID); err != nil {
		handleStoreError(c, err, "Data Source")
		return
	}

	mappings, err := a.store.GetFieldMappings(sourceID)
	if err != nil { // Should ideally not happen if source existence is checked
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list field mappings: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, mappings)
}

func (a *API) getFieldMappingHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	mappingID := c.Param("mapping_id")

	mapping, err := a.store.GetFieldMapping(sourceID, mappingID)
	if err != nil {
		handleStoreError(c, err, "Field Mapping")
		return
	}
	c.JSON(http.StatusOK, mapping)
}

func (a *API) updateFieldMappingHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	mappingID := c.Param("mapping_id")
	var req DataSourceFieldMapping
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// Ensure SourceID from path matches payload, or set it from path
	if req.SourceID != "" && req.SourceID != sourceID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SourceID in path and payload do not match"})
		return
	}
	req.SourceID = sourceID

	mapping, err := a.store.UpdateFieldMapping(sourceID, mappingID, req)
	if err != nil {
		// More specific error handling for FK violations
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Failed to update field mapping: " + err.Error()})
		} else {
			handleStoreError(c, err, "Field Mapping")
		}
		return
	}
	c.JSON(http.StatusOK, mapping)
}

func (a *API) deleteFieldMappingHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	mappingID := c.Param("mapping_id")
	err := a.store.DeleteFieldMapping(sourceID, mappingID)
	if err != nil {
		handleStoreError(c, err, "Field Mapping")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// handleStoreError is a helper to reduce repetition in error handling
func handleStoreError(c *gin.Context, err error, resourceName string) {
	if strings.Contains(err.Error(), "not found") {
		c.JSON(http.StatusNotFound, gin.H{"error": resourceName + " not found"})
	} else if strings.Contains(err.Error(), "cannot be empty") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process " + strings.ToLower(resourceName) + ": " + err.Error()})
	}
}
