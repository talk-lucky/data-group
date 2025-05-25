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

	// Group Definition Routes
	groupDefinitionRoutes := v1.Group("/group-definitions") // Renamed from /groups
	{
		groupDefinitionRoutes.POST("/", a.createGroupDefinitionHandler)
		groupDefinitionRoutes.GET("/", a.listGroupDefinitionsHandler)
		groupDefinitionRoutes.GET("/:group_id", a.getGroupDefinitionHandler) // group_id param name is fine
		groupDefinitionRoutes.PUT("/:group_id", a.updateGroupDefinitionHandler)
		groupDefinitionRoutes.DELETE("/:group_id", a.deleteGroupDefinitionHandler)
	}

	// Workflow Definition Routes
	workflowRoutes := v1.Group("/workflows")
	{
		workflowRoutes.POST("/", a.createWorkflowDefinitionHandler)
		workflowRoutes.GET("/", a.listWorkflowDefinitionsHandler)
		workflowRoutes.GET("/:workflow_id", a.getWorkflowDefinitionHandler)
		workflowRoutes.PUT("/:workflow_id", a.updateWorkflowDefinitionHandler)
		workflowRoutes.DELETE("/:workflow_id", a.deleteWorkflowDefinitionHandler)
	}

	// Action Template Routes
	actionTemplateRoutes := v1.Group("/actiontemplates")
	{
		actionTemplateRoutes.POST("/", a.createActionTemplateHandler)
		actionTemplateRoutes.GET("/", a.listActionTemplatesHandler)
		actionTemplateRoutes.GET("/:template_id", a.getActionTemplateHandler)
		actionTemplateRoutes.PUT("/:template_id", a.updateActionTemplateHandler)
		actionTemplateRoutes.DELETE("/:template_id", a.deleteActionTemplateHandler)
	}

	// Schedule Definition Routes
	scheduleRoutes := v1.Group("/schedules")
	{
		scheduleRoutes.POST("/", a.createScheduleDefinitionHandler)
		scheduleRoutes.GET("/", a.listScheduleDefinitionsHandler)
		scheduleRoutes.GET("/:schedule_id", a.getScheduleDefinitionHandler)
		scheduleRoutes.PUT("/:schedule_id", a.updateScheduleDefinitionHandler)
		scheduleRoutes.DELETE("/:schedule_id", a.deleteScheduleDefinitionHandler)
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


// --- ScheduleDefinition Handlers ---

func (a *API) createScheduleDefinitionHandler(c *gin.Context) {
	var req ScheduleDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = "" 

	schedule, err := a.store.CreateScheduleDefinition(req)
	if err != nil {
		// More specific error handling can be added if store returns typed errors
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Failed to create schedule: " + err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, schedule)
}

func (a *API) listScheduleDefinitionsHandler(c *gin.Context) {
	schedules, err := a.store.ListScheduleDefinitions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list schedules: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

func (a *API) getScheduleDefinitionHandler(c *gin.Context) {
	scheduleID := c.Param("schedule_id")
	schedule, err := a.store.GetScheduleDefinition(scheduleID)
	if err != nil {
		handleStoreError(c, err, "Schedule")
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (a *API) updateScheduleDefinitionHandler(c *gin.Context) {
	scheduleID := c.Param("schedule_id")
	var req ScheduleDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	schedule, err := a.store.UpdateScheduleDefinition(scheduleID, req)
	if err != nil {
		handleStoreError(c, err, "Schedule")
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (a *API) deleteScheduleDefinitionHandler(c *gin.Context) {
	scheduleID := c.Param("schedule_id")
	err := a.store.DeleteScheduleDefinition(scheduleID)
	if err != nil {
		handleStoreError(c, err, "Schedule")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- WorkflowDefinition Handlers ---

func (a *API) createWorkflowDefinitionHandler(c *gin.Context) {
	var req WorkflowDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	req.ID = "" // Set by store
	workflow, err := a.store.CreateWorkflowDefinition(req)
	if err != nil {
		handleStoreError(c, err, "Workflow Definition")
		return
	}
	c.JSON(http.StatusCreated, workflow)
}

func (a *API) listWorkflowDefinitionsHandler(c *gin.Context) {
	workflows, err := a.store.ListWorkflowDefinitions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workflow definitions: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, workflows)
}

func (a *API) getWorkflowDefinitionHandler(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	workflow, err := a.store.GetWorkflowDefinition(workflowID)
	if err != nil {
		handleStoreError(c, err, "Workflow Definition")
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (a *API) updateWorkflowDefinitionHandler(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	var req WorkflowDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	workflow, err := a.store.UpdateWorkflowDefinition(workflowID, req)
	if err != nil {
		handleStoreError(c, err, "Workflow Definition")
		return
	}
	c.JSON(http.StatusOK, workflow)
}

func (a *API) deleteWorkflowDefinitionHandler(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	err := a.store.DeleteWorkflowDefinition(workflowID)
	if err != nil {
		handleStoreError(c, err, "Workflow Definition")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- ActionTemplate Handlers ---

func (a *API) createActionTemplateHandler(c *gin.Context) {
	var req ActionTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	req.ID = "" // Set by store
	template, err := a.store.CreateActionTemplate(req)
	if err != nil {
		handleStoreError(c, err, "Action Template")
		return
	}
	c.JSON(http.StatusCreated, template)
}

func (a *API) listActionTemplatesHandler(c *gin.Context) {
	templates, err := a.store.ListActionTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list action templates: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (a *API) getActionTemplateHandler(c *gin.Context) {
	templateID := c.Param("template_id")
	template, err := a.store.GetActionTemplate(templateID)
	if err != nil {
		handleStoreError(c, err, "Action Template")
		return
	}
	c.JSON(http.StatusOK, template)
}

func (a *API) updateActionTemplateHandler(c *gin.Context) {
	templateID := c.Param("template_id")
	var req ActionTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	template, err := a.store.UpdateActionTemplate(templateID, req)
	if err != nil {
		handleStoreError(c, err, "Action Template")
		return
	}
	c.JSON(http.StatusOK, template)
}

func (a *API) deleteActionTemplateHandler(c *gin.Context) {
	templateID := c.Param("template_id")
	err := a.store.DeleteActionTemplate(templateID)
	if err != nil {
		handleStoreError(c, err, "Action Template")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// --- Attribute Handlers ---

// createAttributeHandler handles requests to create a new attribute for an entity.
func (a *API) createAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	var req struct {
		Name         string `json:"name" binding:"required"`
		DataType     string `json:"data_type" binding:"required"`
		Description  string `json:"description"`
		IsFilterable bool   `json:"is_filterable"`
		IsPii        bool   `json:"is_pii"`
		IsIndexed    bool   `json:"is_indexed"`
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

	attribute, err := a.store.CreateAttribute(entityID, req.Name, req.DataType, req.Description, req.IsFilterable, req.IsPii, req.IsIndexed)
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
		Name         string `json:"name" binding:"required"`
		DataType     string `json:"data_type" binding:"required"`
		Description  string `json:"description"`
		IsFilterable bool   `json:"is_filterable"`
		IsPii        bool   `json:"is_pii"`
		IsIndexed    bool   `json:"is_indexed"`
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

	attribute, err := a.store.UpdateAttribute(entityID, attributeID, req.Name, req.DataType, req.Description, req.IsFilterable, req.IsPii, req.IsIndexed)
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
	// Ensure EntityID is passed if provided in the request
	// The store.CreateDataSource function now expects DataSourceConfig object
	// which includes EntityID.

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
	// Ensure EntityID is passed if provided in the request for update
	// The store.UpdateDataSource function now expects DataSourceConfig object
	// which includes EntityID.

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

// --- GroupDefinition Handlers ---

// createGroupDefinitionHandler handles requests to create a new group definition.
func (a *API) createGroupDefinitionHandler(c *gin.Context) {
	var req GroupDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""

	groupDef, err := a.store.CreateGroupDefinition(req)
	if err != nil {
		// Check if it's a "not found" error for the entity_id
		if strings.Contains(err.Error(), "entity with ID") && strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "cannot be empty") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group definition: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, groupDef)
}

// listGroupDefinitionsHandler handles requests to list all group definitions.
func (a *API) listGroupDefinitionsHandler(c *gin.Context) {
	groupDefs, err := a.store.ListGroupDefinitions()
	if err != nil { // Should generally not happen for a list operation unless there's a fundamental store issue
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list group definitions: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, groupDefs)
}

// getGroupDefinitionHandler handles requests to get a specific group definition by ID.
func (a *API) getGroupDefinitionHandler(c *gin.Context) {
	groupID := c.Param("group_id")
	groupDef, err := a.store.GetGroupDefinition(groupID)
	if err != nil {
		handleStoreError(c, err, "Group Definition")
		return
	}
	c.JSON(http.StatusOK, groupDef)
}

// updateGroupDefinitionHandler handles requests to update a specific group definition.
func (a *API) updateGroupDefinitionHandler(c *gin.Context) {
	groupID := c.Param("group_id")
	var req GroupDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}
	// EntityID is not expected to be in the req payload for update, or if it is, it should match existing or be ignored by store.
	// Store's UpdateGroupDefinition should handle this logic.

	groupDef, err := a.store.UpdateGroupDefinition(groupID, req)
	if err != nil {
		handleStoreError(c, err, "Group Definition")
		return
	}
	c.JSON(http.StatusOK, groupDef)
}

// deleteGroupDefinitionHandler handles requests to delete a specific group definition.
func (a *API) deleteGroupDefinitionHandler(c *gin.Context) {
	groupID := c.Param("group_id")
	err := a.store.DeleteGroupDefinition(groupID)
	if err != nil {
		handleStoreError(c, err, "Group Definition")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
