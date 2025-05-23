package metadata

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Helper function to setup router for tests
func setupRouter() (*gin.Engine, *Store) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := NewStore()
	api := NewAPI(store)
	api.RegisterRoutes(router)
	return router, store
}

// TestCreateEntityHandler tests the POST /api/v1/entities endpoint.
func TestCreateEntityHandler(t *testing.T) {
	router, _ := setupRouter()

	// Test successful creation
	payload := []byte(`{"name": "Test Entity", "description": "A test entity"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/entities/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.NotEmpty(t, entity.ID)
	assert.Equal(t, "Test Entity", entity.Name)
	assert.Equal(t, "A test entity", entity.Description)
	assert.NotZero(t, entity.CreatedAt)
	assert.NotZero(t, entity.UpdatedAt)

	// Test missing name
	payload = []byte(`{"description": "Another test entity"}`)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/entities/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListEntitiesHandler tests the GET /api/v1/entities endpoint.
func TestListEntitiesHandler(t *testing.T) {
	router, store := setupRouter()

	// Create some entities
	_, _ = store.CreateEntity("Entity 1", "Desc 1")
	_, _ = store.CreateEntity("Entity 2", "Desc 2")

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/entities/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entities []EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entities)
	assert.NoError(t, err)
	assert.Len(t, entities, 2)
}

// TestGetEntityHandler tests the GET /api/v1/entities/{entity_id} endpoint.
func TestGetEntityHandler(t *testing.T) {
	router, store := setupRouter()

	// Create an entity
	createdEntity, _ := store.CreateEntity("Test Entity", "Desc")

	// Test get existing entity
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/entities/"+createdEntity.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, createdEntity.ID, entity.ID)

	// Test get non-existent entity
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/entities/nonexistentid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateEntityHandler tests the PUT /api/v1/entities/{entity_id} endpoint.
func TestUpdateEntityHandler(t *testing.T) {
	router, store := setupRouter()

	// Create an entity
	createdEntity, _ := store.CreateEntity("Old Name", "Old Desc")

	// Test successful update
	payload := []byte(`{"name": "New Name", "description": "New Desc"}`)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/entities/"+createdEntity.ID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", entity.Name)
	assert.Equal(t, "New Desc", entity.Description)
	assert.True(t, entity.UpdatedAt.After(createdEntity.UpdatedAt))

	// Test update non-existent entity
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/entities/nonexistentid", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update with missing name
	payload = []byte(`{"description": "Only Desc"}`)
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/entities/"+createdEntity.ID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDeleteEntityHandler tests the DELETE /api/v1/entities/{entity_id} endpoint.
func TestDeleteEntityHandler(t *testing.T) {
	router, store := setupRouter()

	// Create an entity
	createdEntity, _ := store.CreateEntity("To Be Deleted", "Desc")
	// Add an attribute to it to test cascading delete
	_, _ = store.CreateAttribute(createdEntity.ID, "Attr1", "string", "Test Attr")


	// Test delete existing entity
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/entities/"+createdEntity.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, ok := store.GetEntity(createdEntity.ID)
	assert.False(t, ok)
	attrs, _ := store.ListAttributes(createdEntity.ID) // This should error or be empty
	assert.Empty(t, attrs) // Check if attributes are also deleted


	// Test delete non-existent entity
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/entities/nonexistentid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCreateAttributeHandler tests POST /api/v1/entities/{entity_id}/attributes
func TestCreateAttributeHandler(t *testing.T) {
	router, store := setupRouter()
	entity, _ := store.CreateEntity("Test Entity For Attr", "Entity Desc")

	// Test successful creation
	payload := []byte(`{"name": "Test Attr", "data_type": "string", "description": "A test attribute"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/entities/"+entity.ID+"/attributes/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var attr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attr)
	assert.NoError(t, err)
	assert.NotEmpty(t, attr.ID)
	assert.Equal(t, entity.ID, attr.EntityID)
	assert.Equal(t, "Test Attr", attr.Name)
	assert.Equal(t, "string", attr.DataType)

	// Test with non-existent entity ID
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/entities/nonexistententity/attributes/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)


	// Test missing name
	payload = []byte(`{"data_type": "string", "description": "A test attribute"}`)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/entities/"+entity.ID+"/attributes/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test missing data_type
	payload = []byte(`{"name": "Test Attr", "description": "A test attribute"}`)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/entities/"+entity.ID+"/attributes/", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListAttributesHandler tests GET /api/v1/entities/{entity_id}/attributes
func TestListAttributesHandler(t *testing.T) {
	router, store := setupRouter()
	entity, _ := store.CreateEntity("Test Entity", "Desc")
	_, _ = store.CreateAttribute(entity.ID, "Attr1", "string", "Desc1")
	_, _ = store.CreateAttribute(entity.ID, "Attr2", "integer", "Desc2")

	// Test list attributes for existing entity
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/entities/"+entity.ID+"/attributes/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var attrs []AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attrs)
	assert.NoError(t, err)
	assert.Len(t, attrs, 2)

	// Test list attributes for non-existent entity
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/entities/nonexistententity/attributes/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetAttributeHandler tests GET /api/v1/entities/{entity_id}/attributes/{attribute_id}
func TestGetAttributeHandler(t *testing.T) {
	router, store := setupRouter()
	entity, _ := store.CreateEntity("Test Entity", "Desc")
	attr, _ := store.CreateAttribute(entity.ID, "TestAttr", "string", "Desc")

	// Test get existing attribute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &fetchedAttr)
	assert.NoError(t, err)
	assert.Equal(t, attr.ID, fetchedAttr.ID)

	// Test get attribute for non-existent entity
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/entities/nonexistententity/attributes/"+attr.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test get non-existent attribute for existing entity
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/entities/"+entity.ID+"/attributes/nonexistentattr", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateAttributeHandler tests PUT /api/v1/entities/{entity_id}/attributes/{attribute_id}
func TestUpdateAttributeHandler(t *testing.T) {
	router, store := setupRouter()
	entity, _ := store.CreateEntity("Test Entity", "Desc")
	attr, _ := store.CreateAttribute(entity.ID, "Old Attr Name", "string", "Old Attr Desc")

	// Test successful update
	payload := []byte(`{"name": "New Attr Name", "data_type": "integer", "description": "New Attr Desc"}`)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updatedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &updatedAttr)
	assert.NoError(t, err)
	assert.Equal(t, "New Attr Name", updatedAttr.Name)
	assert.Equal(t, "integer", updatedAttr.DataType)
	assert.True(t, updatedAttr.UpdatedAt.After(attr.UpdatedAt))

	// Test update for non-existent entity
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/entities/nonexistententity/attributes/"+attr.ID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update for non-existent attribute
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/entities/"+entity.ID+"/attributes/nonexistentattr", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update with missing name
	payload = []byte(`{"data_type": "boolean", "description": "Only Desc"}`)
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDeleteAttributeHandler tests DELETE /api/v1/entities/{entity_id}/attributes/{attribute_id}
func TestDeleteAttributeHandler(t *testing.T) {
	router, store := setupRouter()
	entity, _ := store.CreateEntity("Test Entity", "Desc")
	attr, _ := store.CreateAttribute(entity.ID, "To Be Deleted Attr", "string", "Desc")

	// Test delete existing attribute
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, ok := store.GetAttribute(entity.ID, attr.ID)
	assert.False(t, ok)

	// Test delete for non-existent entity
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/entities/nonexistententity/attributes/"+attr.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test delete for non-existent attribute
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/entities/"+entity.ID+"/attributes/nonexistentattr", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
