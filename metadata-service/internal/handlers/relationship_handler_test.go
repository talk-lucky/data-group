package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"metadata-service/internal/database" // Used for direct DB interaction in tests
	"metadata-service/internal/models"
)

// testDB and router are assumed to be initialized in a TestMain function
// in another _test.go file in this package (e.g., entity_handler_test.go or handlers_test.go).
// If not, they would need to be initialized here.
// var testDB *gorm.DB
// var router *gin.Engine

// Helper to create a dummy entity for relationship tests
func createTestEntityForRelationships(t *testing.T, namePrefix string) models.EntityDefinition {
	entity := models.EntityDefinition{
		ID:   uuid.New(),
		Name: namePrefix + "_" + uuid.NewString()[:8],
	}
	// Assuming testDB is initialized and migrated from a TestMain
	result := testDB.Create(&entity)
	assert.NoError(t, result.Error)
	return entity
}

// Helper to create a dummy relationship for tests
func createTestRelationship(t *testing.T, sourceID, targetID uuid.UUID, relType string, name string) models.EntityRelationshipDefinition {
	relationship := models.EntityRelationshipDefinition{
		ID:               uuid.New(),
		Name:             name,
		SourceEntityID:   sourceID,
		TargetEntityID:   targetID,
		RelationshipType: relType,
		Description:      "Test description for " + name,
	}
	result := testDB.Create(&relationship)
	assert.NoError(t, result.Error)
	return relationship
}

// Clear relationships table
func clearRelationshipTable() {
	if err := testDB.Exec("DELETE FROM entity_relationship_definitions").Error; err != nil {
		fmt.Printf("Warning: Failed to clear entity_relationship_definitions table: %v\n", err)
	}
}

// Setup Gin context for testing
func setupTestRouter() *gin.Engine {
	// This function might not be necessary if 'router' is already globally initialized by a TestMain.
	// However, to ensure tests are self-contained in terms of router setup for these specific routes:
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		entityRoutes := v1.Group("/entities")
		{
			// Required for /entities/{id}/relationships route
			entityRoutes.GET("/:id/relationships", ListRelationships)
		}
		relationshipRoutes := v1.Group("/relationships")
		{
			relationshipRoutes.POST("/", CreateRelationship)
			relationshipRoutes.GET("/", ListRelationships)
			relationshipRoutes.GET("/:id", GetRelationship)
			relationshipRoutes.PUT("/:id", UpdateRelationship)
			relationshipRoutes.DELETE("/:id", DeleteRelationship)
		}
	}
	return r
}


func TestCreateRelationship(t *testing.T) {
	// Assuming TestMain in another file sets up DB and runs migrations
	// If not, call database.ConnectDatabase("sqlite::memory:") and AutoMigrate here.
	// For now, rely on the shared testDB and router.
	// If 'router' is not global, use router := setupTestRouter() for each test or test suite.

	clearTable() // Clear entity definitions (assuming this is from entity_handler_test.go)
	clearRelationshipTable()

	sourceEntity := createTestEntityForRelationships(t, "SourceEnt")
	targetEntity := createTestEntityForRelationships(t, "TargetEnt")

	t.Run("Success", func(t *testing.T) {
		clearRelationshipTable() // Ensure clean state for sub-test
		payload := models.CreateEntityRelationshipRequest{
			Name:             "TestRel1",
			SourceEntityID:   sourceEntity.ID.String(),
			TargetEntityID:   targetEntity.ID.String(),
			RelationshipType: "ONE_TO_ONE",
			Description:      "A test relationship",
		}
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req) // Use global router

		assert.Equal(t, http.StatusCreated, w.Code)
		var rel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rel)
		assert.NoError(t, err)
		assert.Equal(t, "TestRel1", rel.Name)
		assert.Equal(t, "ONE_TO_ONE", rel.RelationshipType)
		assert.Equal(t, sourceEntity.ID, rel.SourceEntityID)
		assert.Equal(t, targetEntity.ID, rel.TargetEntityID)
		assert.NotEqual(t, uuid.Nil, rel.ID)
	})

	t.Run("Invalid JSON payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing SourceEntityID", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{TargetEntityID: targetEntity.ID.String(), RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Binding required
	})
	
	t.Run("Missing TargetEntityID", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: sourceEntity.ID.String(), RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Binding required
	})

	t.Run("Missing RelationshipType", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: sourceEntity.ID.String(), TargetEntityID: targetEntity.ID.String()}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Binding required
	})

	t.Run("Invalid SourceEntityID format", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: "not-a-uuid", TargetEntityID: targetEntity.ID.String(), RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Handler validation for UUID format
	})

	t.Run("Invalid TargetEntityID format", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: sourceEntity.ID.String(), TargetEntityID: "not-a-uuid", RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Handler validation for UUID format
	})

	t.Run("SourceEntityID does not exist", func(t *testing.T) {
		nonExistentUUID := uuid.New().String()
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: nonExistentUUID, TargetEntityID: targetEntity.ID.String(), RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("TargetEntityID does not exist", func(t *testing.T) {
		nonExistentUUID := uuid.New().String()
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: sourceEntity.ID.String(), TargetEntityID: nonExistentUUID, RelationshipType: "ONE_TO_ONE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid RelationshipType", func(t *testing.T) {
		payload := models.CreateEntityRelationshipRequest{SourceEntityID: sourceEntity.ID.String(), TargetEntityID: targetEntity.ID.String(), RelationshipType: "INVALID_TYPE"}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		// The handler returns 422 UnprocessableEntity for this specific check
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("Duplicate relationship (name, source, target)", func(t *testing.T) {
		clearRelationshipTable()
		relName := "UniqueRelName"
		// Create one first
		createTestRelationship(t, sourceEntity.ID, targetEntity.ID, "ONE_TO_MANY", relName)

		payload := models.CreateEntityRelationshipRequest{
			Name:             relName, // Same name
			SourceEntityID:   sourceEntity.ID.String(),
			TargetEntityID:   targetEntity.ID.String(),
			RelationshipType: "ONE_TO_ONE", // Different type, but name+source+target is the constraint
		}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/relationships", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		// The handler returns 409 Conflict for unique constraint violations
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestListRelationships(t *testing.T) {
	clearTable()
	clearRelationshipTable()
	source1 := createTestEntityForRelationships(t, "Source1")
	target1 := createTestEntityForRelationships(t, "Target1")
	source2 := createTestEntityForRelationships(t, "Source2")
	target2 := createTestEntityForRelationships(t, "Target2")

	rel1 := createTestRelationship(t, source1.ID, target1.ID, "ONE_TO_ONE", "RelS1T1_OTO")
	rel2 := createTestRelationship(t, source1.ID, target2.ID, "ONE_TO_MANY", "RelS1T2_OTM")
	rel3 := createTestRelationship(t, source2.ID, target1.ID, "MANY_TO_MANY", "RelS2T1_MTM")

	testRouter := setupTestRouter() // Use a router that has all relationship routes, including /entities/:id/relationships

	t.Run("No relationships", func(t *testing.T) {
		clearRelationshipTable() // Clear before this specific sub-test
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships", nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 0)
		// Restore for other tests
		createTestRelationship(t, source1.ID, target1.ID, "ONE_TO_ONE", "RelS1T1_OTO")
		createTestRelationship(t, source1.ID, target2.ID, "ONE_TO_MANY", "RelS1T2_OTM")
		createTestRelationship(t, source2.ID, target1.ID, "MANY_TO_MANY", "RelS2T1_MTM")
	})

	t.Run("Multiple relationships exist", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships", nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 3) // rel1, rel2, rel3
	})

	t.Run("Filter by source_entity_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/relationships?source_entity_id=%s", source1.ID.String()), nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 2)
		for _, rel := range rels {
			assert.Equal(t, source1.ID, rel.SourceEntityID)
		}
	})

	t.Run("Filter by target_entity_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/relationships?target_entity_id=%s", target1.ID.String()), nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 2)
		for _, rel := range rels {
			assert.Equal(t, target1.ID, rel.TargetEntityID)
		}
	})

	t.Run("Filter by type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships?type=ONE_TO_MANY", nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 1)
		assert.Equal(t, rel2.ID, rels[0].ID)
		assert.Equal(t, "ONE_TO_MANY", rels[0].RelationshipType)
	})
	
	t.Run("Filter by type - case insensitive", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships?type=one_to_many", nil) // Lowercase
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 1)
		assert.Equal(t, rel2.ID, rels[0].ID)
	})

	t.Run("Filter by invalid type", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships?type=INVALID_TYPE", nil)
		testRouter.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Handler validation
	})

	t.Run("Filter by source_entity_id via /entities/{id}/relationships", func(t *testing.T) {
		w := httptest.NewRecorder()
		// Use the specific router setup for this test to ensure path parameter handling
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/entities/%s/relationships", source1.ID.String()), nil)
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rels []models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &rels)
		assert.NoError(t, err)
		assert.Len(t, rels, 2)
		for _, rel := range rels {
			assert.Equal(t, source1.ID, rel.SourceEntityID)
		}
		// Ensure it doesn't pick up rel3 which has source2
		ids := []uuid.UUID{rels[0].ID, rels[1].ID}
		assert.Contains(t, ids, rel1.ID)
		assert.Contains(t, ids, rel2.ID)
		assert.NotContains(t, ids, rel3.ID)
	})
}

func TestGetRelationship(t *testing.T) {
	clearTable()
	clearRelationshipTable()
	source := createTestEntityForRelationships(t, "SourceGet")
	target := createTestEntityForRelationships(t, "TargetGet")
	rel := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelToGet")

	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/relationships/%s", rel.ID.String()), nil)
		router.ServeHTTP(w, req) // Global router is fine here

		assert.Equal(t, http.StatusOK, w.Code)
		var fetchedRel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &fetchedRel)
		assert.NoError(t, err)
		assert.Equal(t, rel.ID, fetchedRel.ID)
		assert.Equal(t, rel.Name, fetchedRel.Name)
	})

	t.Run("Invalid relationship ID format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/relationships/not-a-uuid", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Relationship does not exist", func(t *testing.T) {
		nonExistentUUID := uuid.New().String()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/relationships/%s", nonExistentUUID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateRelationship(t *testing.T) {
	clearTable()
	clearRelationshipTable()
	source := createTestEntityForRelationships(t, "SourceUpd")
	target := createTestEntityForRelationships(t, "TargetUpd")
	
	t.Run("Success - Update Name", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "OriginalName")
		newName := "UpdatedName"
		payload := models.UpdateEntityRelationshipRequest{Name: &newName}
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updatedRel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &updatedRel)
		assert.NoError(t, err)
		assert.Equal(t, newName, updatedRel.Name)
		assert.Equal(t, relToUpdate.RelationshipType, updatedRel.RelationshipType) // Type should not change
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})

	t.Run("Success - Update Description", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelForDescUpdate")
		newDesc := "Updated Description"
		payload := models.UpdateEntityRelationshipRequest{Description: &newDesc}
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		var updatedRel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &updatedRel)
		assert.NoError(t, err)
		assert.Equal(t, newDesc, updatedRel.Description)
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})
	
	t.Run("Success - Update RelationshipType", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelForTypeUpdate")
		newType := "ONE_TO_MANY"
		payload := models.UpdateEntityRelationshipRequest{RelationshipType: &newType}
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updatedRel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &updatedRel)
		assert.NoError(t, err)
		assert.Equal(t, newType, updatedRel.RelationshipType)
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})

	t.Run("Success - Update All Updatable Fields", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelForAllUpdate")
		newName := "UpdatedNameAgain"
		newDesc := "Updated Description Again"
		newType := "MANY_TO_MANY"
		payload := models.UpdateEntityRelationshipRequest{Name: &newName, Description: &newDesc, RelationshipType: &newType}
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updatedRel models.EntityRelationshipDefinition
		err := json.Unmarshal(w.Body.Bytes(), &updatedRel)
		assert.NoError(t, err)
		assert.Equal(t, newName, updatedRel.Name)
		assert.Equal(t, newDesc, updatedRel.Description)
		assert.Equal(t, newType, updatedRel.RelationshipType)
		assert.Equal(t, relToUpdate.SourceEntityID, updatedRel.SourceEntityID) // Should not change
		assert.Equal(t, relToUpdate.TargetEntityID, updatedRel.TargetEntityID) // Should not change
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})
	
	t.Run("Invalid JSON payload", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelForBadJSON")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBufferString("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})

	t.Run("Invalid relationship ID format", func(t *testing.T) {
		name := "test"
		payload := models.UpdateEntityRelationshipRequest{Name: &name}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/relationships/not-a-uuid", bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Relationship does not exist", func(t *testing.T) {
		nonExistentUUID := uuid.New().String()
		name := "test"
		payload := models.UpdateEntityRelationshipRequest{Name: &name}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", nonExistentUUID), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid RelationshipType in payload", func(t *testing.T) {
		relToUpdate := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelForInvalidTypeUpdate")
		invalidType := "INVALID_TYPE"
		payload := models.UpdateEntityRelationshipRequest{RelationshipType: &invalidType}
		jsonPayload, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", relToUpdate.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code) // Handler validation for enum
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ?", relToUpdate.ID) // Clean up
	})

	t.Run("Update causing duplicate name", func(t *testing.T) {
		clearRelationshipTable()
		rel1 := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "UniqueName1")
		rel2 := createTestRelationship(t, source.ID, target.ID, "ONE_TO_MANY", "NameToTake") // Name we want to update rel1 to

		payload := models.UpdateEntityRelationshipRequest{Name: &rel2.Name} // Try to update rel1's name to rel2's name
		jsonPayload, _ := json.Marshal(payload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/relationships/%s", rel1.ID.String()), bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code) // Unique constraint (name, source_id, target_id)
		
		// Clean up for next tests
		testDB.Exec("DELETE FROM entity_relationship_definitions WHERE id = ? OR id = ?", rel1.ID, rel2.ID)
	})
}

func TestDeleteRelationship(t *testing.T) {
	clearTable()
	clearRelationshipTable()
	source := createTestEntityForRelationships(t, "SourceDel")
	target := createTestEntityForRelationships(t, "TargetDel")

	t.Run("Success", func(t *testing.T) {
		relToDelete := createTestRelationship(t, source.ID, target.ID, "ONE_TO_ONE", "RelToDelete")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/relationships/%s", relToDelete.ID.String()), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify it's actually deleted
		var count int64
		err := testDB.Model(&models.EntityRelationshipDefinition{}).Where("id = ?", relToDelete.ID).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Invalid relationship ID format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/relationships/not-a-uuid", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Relationship does not exist", func(t *testing.T) {
		nonExistentUUID := uuid.New().String()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/relationships/%s", nonExistentUUID), nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// It's assumed that entity_handler_test.go or a similar file in the same package
// defines TestMain, clearTable, boolPtr, stringPtr.
// If TestMain is not present or doesn't initialize the DB and router,
// the following would be needed:
/*
var testDB *gorm.DB
var router *gin.Engine

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	var err error
	testDB, err = database.ConnectTestDatabase() // Uses GORM to connect to sqlite::memory:
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// AutoMigrate the schema
	err = testDB.AutoMigrate(&models.EntityDefinition{}, &models.AttributeDefinition{}, &models.EntityRelationshipDefinition{})
	if err != nil {
		log.Fatalf("Failed to migrate test database: %v", err)
	}

	// Setup router (can be a global router or setup per test suite)
	router = gin.Default()
	// Register all routes that might be needed, or use setupTestRouter() in each TestXxx function.
    // This global router is used by attribute_handler_test.go as well.
    // Copied from main.go and adapted for test
    v1 := router.Group("/api/v1")
	{
		entityRoutes := v1.Group("/entities")
		{
			entityRoutes.POST("/", CreateEntity)
			entityRoutes.GET("/", ListEntities)
			entityRoutes.GET("/:id", GetEntity)
			entityRoutes.PUT("/:id", UpdateEntity)
			entityRoutes.DELETE("/:id", DeleteEntity)
			entityRoutes.POST("/:id/attributes", CreateAttribute)
			entityRoutes.GET("/:id/attributes", ListAttributes)
			entityRoutes.GET("/:id/relationships", ListRelationships) // For relationship tests
		}
		attributeRoutes := v1.Group("/attributes")
		{
			attributeRoutes.GET("/:attribute_id", GetAttribute)
			attributeRoutes.PUT("/:attribute_id", UpdateAttribute)
			attributeRoutes.DELETE("/:attribute_id", DeleteAttribute)
		}
		relationshipRoutes := v1.Group("/relationships")
		{
			relationshipRoutes.POST("/", CreateRelationship)
			relationshipRoutes.GET("/", ListRelationships)
			relationshipRoutes.GET("/:id", GetRelationship)
			relationshipRoutes.PUT("/:id", UpdateRelationship)
			relationshipRoutes.DELETE("/:id", DeleteRelationship)
		}
	}


	// Run tests
	exitVal := m.Run()

	// Teardown - close DB connection or clean up tables if necessary
	// For sqlite::memory:, connection close might be enough, or let it be handled by OS.
	// If using a file-based SQLite for testing, might want to delete it.
	sqlDB, _ := testDB.DB()
    sqlDB.Close()

	os.Exit(exitVal)
}

func clearTable() { // Clears entity_definitions
    if err := testDB.Exec("DELETE FROM entity_definitions").Error; err != nil {
        log.Fatalf("Failed to clear entity_definitions table: %v", err)
    }
}
// stringPtr and boolPtr would also be needed if not in another file.
*/

// The stringPtr and boolPtr helper functions are assumed to be available from
// another test file in the same package (e.g., entity_handler_test.go).
// If not, they would be defined as:
/*
func stringPtr(s string) *string {
    return &s
}

func boolPtr(b bool) *bool {
    return &b
}
*/

// Ensure database.GetDB() in handlers returns the testDB instance during tests.
// This is typically handled by replacing the global DB instance in the database package
// or by ensuring GetDB() is designed to be test-aware.
// The current structure of database.GetDB() seems to return a global variable,
// so ConnectTestDatabase() should set that global.
// The database.ConnectDatabase() function used in main.go initializes a global 'DB' var.
// A database.ConnectTestDatabase() would do similarly for the test DB.
// For this test file, we are *assuming* that 'testDB' (a *gorm.DB instance) and 'router' (*gin.Engine)
// are correctly initialized and available globally from a TestMain in entity_handler_test.go or handlers_test.go.
// And that 'clearTable()' is also available from there for clearing entity_definitions.
// The 'clearRelationshipTable()' is defined locally.
// 'createTestEntityForRelationships' is a local helper.
// 'createTestRelationship' is a local helper.
// 'setupTestRouter()' is a local helper for tests needing specific route setups like /entities/:id/relationships.

// A note on the global router vs. setupTestRouter():
// The attribute_handler_test.go seems to imply a global 'router' is initialized by a TestMain.
// I've used this global 'router' for most tests.
// For TestListRelationships, specifically the sub-test for "/entities/{id}/relationships",
// I've used 'testRouter := setupTestRouter()' to ensure that specific route is definitely configured,
// as the global router's setup is not visible in this file. If the global router is comprehensively
// configured in TestMain (as shown in the commented-out TestMain example), then using the global 'router'
// would be fine for all tests.
// The local setupTestRouter() ensures that if other tests (e.g. attribute tests) set up a more minimal
// global router, these relationship tests still work with all their required routes.
// The current handlers.CreateRelationship etc. are package-level functions.
// The database.GetDB() is assumed to be correctly pointing to the test DB.

```
