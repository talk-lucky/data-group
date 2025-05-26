package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"metadata-service/internal/database"
	"metadata-service/internal/models"
)

var testDB *gorm.DB
var router *gin.Engine

// TestMain sets up the test database and router, runs tests, and then tears down.
func TestMain(m *testing.M) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Setup Test Database
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	err = testDB.AutoMigrate(&models.EntityDefinition{}, &models.AttributeDefinition{})
	if err != nil {
		log.Fatalf("Failed to migrate test database schema: %v", err)
	}
	database.DB = testDB

	// Setup Router
	router = gin.Default()
	v1 := router.Group("/api/v1")
	{
		entityRoutes := v1.Group("/entities")
		{
			entityRoutes.POST("/", CreateEntity)
			entityRoutes.GET("/", ListEntities)
			entityRoutes.GET("/:id", GetEntity)
			entityRoutes.PUT("/:id", UpdateEntity)
			entityRoutes.DELETE("/:id", DeleteEntity)

			// Routes for attributes specific to an entity (as defined in main.go)
			entityRoutes.POST("/:id/attributes", CreateAttribute) // Changed :entity_id to :id
			entityRoutes.GET("/:id/attributes", ListAttributes)   // Changed :entity_id to :id
		}

		// Standalone routes for attributes (as defined in main.go)
		attributeRoutes := v1.Group("/attributes")
		{
			attributeRoutes.GET("/:attribute_id", GetAttribute)
			attributeRoutes.PUT("/:attribute_id", UpdateAttribute)
			attributeRoutes.DELETE("/:attribute_id", DeleteAttribute)
		}
	}

	exitCode := m.Run()

	sqlDB, err := testDB.DB()
	if err == nil {
		sqlDB.Close()
	} else {
		log.Printf("Error getting DB for teardown: %v", err)
	}
	os.Exit(exitCode)
}

func clearTable() {
	if err := testDB.Exec("DELETE FROM entity_definitions").Error; err != nil {
		log.Fatalf("Failed to clear entity_definitions table: %v", err)
	}
}

func TestCreateEntity(t *testing.T) {
	clearTable()
	payload := models.CreateEntityRequest{Name: "Test Entity", Description: "A test entity"}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/entities/", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var entity models.EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, "Test Entity", entity.Name)
	assert.Equal(t, "A test entity", entity.Description)
	assert.NotEqual(t, uuid.Nil, entity.ID, "ID should not be Nil")
}

func TestCreateEntity_MissingName(t *testing.T) {
	clearTable()
	payload := models.CreateEntityRequest{Description: "A test entity without name"}
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/entities/", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request payload")
}

func TestListEntities(t *testing.T) {
	clearTable()
	testDB.Create(&models.EntityDefinition{ID: uuid.New(), Name: "Entity 1", Description: "Desc 1"})
	testDB.Create(&models.EntityDefinition{ID: uuid.New(), Name: "Entity 2", Description: "Desc 2"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/entities/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entities []models.EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entities)
	assert.NoError(t, err)
	assert.Len(t, entities, 2, "Should list two entities")
}

func TestGetEntity(t *testing.T) {
	clearTable()
	entityID := uuid.New()
	createdEntity := models.EntityDefinition{ID: entityID, Name: "Specific Entity", Description: "To be fetched"}
	testDB.Create(&createdEntity)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/entities/%s", entityID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entity models.EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, createdEntity.Name, entity.Name)
	assert.Equal(t, entityID, entity.ID)
}

func TestGetEntity_NotFound(t *testing.T) {
	clearTable()
	nonExistentID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/entities/%s", nonExistentID.String()), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetEntity_InvalidUUID(t *testing.T) {
	clearTable()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/entities/not-a-uuid", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateEntity(t *testing.T) {
	clearTable()
	entityID := uuid.New()
	entityToUpdate := models.EntityDefinition{ID: entityID, Name: "Old Name", Description: "Old Desc"}
	testDB.Create(&entityToUpdate)

	updatePayload := models.UpdateEntityRequest{Name: stringPtr("New Name"), Description: stringPtr("New Desc")}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/entities/%s", entityID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updatedEntity models.EntityDefinition
	testDB.First(&updatedEntity, "id = ?", entityID) // Fetch from DB to confirm update
	assert.Equal(t, "New Name", updatedEntity.Name)
	assert.Equal(t, "New Desc", updatedEntity.Description)
}

func TestUpdateEntity_PartialUpdateNameOnly(t *testing.T) {
	clearTable()
	entityID := uuid.New()
	entityToUpdate := models.EntityDefinition{ID: entityID, Name: "Original Name", Description: "Original Description"}
	testDB.Create(&entityToUpdate)

	updatePayload := models.UpdateEntityRequest{Name: stringPtr("Updated Name Only")}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/entities/%s", entityID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updatedEntity models.EntityDefinition
	testDB.First(&updatedEntity, "id = ?", entityID) // Fetch from DB
	assert.Equal(t, "Updated Name Only", updatedEntity.Name)
	assert.Equal(t, "Original Description", updatedEntity.Description)
}

func TestUpdateEntity_NotFound(t *testing.T) {
	clearTable()
	nonExistentID := uuid.New()
	updatePayload := models.UpdateEntityRequest{Name: stringPtr("New Name")}
	jsonPayload, _ := json.Marshal(updatePayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/entities/%s", nonExistentID.String()), bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEntity(t *testing.T) {
	clearTable()
	entityID := uuid.New()
	entityToDelete := models.EntityDefinition{ID: entityID, Name: "To Be Deleted", Description: "Delete me"}
	testDB.Create(&entityToDelete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/entities/%s", entityID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	var count int64
	testDB.Model(&models.EntityDefinition{}).Where("id = ?", entityID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteEntity_NotFound(t *testing.T) {
	clearTable()
	nonExistentID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/entities/%s", nonExistentID.String()), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func stringPtr(s string) *string {
	return &s
}

// Helper to convert bool to *bool
func boolPtr(b bool) *bool {
	return &b
}
