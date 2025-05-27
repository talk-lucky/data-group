package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	// testDB and router are already set up in entity_handler_test.go's TestMain.
	// We rely on that setup for these tests as well.
	"metadata-service/internal/models"
)

// Helper to create a dummy entity for attribute tests
func createTestEntityForAttributes(t *testing.T) models.EntityDefinition {
	entity := models.EntityDefinition{
		ID:   uuid.New(),
		Name: "TestEntityForAttributes_" + uuid.NewString()[:8],
	}
	result := testDB.Create(&entity)
	assert.NoError(t, result.Error)
	return entity
}

// Clear attributes table - TestMain in entity_handler_test.go clears entity_definitions
func clearAttributeTable() {
	if err := testDB.Exec("DELETE FROM attribute_definitions").Error; err != nil {
		// log.Fatalf("Failed to clear attribute_definitions table: %v", err)
		// Allow tests to continue if this fails, but log it.
		fmt.Printf("Warning: Failed to clear attribute_definitions table: %v\n", err)
	}
}


func TestCreateAttribute(t *testing.T) {
	clearTable()        // Clear entity definitions
	clearAttributeTable() // Clear attributes
	parentEntity := createTestEntityForAttributes(t)

	payload := models.CreateAttributeRequest{
		Name:         "TestAttribute",
		DataType:     "STRING",
		Description:  "A test attribute",
		IsFilterable: boolPtr(true), // Now using local boolPtr
		IsPII:        boolPtr(false),// Now using local boolPtr
	}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/entities/%s/attributes", parentEntity.ID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var attr models.AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attr)
	assert.NoError(t, err)
	assert.Equal(t, "TestAttribute", attr.Name)
	assert.Equal(t, "STRING", attr.DataType)
	assert.Equal(t, parentEntity.ID, attr.EntityID)
	assert.True(t, attr.IsFilterable)
	assert.False(t, attr.IsPII)
	assert.NotEqual(t, uuid.Nil, attr.ID)
}

func TestCreateAttribute_InvalidEntityID(t *testing.T) {
	clearTable()
	clearAttributeTable()
	payload := models.CreateAttributeRequest{Name: "TestAttribute", DataType: "STRING"}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/entities/not-a-uuid/attributes", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAttribute_EntityNotFound(t *testing.T) {
	clearTable()
	clearAttributeTable()
	nonExistentEntityID := uuid.New()
	payload := models.CreateAttributeRequest{Name: "TestAttribute", DataType: "STRING"}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/entities/%s/attributes", nonExistentEntityID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateAttribute_InvalidDataType(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	payload := models.CreateAttributeRequest{Name: "TestAttribute", DataType: "INVALID_TYPE"}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/entities/%s/attributes", parentEntity.ID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code) // due to binding oneof
}

func TestCreateAttribute_DuplicateNameForEntity(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)

	attr1 := models.AttributeDefinition{ID: uuid.New(), EntityID: parentEntity.ID, Name: "UniqueName", DataType: "STRING"}
	testDB.Create(&attr1)

	payload := models.CreateAttributeRequest{Name: "UniqueName", DataType: "INTEGER"} // Same name
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/entities/%s/attributes", parentEntity.ID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// GORM/SQLite might return a generic error for unique constraint violations.
	// In a real setup with PostgreSQL, you'd get a more specific error code.
	// For SQLite, this often results in a general server error if not handled specifically by GORM.
	// The current handler returns 500 for general DB errors during create.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAttributes(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	attr1 := models.AttributeDefinition{ID: uuid.New(), EntityID: parentEntity.ID, Name: "Attr1", DataType: "STRING"}
	attr2 := models.AttributeDefinition{ID: uuid.New(), EntityID: parentEntity.ID, Name: "Attr2", DataType: "INTEGER"}
	testDB.Create(&attr1)
	testDB.Create(&attr2)

	// Create another entity and its attribute to ensure we only get attributes for parentEntity
	otherEntity := createTestEntityForAttributes(t)
	testDB.Create(&models.AttributeDefinition{ID: uuid.New(), EntityID: otherEntity.ID, Name: "AttrOther", DataType: "BOOLEAN"})


	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/entities/%s/attributes", parentEntity.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var attributes []models.AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attributes)
	assert.NoError(t, err)
	assert.Len(t, attributes, 2)
	// Check if names match (order might vary)
	names := []string{attributes[0].Name, attributes[1].Name}
	assert.Contains(t, names, "Attr1")
	assert.Contains(t, names, "Attr2")
}

func TestListAttributes_EntityNotFound(t *testing.T) {
	clearTable()
	clearAttributeTable()
	nonExistentEntityID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/entities/%s/attributes", nonExistentEntityID.String()), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAttribute(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	attr := models.AttributeDefinition{
		ID:       uuid.New(),
		EntityID: parentEntity.ID,
		Name:     "SpecificAttribute",
		DataType: "DATETIME",
	}
	testDB.Create(&attr)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/attributes/%s", attr.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedAttr models.AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &fetchedAttr)
	assert.NoError(t, err)
	assert.Equal(t, attr.Name, fetchedAttr.Name)
	assert.Equal(t, attr.ID, fetchedAttr.ID)
	assert.Equal(t, attr.EntityID, fetchedAttr.EntityID)
}

func TestGetAttribute_NotFound(t *testing.T) {
	clearTable()
	clearAttributeTable()
	nonExistentAttrID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/attributes/%s", nonExistentAttrID.String()), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAttribute(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	attr := models.AttributeDefinition{
		ID:           uuid.New(),
		EntityID:     parentEntity.ID,
		Name:         "OldName",
		DataType:     "STRING",
		IsFilterable: false,
	}
	testDB.Create(&attr)

	newName := "NewName"
	newDataType := "INTEGER"
	newIsFilterable := true
	updatePayload := models.UpdateAttributeRequest{
		Name:         &newName,
		DataType:     &newDataType,
		IsFilterable: &newIsFilterable,
	}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/attributes/%s", attr.ID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updatedAttr models.AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &updatedAttr)
	assert.NoError(t, err)
	assert.Equal(t, newName, updatedAttr.Name)
	assert.Equal(t, newDataType, updatedAttr.DataType)
	assert.Equal(t, newIsFilterable, updatedAttr.IsFilterable)
	assert.Equal(t, attr.EntityID, updatedAttr.EntityID) // EntityID should not change
}

func TestUpdateAttribute_InvalidDataType(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	attr := models.AttributeDefinition{ID: uuid.New(), EntityID: parentEntity.ID, Name: "AttrToUpdate", DataType: "STRING"}
	testDB.Create(&attr)

	invalidDataType := "SUPER_STRING"
	updatePayload := models.UpdateAttributeRequest{DataType: &invalidDataType}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/attributes/%s", attr.ID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code) // Custom validation in handler for oneof
}

func TestUpdateAttribute_NotFound(t *testing.T) {
	clearTable()
	clearAttributeTable()
	nonExistentAttrID := uuid.New()
	name := "some name"
	updatePayload := models.UpdateAttributeRequest{Name: &name}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/attributes/%s", nonExistentAttrID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAttribute(t *testing.T) {
	clearTable()
	clearAttributeTable()
	parentEntity := createTestEntityForAttributes(t)
	attr := models.AttributeDefinition{ID: uuid.New(), EntityID: parentEntity.ID, Name: "ToDelete", DataType: "BOOLEAN"}
	testDB.Create(&attr)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/attributes/%s", attr.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's actually deleted
	var count int64
	testDB.Model(&models.AttributeDefinition{}).Where("id = ?", attr.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteAttribute_NotFound(t *testing.T) {
	clearTable()
	clearAttributeTable()
	nonExistentAttrID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/attributes/%s", nonExistentAttrID.String()), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// boolPtr is already defined in entity_handler_test.go, if this file was standalone, it would be needed.
// func boolPtr(b bool) *bool { return &b }
// stringPtr is also defined in entity_handler_test.go
// func stringPtr(s string) *string { return &s }

// Note: TestMain from entity_handler_test.go is assumed to handle setup/teardown.
