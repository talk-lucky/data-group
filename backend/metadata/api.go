package metadata

import (
	"net/http"
	"fmt"
	"net/http"
	"strconv" // Added for Atoi
	"strings"

	"github.com/gin-gonic/gin"
	// Assuming ListParams is in models package after previous (intended) move
	// If not, and it's still in metadata (store.go), this import might not be needed
	// or "models" should be the alias for the current package "metadata"
	// For now, we'll assume ListParams is accessible directly or via current package scope.
)

// DefaultLimitStr is used if models.DefaultLimitStr is not accessible due to tool state issues.
const DefaultLimitStr = "20"

// handleAPIError is a helper function to standardize API error responses.
// It takes a gin.Context, an HTTP status code, and an error message string or an error object.
func handleAPIError(c *gin.Context, statusCode int, err interface{}) {
	var message string
	switch e := err.(type) {
	case error:
		message = e.Error()
	case string:
		message = e
	default:
		message = "An unexpected error occurred"
	}
	c.JSON(statusCode, APIError{Code: statusCode, Message: message})
}

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

	// Entity Relationship Routes
	entityRelationshipRoutes := v1.Group("/entity-relationships")
	{
		entityRelationshipRoutes.POST("/", a.createEntityRelationshipHandler)
		entityRelationshipRoutes.GET("/", a.listEntityRelationshipsHandler)
		entityRelationshipRoutes.GET("/:id", a.getEntityRelationshipHandler)
		entityRelationshipRoutes.PUT("/:id", a.updateEntityRelationshipHandler)
		entityRelationshipRoutes.DELETE("/:id", a.deleteEntityRelationshipHandler)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	entity, err := a.store.CreateEntity(req.Name, req.Description)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to create entity: "+err.Error())
		return
	}
	c.JSON(http.StatusCreated, entity)
}

// listEntitiesHandler handles requests to list all entities.
func (a *API) listEntitiesHandler(c *gin.Context) {
	// Assuming a.store.ListEntities() returns []EntityDefinition and no error for now
	// In a real scenario, if a.store.ListEntities could return an error:
	// entities, total, err := a.store.ListEntities(offset, limit) // or similar
	// if err != nil {
	//    handleAPIError(c, http.StatusInternalServerError, "Failed to list entities: "+err.Error())
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr) // Use local or models.DefaultLimitStr

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	entities, total, err := a.store.ListEntities(params) // Call updated store method
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list entities: "+err.Error())
		return
	}

	response := ListResponse{Data: entities, Total: total}
	c.JSON(http.StatusOK, response)
}

// getEntityHandler handles requests to get a specific entity by ID.
func (a *API) getEntityHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	entity, ok := a.store.GetEntity(entityID)
	if !ok {
		handleAPIError(c, http.StatusNotFound, "Entity not found")
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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

// determineBulkResponseStatus determines the appropriate HTTP status code for a bulk operation.
// It returns 201 for create if all items succeeded, 200 for update/delete if all items succeeded.
// If any item failed, it returns 207 Multi-Status.
// If results are empty, it returns 200 OK.
func determineBulkResponseStatus(results []BulkOperationResultItem, operationType string) int {
	if len(results) == 0 {
		return http.StatusOK // Or perhaps http.StatusNoContent if preferred for empty requests
	}

	allSucceeded := true
	for _, result := range results {
		if !result.Success {
			allSucceeded = false
			break
		}
	}

	if allSucceeded {
		if operationType == "create" {
			return http.StatusCreated
		}
		return http.StatusOK
	}
	return http.StatusMultiStatus
}

// --- Bulk Entity Handlers ---

// bulkCreateEntitiesHandler handles requests to create multiple entities in bulk.
func (a *API) bulkCreateEntitiesHandler(c *gin.Context) {
	var req BulkCreateEntitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	if len(req.Entities) == 0 {
		c.JSON(http.StatusOK, BulkOperationResponse{Results: []BulkOperationResultItem{}})
		return
	}

	results, err := a.store.BulkCreateEntities(req.Entities)
	if err != nil {
		// This would be an unexpected store-level error, not individual item errors
		handleAPIError(c, http.StatusInternalServerError, "Failed to process bulk create entities: "+err.Error())
		return
	}

	responseStatus := determineBulkResponseStatus(results, "create")
	c.JSON(responseStatus, BulkOperationResponse{Results: results})
}

// bulkUpdateEntitiesHandler handles requests to update multiple entities in bulk.
func (a *API) bulkUpdateEntitiesHandler(c *gin.Context) {
	var req BulkUpdateEntitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	if len(req.Entities) == 0 {
		c.JSON(http.StatusOK, BulkOperationResponse{Results: []BulkOperationResultItem{}})
		return
	}

	results, err := a.store.BulkUpdateEntities(req.Entities)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to process bulk update entities: "+err.Error())
		return
	}

	responseStatus := determineBulkResponseStatus(results, "update")
	c.JSON(responseStatus, BulkOperationResponse{Results: results})
}

// bulkDeleteEntitiesHandler handles requests to delete multiple entities in bulk.
func (a *API) bulkDeleteEntitiesHandler(c *gin.Context) {
	var req BulkDeleteEntitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	if len(req.EntityIDs) == 0 {
		c.JSON(http.StatusOK, BulkOperationResponse{Results: []BulkOperationResultItem{}})
		return
	}

	results, err := a.store.BulkDeleteEntities(req.EntityIDs)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to process bulk delete entities: "+err.Error())
		return
	}

	responseStatus := determineBulkResponseStatus(results, "delete")
	c.JSON(responseStatus, BulkOperationResponse{Results: results})
}

// --- EntityRelationshipDefinition Handlers ---

func (a *API) createEntityRelationshipHandler(c *gin.Context) {
	var req EntityRelationshipDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""

	// Basic validation for RelationshipType
	switch req.RelationshipType {
	case OneToOne, OneToMany, ManyToOne:
		// valid
	default:
		handleAPIError(c, http.StatusBadRequest, "Invalid relationship_type. Must be ONE_TO_ONE, ONE_TO_MANY, or MANY_TO_ONE")
		return
	}

	// TODO: Add validation to check if Source/Target Entity/Attribute IDs exist
	// This requires querying the store for those entities/attributes.
	// For now, we rely on DB foreign key constraints.

	er, err := a.store.CreateEntityRelationship(req)
	if err != nil {
		// Refactored handleStoreError now uses handleAPIError
		handleStoreError(c, err, "EntityRelationship")
		return
	}
	c.JSON(http.StatusCreated, er)
}

func (a *API) listEntityRelationshipsHandler(c *gin.Context) {
	sourceEntityID := c.Query("source_entity_id")
	// Placeholder for actual pagination parameters (offset, limit)
	// offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	// limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr) // Use local or models.DefaultLimitStr

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	filters := make(map[string]interface{})
	if sourceEntityID != "" {
		filters["source_entity_id"] = sourceEntityID
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: filters}

	relationships, total, err := a.store.ListEntityRelationships(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list entity relationships: "+err.Error())
		return
	}

	response := ListResponse{Data: relationships, Total: total}
	c.JSON(http.StatusOK, response)
}

func (a *API) getEntityRelationshipHandler(c *gin.Context) {
	erID := c.Param("id")
	er, err := a.store.GetEntityRelationship(erID)
	if err != nil {
		handleStoreError(c, err, "EntityRelationship")
		return
	}
	c.JSON(http.StatusOK, er)
}

func (a *API) updateEntityRelationshipHandler(c *gin.Context) {
	erID := c.Param("id")
	var req EntityRelationshipDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Basic validation for RelationshipType
	switch req.RelationshipType {
	case OneToOne, OneToMany, ManyToOne:
		// valid
	default:
		handleAPIError(c, http.StatusBadRequest, "Invalid relationship_type. Must be ONE_TO_ONE, ONE_TO_MANY, or MANY_TO_ONE")
		return
	}

	// TODO: Add validation for IDs if necessary

	er, err := a.store.UpdateEntityRelationship(erID, req)
	if err != nil {
		// Refactored handleStoreError now uses handleAPIError
		handleStoreError(c, err, "EntityRelationship")
		return
	}
	c.JSON(http.StatusOK, er)
}

func (a *API) deleteEntityRelationshipHandler(c *gin.Context) {
	erID := c.Param("id")
	err := a.store.DeleteEntityRelationship(erID)
	if err != nil {
		handleStoreError(c, err, "EntityRelationship")
		return
	}
	c.JSON(http.StatusNoContent, nil)
}


// --- ScheduleDefinition Handlers ---

func (a *API) createScheduleDefinitionHandler(c *gin.Context) {
	var req ScheduleDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""

	schedule, err := a.store.CreateScheduleDefinition(req)
	if err != nil {
		// Using handleStoreError which now calls handleAPIError
		handleStoreError(c, err, "ScheduleDefinition")
		return
	}
	c.JSON(http.StatusCreated, schedule)
}

func (a *API) listScheduleDefinitionsHandler(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	schedules, total, err := a.store.ListScheduleDefinitions(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list schedules: "+err.Error())
		return
	}

	response := ListResponse{Data: schedules, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	workflows, total, err := a.store.ListWorkflowDefinitions(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list workflow definitions: "+err.Error())
		return
	}

	response := ListResponse{Data: workflows, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	templates, total, err := a.store.ListActionTemplates(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list action templates: "+err.Error())
		return
	}

	response := ListResponse{Data: templates, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		handleAPIError(c, http.StatusNotFound, "Entity not found")
		return
	}

	attribute, err := a.store.CreateAttribute(entityID, req.Name, req.DataType, req.Description, req.IsFilterable, req.IsPii, req.IsIndexed)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to create attribute: "+err.Error())
		return
	}
	c.JSON(http.StatusCreated, attribute)
}

// listAttributesHandler handles requests to list all attributes for an entity.
func (a *API) listAttributesHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		handleAPIError(c, http.StatusNotFound, "Entity not found")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	attributes, total, err := a.store.ListAttributes(entityID, params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list attributes for entity "+entityID+": "+err.Error())
		return
	}
	response := ListResponse{Data: attributes, Total: total}
	c.JSON(http.StatusOK, response)
}

// getAttributeHandler handles requests to get a specific attribute by ID.
func (a *API) getAttributeHandler(c *gin.Context) {
	entityID := c.Param("entity_id")
	attributeID := c.Param("attribute_id")

	// Check if entity exists
	if _, ok := a.store.GetEntity(entityID); !ok {
		handleAPIError(c, http.StatusNotFound, "Entity not found")
		return
	}

	attribute, ok := a.store.GetAttribute(entityID, attributeID)
	if !ok {
		handleAPIError(c, http.StatusNotFound, "Attribute not found")
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Check if entity exists first
	if _, ok := a.store.GetEntity(entityID); !ok {
		handleAPIError(c, http.StatusNotFound, "Entity not found")
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
		handleAPIError(c, http.StatusNotFound, "Entity not found")
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""
	// Ensure EntityID is passed if provided in the request
	// The store.CreateDataSource function now expects DataSourceConfig object
	// which includes EntityID.

	ds, err := a.store.CreateDataSource(req)
	if err != nil {
		// Using handleStoreError which now calls handleAPIError
		handleStoreError(c, err, "DataSource")
		return
	}
	c.JSON(http.StatusCreated, ds)
}

func (a *API) listDataSourcesHandler(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	sources, total, err := a.store.GetDataSources(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list data sources: "+err.Error())
		return
	}
	response := ListResponse{Data: sources, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// Ensure SourceID from path matches payload, or set it from path
	if req.SourceID != "" && req.SourceID != sourceID {
		handleAPIError(c, http.StatusBadRequest, "SourceID in path and payload do not match")
		return
	}
	req.SourceID = sourceID
	req.ID = "" // ID is set by the store

	mapping, err := a.store.CreateFieldMapping(req)
	if err != nil {
		// Using handleStoreError which now calls handleAPIError
		handleStoreError(c, err, "FieldMapping")
		return
	}
	c.JSON(http.StatusCreated, mapping)
}

func (a *API) listFieldMappingsHandler(c *gin.Context) {
	sourceID := c.Param("source_id")
	// Check if data source exists
	if _, err := a.store.GetDataSource(sourceID); err != nil {
		handleStoreError(c, err, "Data Source") // This already uses handleAPIError
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	mappings, total, err := a.store.GetFieldMappings(sourceID, params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list field mappings for source "+sourceID+": "+err.Error())
		return
	}
	response := ListResponse{Data: mappings, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// Ensure SourceID from path matches payload, or set it from path
	if req.SourceID != "" && req.SourceID != sourceID {
		handleAPIError(c, http.StatusBadRequest, "SourceID in path and payload do not match")
		return
	}
	req.SourceID = sourceID

	mapping, err := a.store.UpdateFieldMapping(sourceID, mappingID, req)
	if err != nil {
		// Using handleStoreError which now calls handleAPIError
		handleStoreError(c, err, "FieldMapping")
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
// It now calls handleAPIError for standardized error responses.
func handleStoreError(c *gin.Context, err error, resourceName string) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") {
		handleAPIError(c, http.StatusNotFound, fmt.Sprintf("%s not found: %s", resourceName, errMsg))
	} else if strings.Contains(errMsg, "cannot be empty") || strings.Contains(errMsg, "violates foreign key constraint") || strings.Contains(errMsg, "unique constraint") || strings.Contains(errMsg, "already exists") {
		handleAPIError(c, http.StatusBadRequest, fmt.Sprintf("Invalid input for %s: %s", resourceName, errMsg))
	} else {
		handleAPIError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to process %s: %s", strings.ToLower(resourceName), errMsg))
	}
}

// --- GroupDefinition Handlers ---

// createGroupDefinitionHandler handles requests to create a new group definition.
func (a *API) createGroupDefinitionHandler(c *gin.Context) {
	var req GroupDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// ID, CreatedAt, UpdatedAt are set by the store
	req.ID = ""

	groupDef, err := a.store.CreateGroupDefinition(req)
	if err != nil {
		// Using handleStoreError which now calls handleAPIError
		handleStoreError(c, err, "GroupDefinition")
		return
	}
	c.JSON(http.StatusCreated, groupDef)
}

// listGroupDefinitionsHandler handles requests to list all group definitions.
func (a *API) listGroupDefinitionsHandler(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", DefaultLimitStr)

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid offset parameter. Must be a non-negative integer.")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		handleAPIError(c, http.StatusBadRequest, "Invalid limit parameter. Must be a positive integer.")
		return
	}

	params := ListParams{Offset: offset, Limit: limit, Filters: make(map[string]interface{})}

	groupDefs, total, err := a.store.ListGroupDefinitions(params)
	if err != nil {
		handleAPIError(c, http.StatusInternalServerError, "Failed to list group definitions: "+err.Error())
		return
	}
	response := ListResponse{Data: groupDefs, Total: total}
	c.JSON(http.StatusOK, response)
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
		handleAPIError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
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
