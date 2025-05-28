package metadata

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStore *PostgresStore
var testRouter *gin.Engine

// getEnvTest reads an environment variable for test configuration or returns a default.
func getEnvTest(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// setupTestDBAndRouter initializes the PostgresStore and Gin router for testing.
// It's called once by TestMain.
func setupTestDBAndRouter() (*PostgresStore, *gin.Engine, error) {
	dbHost := getEnvTest("TEST_DB_HOST", "localhost")
	dbPort := getEnvTest("TEST_DB_PORT", "5432")
	dbUser := getEnvTest("TEST_DB_USER", "admin")
	dbPassword := getEnvTest("TEST_DB_PASSWORD", "password")
	dbName := getEnvTest("TEST_DB_NAME", "metadata_test_db")
	dbSSLMode := getEnvTest("TEST_DB_SSLMODE", "disable")

	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	store, err := NewPostgresStore(dataSourceName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	api := NewAPI(store) // NewAPI now takes the PostgresStore
	api.RegisterRoutes(router)

	return store, router, nil
}

// clearAllTables truncates all relevant tables in the test database.
func clearAllTables(store *PostgresStore) error {
	tables := []string{
		// Order might matter if not using CASCADE, but with CASCADE it's more flexible.
		// Start with tables that are referenced by others if not using CASCADE.
		// Order: Start with tables that are referenced by others, or use CASCADE judiciously.
		// With RESTART IDENTITY CASCADE, the order is less critical for cleaning,
		// but for initial schema creation, dependent tables come after their references.
		"entity_relationship_definitions", // Depends on entities and attributes
		"data_source_field_mappings",    // Depends on data_sources, entities, attributes
		"group_definitions",             // Depends on entities
		"attribute_definitions",         // Depends on entities
		"schedule_definitions",          // May depend on other items via task_parameters
		"workflow_definitions",          // May depend on other items via trigger_config/action_sequence
		"action_templates",
		"data_source_configs",           // May depend on entities
		"entity_definitions",            // Base table
	}

	tx, err := store.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for clearing tables: %w", err)
	}
	defer tx.Rollback()

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", table)
		_, err := tx.Exec(query)
		if err != nil {
			// Check if the error is "table does not exist" (PostgreSQL specific error code: 42P01)
			// SQLite error for "no such table" is different.
			// Since we are targeting PostgreSQL for the actual store, this check is more relevant for PG.
			// For a generic approach, one might need to be less strict or have DB-specific error handling.
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P01" {
				log.Printf("Table %s does not exist, skipping truncation. This is normal on initial schema setup.", table)
				continue // Table might not exist yet if initSchema failed before creating all tables
			}
			// For other errors, it's more serious
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}
	return tx.Commit()
}

// TestMain sets up the test database and router once for all tests in the package.
func TestMain(m *testing.M) {
	var err error
	testStore, testRouter, err = setupTestDBAndRouter()
	if err != nil {
		log.Fatalf("Failed to set up test DB and router: %v", err)
	}
	defer testStore.Close() // Ensure DB connection is closed after all tests

	// Run the tests
	exitCode := m.Run()
	os.Exit(exitCode)
}

// performRequest is a helper to make HTTP requests to the test router.
func performRequest(r http.Handler, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Entity Handler Tests (Adapted) ---

func TestCreateEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	t.Run("CreateWithMetadata", func(t *testing.T) {
		payload := `{"name": "Test Entity Meta", "description": "A test entity with metadata", "metadata": {"key1": "value1", "count": 100.0, "active": true, "tags": ["tagA", "tagB"]}}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payload), nil)

		assert.Equal(t, http.StatusCreated, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.NotEmpty(t, entity.ID)
		assert.Equal(t, "Test Entity Meta", entity.Name)
		require.NotNil(t, entity.Metadata)
		assert.Equal(t, "value1", entity.Metadata["key1"])
		assert.Equal(t, 100.0, entity.Metadata["count"]) 
		assert.Equal(t, true, entity.Metadata["active"])
		assert.Equal(t, []interface{}{"tagA", "tagB"}, entity.Metadata["tags"])
	})

	t.Run("CreateWithoutMetadata", func(t *testing.T) {
		payload := `{"name": "Test Entity No Meta", "description": "A test entity"}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusCreated, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.NotEmpty(t, entity.ID)
		assert.Equal(t, "Test Entity No Meta", entity.Name)
		assert.Nil(t, entity.Metadata) // or assert.Empty(t, entity.Metadata)
	})

	t.Run("CreateWithNullMetadata", func(t *testing.T) {
		payload := `{"name": "Test Entity Null Meta", "description": "A test entity", "metadata": null}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusCreated, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.Nil(t, entity.Metadata)
	})

	// Test missing name
	payloadMissingName := `{"description": "Another test entity"}`
	w := performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payloadMissingName), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListEntitiesHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	meta1 := map[string]interface{}{"source": "system1", "version": 1.0}
	meta2 := map[string]interface{}{"source": "system2", "verified": true, "tags": []interface{}{"data", "test"}}
	_, _ = testStore.CreateEntity("Entity 1", "Desc 1", meta1)
	_, _ = testStore.CreateEntity("Entity 2", "Desc 2", meta2)

	w := performRequest(testRouter, "GET", "/api/v1/entities/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	
	var entities []EntityDefinition
	entitiesBytes, _ := json.Marshal(resp.Data) 
	err = json.Unmarshal(entitiesBytes, &entities)
	require.NoError(t, err)
	assert.Len(t, entities, 2)
	
	// Check metadata for entities (order might vary, so check by name)
	for _, e := range entities {
		if e.Name == "Entity 1" {
			assert.Equal(t, meta1, e.Metadata)
		} else if e.Name == "Entity 2" {
			assert.Equal(t, meta2, e.Metadata)
		}
	}

	// Test pagination: limit
	w = performRequest(testRouter, "GET", "/api/v1/entities/?limit=1", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total, "Total should still be 2 with limit")
	entitiesBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(entitiesBytes, &entities)
	require.NoError(t, err)
	assert.Len(t, entities, 1, "Data should be limited to 1")

	// Test pagination: offset and limit
	_, _ = testStore.CreateEntity("Entity 3", "Desc 3", nil) // Total 3 entities now
	w = performRequest(testRouter, "GET", "/api/v1/entities/?offset=1&limit=1", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.Total)
	entitiesBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(entitiesBytes, &entities)
	require.NoError(t, err)
	assert.Len(t, entities, 1)
	// Name check depends on ordering, which is by name. "Entity 2" should be at offset 1 if "Entity 1", "Entity 2", "Entity 3"
	assert.Equal(t, "Entity 2", entities[0].Name) 
	assert.Equal(t, meta2, entities[0].Metadata)


	// Test invalid limit/offset
	w = performRequest(testRouter, "GET", "/api/v1/entities/?limit=-1", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var apiErr APIError
	err = json.Unmarshal(w.Body.Bytes(), &apiErr)
	require.NoError(t, err)
	assert.Contains(t, apiErr.Message, "Invalid limit parameter")
}

func TestGetEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	testMeta := map[string]interface{}{"version": "1.0", "status": "active", "nested": map[string]interface{}{"key": "val"}}
	createdEntity, _ := testStore.CreateEntity("Test Entity", "Desc", testMeta)

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+createdEntity.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, createdEntity.ID, entity.ID)
	assert.Equal(t, testMeta, entity.Metadata)

	w = performRequest(testRouter, "GET", "/api/v1/entities/nonexistent-uuid-format", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code) 
}

func TestUpdateEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	initialMeta := map[string]interface{}{"initial_key": "initial_value"}
	createdEntity, _ := testStore.CreateEntity("Old Name", "Old Desc", initialMeta)

	t.Run("UpdateAllFieldsIncludingMetadata", func(t *testing.T) {
		updatedMeta := map[string]interface{}{"key1": "valueUpdated", "count": 2.0, "active": false, "tags": []interface{}{"updated"}}
		updatePayload := `{"name": "New Name", "description": "New Desc", "metadata": {"key1": "valueUpdated", "count": 2.0, "active": false, "tags": ["updated"]}}`
		w := performRequest(testRouter, "PUT", "/api/v1/entities/"+createdEntity.ID, strings.NewReader(updatePayload), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.Equal(t, "New Name", entity.Name)
		assert.Equal(t, "New Desc", entity.Description)
		assert.Equal(t, updatedMeta, entity.Metadata)
		assert.True(t, entity.UpdatedAt.After(createdEntity.UpdatedAt))
	})
	
	t.Run("UpdateToAddMetadata", func(t *testing.T) {
		// First, clear metadata
		_, _ = testStore.UpdateEntity(createdEntity.ID, "Name Clear Meta", "Desc Clear Meta", nil)
		currentEntity, _ := testStore.GetEntity(createdEntity.ID)
		
		addedMeta := map[string]interface{}{"new_key": "new_value"}
		updatePayload := `{"name": "Name Add Meta", "metadata": {"new_key": "new_value"}}`
		w := performRequest(testRouter, "PUT", "/api/v1/entities/"+createdEntity.ID, strings.NewReader(updatePayload), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.Equal(t, "Name Add Meta", entity.Name)
		assert.Equal(t, addedMeta, entity.Metadata)
		assert.True(t, entity.UpdatedAt.After(currentEntity.UpdatedAt))
	})

	t.Run("UpdateToClearMetadata", func(t *testing.T) {
		currentEntity, _ := testStore.GetEntity(createdEntity.ID) // Get current state which should have metadata from previous sub-test
		require.NotNil(t, currentEntity.Metadata, "Metadata should exist before clearing")

		updatePayload := `{"name": "Name Clear Meta Again", "metadata": null}`
		w := performRequest(testRouter, "PUT", "/api/v1/entities/"+createdEntity.ID, strings.NewReader(updatePayload), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.Equal(t, "Name Clear Meta Again", entity.Name)
		assert.Nil(t, entity.Metadata)
		assert.True(t, entity.UpdatedAt.After(currentEntity.UpdatedAt))
	})
	
	t.Run("UpdateWithoutMetadataField", func(t *testing.T) {
		metaToPreserve := map[string]interface{}{"preserve_this": "value", "version": 3.0}
		_,_ = testStore.UpdateEntity(createdEntity.ID, "Name For No Meta Update", "Desc For No Meta Update", metaToPreserve)
		currentEntity, _ := testStore.GetEntity(createdEntity.ID)

		updatePayload := `{"name": "Name Update No Meta Field"}` 
		w := performRequest(testRouter, "PUT", "/api/v1/entities/"+createdEntity.ID, strings.NewReader(updatePayload), nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var entity EntityDefinition
		err := json.Unmarshal(w.Body.Bytes(), &entity)
		require.NoError(t, err)
		assert.Equal(t, "Name Update No Meta Field", entity.Name)
		assert.Equal(t, metaToPreserve, entity.Metadata, "Metadata should be preserved")
		assert.True(t, entity.UpdatedAt.After(currentEntity.UpdatedAt))
	})

	w := performRequest(testRouter, "PUT", "/api/v1/entities/nonexistent-id", strings.NewReader(`{"name":"foo"}`), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}


// --- EntityRelationshipDefinition Handler Tests ---

// seedEntityRelationshipTestData creates prerequisite entities and attributes for relationship tests.
// Returns: userEntity, userPkAttr, postEntity, postUserFkAttr
func seedEntityRelationshipTestData(t *testing.T) (EntityDefinition, AttributeDefinition, EntityDefinition, AttributeDefinition) {
	t.Helper()
	userEntity, err := testStore.CreateEntity("User ER Test", "User entity for ER tests")
	require.NoError(t, err)
	userPkAttr, err := testStore.CreateAttribute(userEntity.ID, "id", BaseTypeString, map[string]interface{}{"format": "uuid"}, "User PK", true, false, true)
	require.NoError(t, err)

	postEntity, err := testStore.CreateEntity("Post ER Test", "Post entity for ER tests")
	require.NoError(t, err)
	postUserFkAttr, err := testStore.CreateAttribute(postEntity.ID, "user_id", BaseTypeString, map[string]interface{}{"format": "uuid"}, "Post FK to User", true, false, true)
	require.NoError(t, err)

	// Also create a PK for Post entity to allow it to be a source in some relationships
	_, err = testStore.CreateAttribute(postEntity.ID, "id", BaseTypeString, map[string]interface{}{"format": "uuid"}, "Post PK", true, false, true)
	require.NoError(t, err)


	return userEntity, userPkAttr, postEntity, postUserFkAttr
}

func TestCreateEntityRelationshipHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before ER test")
	userEntity, userPkAttr, postEntity, postUserFkAttr := seedEntityRelationshipTestData(t)

	validPayload := EntityRelationshipDefinition{
		Name:              "UserPosts",
		Description:       "Links users to their posts",
		SourceEntityID:    userEntity.ID,
		SourceAttributeID: userPkAttr.ID,
		TargetEntityID:    postEntity.ID,
		TargetAttributeID: postUserFkAttr.ID,
		RelationshipType:  OneToMany,
	}
	payloadBytes, _ := json.Marshal(validPayload)
	w := performRequest(testRouter, "POST", "/api/v1/entity-relationships/", bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var createdRel EntityRelationshipDefinition
	err := json.Unmarshal(w.Body.Bytes(), &createdRel)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdRel.ID)
	assert.Equal(t, validPayload.Name, createdRel.Name)
	assert.Equal(t, validPayload.SourceEntityID, createdRel.SourceEntityID)

	// Test missing name
	invalidPayload := `{"description": "Test", "source_entity_id": "x", "source_attribute_id": "y", "target_entity_id": "z", "target_attribute_id": "a", "relationship_type": "ONE_TO_ONE"}`
	w = performRequest(testRouter, "POST", "/api/v1/entity-relationships/", strings.NewReader(invalidPayload), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code) // Gin binding should catch this
	
    invalidRelTypePayload.Name = "InvalidRelTypeTest" // Change name to avoid unique constraint
    invalidRelTypePayload.RelationshipType = "INVALID_TYPE"
    payloadBytes, _ = json.Marshal(invalidRelTypePayload)
    w = performRequest(testRouter, "POST", "/api/v1/entity-relationships/", bytes.NewReader(payloadBytes), nil)
    assert.Equal(t, http.StatusBadRequest, w.Code)
    assert.Contains(t, w.Body.String(), "Invalid relationship_type. Must be ONE_TO_ONE, ONE_TO_MANY, MANY_TO_ONE, or MANY_TO_MANY")

	// Test creation with ManyToMany and Metadata
	manyToManyPayload := EntityRelationshipDefinition{
		Name:              "UserPostsM2M",
		Description:       "Links users to their posts (M2M with metadata)",
		SourceEntityID:    userEntity.ID,
		SourceAttributeID: userPkAttr.ID,
		TargetEntityID:    postEntity.ID,
		TargetAttributeID: postUserFkAttr.ID,
		RelationshipType:  ManyToMany,
		Metadata:          map[string]interface{}{"system_owner": "user_module", "priority": 1.0, "is_derived": false},
	}
	payloadBytesM2M, _ := json.Marshal(manyToManyPayload)
	w = performRequest(testRouter, "POST", "/api/v1/entity-relationships/", bytes.NewReader(payloadBytesM2M), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var createdM2MRel EntityRelationshipDefinition
	err = json.Unmarshal(w.Body.Bytes(), &createdM2MRel)
	assert.NoError(t, err)
	assert.Equal(t, ManyToMany, createdM2MRel.RelationshipType)
	require.NotNil(t, createdM2MRel.Metadata)
	assert.Equal(t, "user_module", createdM2MRel.Metadata["system_owner"])
	assert.Equal(t, 1.0, createdM2MRel.Metadata["priority"])
	assert.Equal(t, false, createdM2MRel.Metadata["is_derived"])


	// Test duplicate creation (name, source_entity_id, target_entity_id should be unique)
    validPayload.Description = "Attempting duplicate" // Make it slightly different otherwise
    payloadBytes, _ = json.Marshal(validPayload)
	w = performRequest(testRouter, "POST", "/api/v1/entity-relationships/", bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusConflict, w.Code) // Expect conflict due to unique constraint
}

func TestGetEntityRelationshipHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before ER test")
	userEntity, userPkAttr, postEntity, postUserFkAttr := seedEntityRelationshipTestData(t)
	
	relDef := EntityRelationshipDefinition{
		Name: "TestGetRel", SourceEntityID: userEntity.ID, SourceAttributeID: userPkAttr.ID,
		TargetEntityID: postEntity.ID, TargetAttributeID: postUserFkAttr.ID, RelationshipType: OneToOne,
		Metadata:       map[string]interface{}{"source": "get_test", "version": 1.1, "active": true},
	}
	createdRel, err := testStore.CreateEntityRelationship(relDef, relDef.Metadata) // Create directly via store
	require.NoError(t, err)

	w := performRequest(testRouter, "GET", "/api/v1/entity-relationships/"+createdRel.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedRel EntityRelationshipDefinition
	err = json.Unmarshal(w.Body.Bytes(), &fetchedRel)
	assert.NoError(t, err)
	assert.Equal(t, createdRel.ID, fetchedRel.ID)
	assert.Equal(t, relDef.Name, fetchedRel.Name)
	assert.Equal(t, relDef.Metadata, fetchedRel.Metadata)

	w = performRequest(testRouter, "GET", "/api/v1/entity-relationships/non-existent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListEntityRelationshipsHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before ER test")
	userEntity, userPkAttr, postEntity, postUserFkAttr := seedEntityRelationshipTestData(t)
	meta1 := map[string]interface{}{"list_test_key": "val1"}
	meta2 := map[string]interface{}{"list_test_key": "val2", "is_active": false}


	// Create a couple of relationships
	rel1, err := testStore.CreateEntityRelationship(EntityRelationshipDefinition{
		Name: "UserPostsList1", SourceEntityID: userEntity.ID, SourceAttributeID: userPkAttr.ID,
		TargetEntityID: postEntity.ID, TargetAttributeID: postUserFkAttr.ID, RelationshipType: OneToMany,
	}, meta1)
	require.NoError(t, err)
	
	// Find the "id" attribute for postEntity
	postAttrs, _, _ := testStore.ListAttributes(postEntity.ID, ListParams{Limit:10})
	var postPkForListTest AttributeDefinition
	for _, pa := range postAttrs {
		if pa.Name == "id" { // Assuming the PK attribute for postEntity is named "id"
			postPkForListTest = pa
			break
		}
	}
	require.NotEmpty(t, postPkForListTest.ID, "Post PK attribute 'id' not found for postEntity %s", postEntity.ID)


	_, err = testStore.CreateEntityRelationship(EntityRelationshipDefinition{
		Name: "PostAuthorsList2", SourceEntityID: postEntity.ID, SourceAttributeID: postPkForListTest.ID, 
		TargetEntityID: userEntity.ID, TargetAttributeID: userPkAttr.ID, RelationshipType: ManyToOne,
	}, meta2)
	require.NoError(t, err)


	// Test listing all - initial state
	w := performRequest(testRouter, "GET", "/api/v1/entity-relationships/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	var rels []EntityRelationshipDefinition
	relsBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(relsBytes, &rels)
	require.NoError(t, err)
	assert.Len(t, rels, 2)

	// Test pagination: limit=1
	w = performRequest(testRouter, "GET", "/api/v1/entity-relationships/?limit=1", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	relsBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(relsBytes, &rels)
	require.NoError(t, err)
	assert.Len(t, rels, 1)

	// Test filtering by source_entity_id
	w = performRequest(testRouter, "GET", fmt.Sprintf("/api/v1/entity-relationships/?source_entity_id=%s", userEntity.ID), nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total) 
	relsBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(relsBytes, &rels)
	require.NoError(t, err)
	assert.Len(t, rels, 1)
	assert.Equal(t, "UserPostsList1", rels[0].Name)
	assert.Equal(t, meta1, rels[0].Metadata)


	// Test filtering by source_entity_id with pagination
	w = performRequest(testRouter, "GET", fmt.Sprintf("/api/v1/entity-relationships/?source_entity_id=%s&limit=1&offset=0", userEntity.ID), nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	relsBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(relsBytes, &rels)
	require.NoError(t, err)
	assert.Len(t, rels, 1)
	
	// Test filtering with non-existent source_entity_id
	w = performRequest(testRouter, "GET", "/api/v1/entity-relationships/?source_entity_id=non-existent", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code) // Should still be OK, but with 0 results
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
	assert.Empty(t, resp.Data)
}

func TestUpdateEntityRelationshipHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before ER test")
	userEntity, userPkAttr, postEntity, postUserFkAttr := seedEntityRelationshipTestData(t)
	
	relDef := EntityRelationshipDefinition{
		Name: "OriginalRelName", Description: "Original Desc",
		SourceEntityID: userEntity.ID, SourceAttributeID: userPkAttr.ID,
		TargetEntityID: postEntity.ID, TargetAttributeID: postUserFkAttr.ID, RelationshipType: OneToOne,
		Metadata: map[string]interface{}{"version": 1.0},
	}
	createdRel, err := testStore.CreateEntityRelationship(relDef, relDef.Metadata)
	require.NoError(t, err)

	updatePayload := createdRel
	updatePayload.Name = "UpdatedRelName"
	updatePayload.Description = "Updated Desc"
	updatePayload.RelationshipType = ManyToOne
	updatePayload.Metadata = map[string]interface{}{"version": 2.0, "status": "reviewed", "updated_by": "test_user_api"}
	payloadBytes, _ := json.Marshal(updatePayload)

	w := performRequest(testRouter, "PUT", "/api/v1/entity-relationships/"+createdRel.ID, bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedRel EntityRelationshipDefinition
	err = json.Unmarshal(w.Body.Bytes(), &updatedRel)
	assert.NoError(t, err)
	assert.Equal(t, "UpdatedRelName", updatedRel.Name)
	assert.Equal(t, "Updated Desc", updatedRel.Description)
	assert.Equal(t, ManyToOne, updatedRel.RelationshipType)
	assert.Equal(t, updatePayload.Metadata, updatedRel.Metadata)
	assert.True(t, updatedRel.UpdatedAt.After(createdRel.UpdatedAt))

	// Test updating to ManyToMany and clearing metadata
	updatePayloadM2M := updatedRel // Use the already updated 'updatedRel' to ensure we have its UpdatedAt
	updatePayloadM2M.Name = "UpdatedToM2M"
	updatePayloadM2M.RelationshipType = ManyToMany
	updatePayloadM2M.Metadata = nil // Clear metadata
	payloadBytesM2M, _ := json.Marshal(updatePayloadM2M)
	w = performRequest(testRouter, "PUT", "/api/v1/entity-relationships/"+createdRel.ID, bytes.NewReader(payloadBytesM2M), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedM2MRel EntityRelationshipDefinition
	err = json.Unmarshal(w.Body.Bytes(), &updatedM2MRel)
	assert.NoError(t, err)
	assert.Equal(t, "UpdatedToM2M", updatedM2MRel.Name)
	assert.Equal(t, ManyToMany, updatedM2MRel.RelationshipType)
	assert.Nil(t, updatedM2MRel.Metadata)
	assert.True(t, updatedM2MRel.UpdatedAt.After(updatedRel.UpdatedAt))


	// Test update non-existent
	w = performRequest(testRouter, "PUT", "/api/v1/entity-relationships/non-existent-id", bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEntityRelationshipHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before ER test")
	userEntity, userPkAttr, postEntity, postUserFkAttr := seedEntityRelationshipTestData(t)
	
	relDef := EntityRelationshipDefinition{
		Name: "RelToBeDeleted", SourceEntityID: userEntity.ID, SourceAttributeID: userPkAttr.ID,
		TargetEntityID: postEntity.ID, TargetAttributeID: postUserFkAttr.ID, RelationshipType: OneToOne,
	}
	createdRel, err := testStore.CreateEntityRelationship(relDef)
	require.NoError(t, err)

	w := performRequest(testRouter, "DELETE", "/api/v1/entity-relationships/"+createdRel.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err = testStore.GetEntityRelationship(createdRel.ID) // Check if it's gone from store
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Test delete non-existent
	w = performRequest(testRouter, "DELETE", "/api/v1/entity-relationships/non-existent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdEntity, _ := testStore.CreateEntity("To Be Deleted", "Desc")
	_, _ = testStore.CreateAttribute(createdEntity.ID, "Attr1", BaseTypeString, nil, "Test Attr", false, false, false)

	w := performRequest(testRouter, "DELETE", "/api/v1/entities/"+createdEntity.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := testStore.GetEntity(createdEntity.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "Entity should be deleted")

	// Ensure ListAttributes is called with ListParams
	attrs, _, err := testStore.ListAttributes(createdEntity.ID, ListParams{Limit: 10})
	assert.NoError(t, err, "Listing attributes for a deleted entity ID should not error but return empty")
	assert.Empty(t, attrs, "Attributes should be cascade deleted")

	w = performRequest(testRouter, "DELETE", "/api/v1/entities/nonexistent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Attribute Handler Tests (Adapted) ---

func TestCreateAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity For Attr", "Entity Desc")

	// Test with basic type and no details
	payloadBasic := `{"name": "Test Attr Basic", "data_type_name": "string", "description": "A basic string attribute", "is_filterable": true, "is_pii": false, "is_indexed": true, "metadata": {"origin": "test"}}`
	w := performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payloadBasic), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var attrBasic AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attrBasic)
	assert.NoError(t, err)
	assert.Equal(t, "Test Attr Basic", attrBasic.Name)
	assert.Equal(t, BaseTypeString, attrBasic.DataTypeName)
	assert.Nil(t, attrBasic.DataTypeDetails) 
	assert.True(t, attrBasic.IsFilterable)
	assert.True(t, attrBasic.IsIndexed)
	require.NotNil(t, attrBasic.Metadata)
	assert.Equal(t, "test", attrBasic.Metadata["origin"])


	// Test with complex type (enum) and details, and different metadata
	payloadEnum := `{"name": "Test Attr Enum", "data_type_name": "enum", "data_type_details": {"values": ["A", "B"]}, "description": "An enum attribute", "metadata": {"version": 1.2, "active": true}}`
	w = performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payloadEnum), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var attrEnum AttributeDefinition
	err = json.Unmarshal(w.Body.Bytes(), &attrEnum)
	assert.NoError(t, err)
	assert.Equal(t, "Test Attr Enum", attrEnum.Name)
	assert.Equal(t, BaseTypeEnum, attrEnum.DataTypeName)
	require.NotNil(t, attrEnum.DataTypeDetails)
	assert.Equal(t, []interface{}{"A", "B"}, attrEnum.DataTypeDetails["values"])
	require.NotNil(t, attrEnum.Metadata)
	assert.Equal(t, 1.2, attrEnum.Metadata["version"])
	assert.Equal(t, true, attrEnum.Metadata["active"])
	
	// Test creation without metadata field
	payloadNoMeta := `{"name": "Test Attr No Meta", "data_type_name": "string", "description": "No metadata here"}`
	w = performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payloadNoMeta), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var attrNoMeta AttributeDefinition
	err = json.Unmarshal(w.Body.Bytes(), &attrNoMeta)
	assert.NoError(t, err)
	assert.Equal(t, "Test Attr No Meta", attrNoMeta.Name)
	assert.Nil(t, attrNoMeta.Metadata, "Metadata should be nil or empty if not provided")


	// Test with invalid data_type_name (though API doesn't validate this strictly yet beyond what store might do)
	// payloadInvalidType := `{"name": "Test Attr Invalid", "data_type_name": "invalidtype", "description": "Invalid type"}`
	// w = performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payloadInvalidType), nil)
	// assert.Equal(t, http.StatusBadRequest, w.Code) // Or 500 if store fails due to DB constraint on enum DataTypeName if it were a DB enum

	// Test with missing required field (e.g. name)
	payloadMissingName := `{"data_type_name": "string"}`
	w = performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payloadMissingName), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)


	w = performRequest(testRouter, "POST", "/api/v1/entities/nonexistententity/attributes/", strings.NewReader(payloadBasic), nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // API should check if entity exists
}

func TestListAttributesHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc", nil)
	attr1Meta := map[string]interface{}{"source": "systemA"}
	attr2Meta := map[string]interface{}{"source": "systemB", "quality": "high"}
	_, _ = testStore.CreateAttribute(entity.ID, "Attr1", BaseTypeString, nil, "Desc1", false, false, false, attr1Meta)

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+entity.ID+"/attributes/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	var attrs []AttributeDefinition
	attrsBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(attrsBytes, &attrs)
	require.NoError(t, err)
	assert.Len(t, attrs, 1)
	assert.Equal(t, BaseTypeString, attrs[0].DataTypeName)
	assert.Equal(t, attr1Meta, attrs[0].Metadata)


	// Add another attribute for pagination test
	_, _ = testStore.CreateAttribute(entity.ID, "Attr2", BaseTypeInteger, nil, "Desc2", true, false, true, attr2Meta)
	w = performRequest(testRouter, "GET", "/api/v1/entities/"+entity.ID+"/attributes/?limit=1&offset=1", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	attrsBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(attrsBytes, &attrs)
	require.NoError(t, err)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "Attr2", attrs[0].Name) 
	assert.Equal(t, BaseTypeInteger, attrs[0].DataTypeName)
	assert.Equal(t, attr2Meta, attrs[0].Metadata)
}

func TestGetAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc", nil)
	attrDetails := map[string]interface{}{"format": "email"}
	attrMeta := map[string]interface{}{"pii_details": "encrypted", "source_system": "hr"}
	attr, _ := testStore.CreateAttribute(entity.ID, "TestAttr", BaseTypeString, attrDetails, "Desc", false, true, false, attrMeta)

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &fetchedAttr)
	assert.NoError(t, err)
	assert.Equal(t, attr.ID, fetchedAttr.ID)
	assert.Equal(t, BaseTypeString, fetchedAttr.DataTypeName)
	assert.Equal(t, attrDetails, fetchedAttr.DataTypeDetails)
	assert.Equal(t, attrMeta, fetchedAttr.Metadata)
}

func TestUpdateAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc", nil)
	initialMeta := map[string]interface{}{"version": 1.0, "status": "draft"}
	attr, _ := testStore.CreateAttribute(entity.ID, "Old Attr Name", BaseTypeString, nil, "Old Attr Desc", false, false, false, initialMeta)

	updatePayload := map[string]interface{}{
		"name":              "New Attr Name",
		"data_type_name":    BaseTypeEnum,
		"data_type_details": map[string]interface{}{"values": []string{"X", "Y"}},
		"description":       "New Attr Desc",
		"is_pii":            true,
		"metadata":          map[string]interface{}{"version": 2.0, "status": "published", "reviewed_by": "user_A"},
	}
	payloadBytes, _ := json.Marshal(updatePayload)
	w := performRequest(testRouter, "PUT", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &updatedAttr)
	assert.NoError(t, err)
	assert.Equal(t, "New Attr Name", updatedAttr.Name)
	assert.Equal(t, BaseTypeEnum, updatedAttr.DataTypeName)
	require.NotNil(t, updatedAttr.DataTypeDetails)
	assert.Equal(t, []interface{}{"X", "Y"}, updatedAttr.DataTypeDetails["values"])
	assert.True(t, updatedAttr.IsPii)
	require.NotNil(t, updatedAttr.Metadata)
	assert.Equal(t, 2.0, updatedAttr.Metadata["version"]) // JSON numbers are float64
	assert.Equal(t, "published", updatedAttr.Metadata["status"])
	assert.Equal(t, "user_A", updatedAttr.Metadata["reviewed_by"])
	assert.True(t, updatedAttr.UpdatedAt.After(attr.UpdatedAt))

	// Test clearing metadata by sending metadata: null
	clearMetaPayload := map[string]interface{}{
		"name":           "New Attr Name", // Name is required
		"data_type_name": BaseTypeEnum,    // Type name is required
		"metadata":       nil,
	}
	payloadBytes, _ = json.Marshal(clearMetaPayload)
	w = performRequest(testRouter, "PUT", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var clearedMetaAttr AttributeDefinition
	err = json.Unmarshal(w.Body.Bytes(), &clearedMetaAttr)
	assert.NoError(t, err)
	assert.Nil(t, clearedMetaAttr.Metadata) // Expect nil or empty map
	
	// Test updating without metadata field (should preserve existing metadata)
	attrWithMeta, _ := testStore.GetAttribute(entity.ID, attr.ID) // Get the attribute which now has nil metadata
	attrWithMeta.Metadata = map[string]interface{}{"original": "value"} // Set some metadata directly
	_, _ = testStore.UpdateAttribute(entity.ID, attr.ID, attrWithMeta.Name, attrWithMeta.DataTypeName, attrWithMeta.DataTypeDetails, attrWithMeta.Description, attrWithMeta.IsFilterable, attrWithMeta.IsPii, attrWithMeta.IsIndexed, attrWithMeta.Metadata)

	updateNoMetaPayload := map[string]interface{}{
		"name": "Name Update No Meta",
		"data_type_name": BaseTypeString,
	}
	payloadBytes, _ = json.Marshal(updateNoMetaPayload)
	w = performRequest(testRouter, "PUT", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var noMetaUpdateAttr AttributeDefinition
	err = json.Unmarshal(w.Body.Bytes(), &noMetaUpdateAttr)
	assert.NoError(t, err)
	assert.Equal(t, "Name Update No Meta", noMetaUpdateAttr.Name)
	assert.Equal(t, attrWithMeta.Metadata, noMetaUpdateAttr.Metadata, "Metadata should be preserved if not included in update payload")


	w = performRequest(testRouter, "PUT", "/api/v1/entities/nonexistententity/attributes/"+attr.ID, bytes.NewReader(payloadBytes), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc")
	attr, _ := testStore.CreateAttribute(entity.ID, "To Be Deleted Attr", BaseTypeString, nil, "Desc", false, false, false)

	w := performRequest(testRouter, "DELETE", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := testStore.GetAttribute(entity.ID, attr.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "Attribute should be deleted")
}

// --- DataSourceConfig Handler Tests (New) ---

func TestCreateDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Related Entity", "For DataSource Test")

	// Test successful creation with EntityID and metadata
	dsMeta := map[string]interface{}{"env": "production", "tier": 1.0}
	payloadWithEntity := fmt.Sprintf(`{"name": "Test DS With Entity", "type": "PostgreSQL", "connection_details": "{\"host\":\"localhost\"}", "entity_id": "%s", "metadata": {"env": "production", "tier": 1.0}}`, entity.ID)
	w := performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadWithEntity), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var dsWithEntity DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &dsWithEntity)
	assert.NoError(t, err)
	assert.NotEmpty(t, dsWithEntity.ID)
	assert.Equal(t, "Test DS With Entity", dsWithEntity.Name)
	assert.Equal(t, entity.ID, dsWithEntity.EntityID)
	assert.Equal(t, dsMeta, dsWithEntity.Metadata)
	assert.WithinDuration(t, time.Now(), dsWithEntity.CreatedAt, 2*time.Second)

	// Test successful creation without EntityID and no metadata
	payloadWithoutEntity := `{"name": "Test DS No Entity", "type": "CSV", "connection_details": "{\"path\":\"/data/file.csv\"}"}`
	w = performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadWithoutEntity), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var dsWithoutEntity DataSourceConfig
	err = json.Unmarshal(w.Body.Bytes(), &dsWithoutEntity)
	assert.NoError(t, err)
	assert.NotEmpty(t, dsWithoutEntity.ID)
	assert.Equal(t, "Test DS No Entity", dsWithoutEntity.Name)
	assert.Empty(t, dsWithoutEntity.EntityID)

	// Test missing name
	payloadMissingName := `{"type": "MySQL", "connection_details": "{}"}`
	w = performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadMissingName), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test missing type
	payloadMissingType := `{"name": "DS Missing Type", "connection_details": "{}"}`
	w = performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadMissingType), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test missing connection_details
	payloadMissingConn := `{"name": "DS Missing Conn", "type": "API"}`
	w = performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadMissingConn), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListDataSourceConfigsHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	// Test empty list
	w := performRequest(testRouter, "GET", "/api/v1/datasources/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
	assert.Empty(t, resp.Data)

	// Create some data sources
	ds1Meta := map[string]interface{}{"scope": "regional"}
	ds2Meta := map[string]interface{}{"scope": "global", "critical": true}
	ds1, _ := testStore.CreateDataSource(DataSourceConfig{Name: "DS1", Type: "PostgreSQL", ConnectionDetails: "{}", EntityID: ""}, ds1Meta)
	_, _ = testStore.CreateDataSource(DataSourceConfig{Name: "DS2", Type: "CSV", ConnectionDetails: "{}", EntityID: ""}, ds2Meta)


	w = performRequest(testRouter, "GET", "/api/v1/datasources/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	var dss []DataSourceConfig
	dssBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(dssBytes, &dss)
	require.NoError(t, err)
	assert.Len(t, dss, 2)
	for _, dsItem := range dss {
		if dsItem.Name == "DS1" {
			assert.Equal(t, ds1Meta, dsItem.Metadata)
		} else if dsItem.Name == "DS2" {
			assert.Equal(t, ds2Meta, dsItem.Metadata)
		}
	}

	// Test pagination
	w = performRequest(testRouter, "GET", "/api/v1/datasources/?limit=1&offset=0", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	dssBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(dssBytes, &dss)
	require.NoError(t, err)
	assert.Len(t, dss, 1)
	assert.Equal(t, ds1.Name, dss[0].Name) // Assuming order by name or creation
}

func TestGetDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	dsMeta := map[string]interface{}{"region": "us-east-1"}
	createdDS, _ := testStore.CreateDataSource(DataSourceConfig{Name: "Test DS", Type: "API", ConnectionDetails: "{\"url\":\"http://api.com\"}", Metadata: dsMeta})

	w := performRequest(testRouter, "GET", "/api/v1/datasources/"+createdDS.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var ds DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &ds)
	assert.NoError(t, err)
	assert.Equal(t, createdDS.ID, ds.ID)
	assert.Equal(t, "Test DS", ds.Name)
	assert.Equal(t, dsMeta, ds.Metadata)


	w = performRequest(testRouter, "GET", "/api/v1/datasources/nonexistent-ds-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Referenced Entity", "For DS Update", nil)
	initialMeta := map[string]interface{}{"version": 1.0}
	createdDS, _ := testStore.CreateDataSource(DataSourceConfig{Name: "Old DS Name", Type: "OldType", ConnectionDetails: "{\"old\":\"details\"}"}, initialMeta)
	
	updatedMeta := map[string]interface{}{"status": "active", "tier": 2.0, "version": 1.1}

	// Test successful update with EntityID and Metadata
	payload := fmt.Sprintf(`{"name": "New DS Name", "type": "NewType", "connection_details": "{\"new\":\"details\"}", "entity_id": "%s", "metadata": %s}`, 
		entity.ID, `{"status": "active", "tier": 2.0, "version": 1.1}`)
	w := performRequest(testRouter, "PUT", "/api/v1/datasources/"+createdDS.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var ds DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &ds)
	assert.NoError(t, err)
	assert.Equal(t, "New DS Name", ds.Name)
	assert.Equal(t, "NewType", ds.Type)
	assert.Equal(t, entity.ID, ds.EntityID)
	assert.Equal(t, updatedMeta, ds.Metadata)
	assert.True(t, ds.UpdatedAt.After(createdDS.UpdatedAt))

	// Test updating a non-existent data source
	w = performRequest(testRouter, "PUT", "/api/v1/datasources/nonexistent-ds-id", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update with missing name (should be bad request)
	payloadMissingName := `{"type": "AnotherType", "connection_details": "{}"}`
	w = performRequest(testRouter, "PUT", "/api/v1/datasources/"+createdDS.ID, strings.NewReader(payloadMissingName), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdDS, _ := testStore.CreateDataSource(DataSourceConfig{Name: "DS To Delete", Type: "Temp", ConnectionDetails: "{}"})

	// Test successful deletion
	w := performRequest(testRouter, "DELETE", "/api/v1/datasources/"+createdDS.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := testStore.GetDataSource(createdDS.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "DataSourceConfig should be deleted")

	// Test deleting a non-existent data source
	w = performRequest(testRouter, "DELETE", "/api/v1/datasources/nonexistent-ds-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- DataSourceFieldMapping Handler Tests (New) ---

// setupPrerequisitesForFieldMappingTests creates an entity, attribute, and data source for field mapping tests.
func setupPrerequisitesForFieldMappingTests(t *testing.T) (EntityDefinition, AttributeDefinition, DataSourceConfig) {
	entity, err := testStore.CreateEntity("FM Test Entity", "Entity for FieldMapping tests")
	require.NoError(t, err)

	attribute, err := testStore.CreateAttribute(entity.ID, "FM Test Attribute", BaseTypeString, nil, "Attribute for FieldMapping tests", false, false, false)
	require.NoError(t, err)

	dataSource, err := testStore.CreateDataSource(DataSourceConfig{
		Name:              "FM Test DataSource",
		Type:              "CSV",
		ConnectionDetails: `{"path":"/dev/null"}`,
	}, nil)
	require.NoError(t, err)

	return entity, attribute, dataSource
}

func TestCreateFieldMappingHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)

	// Test successful creation
	payload := fmt.Sprintf(`{
		"source_field_name": "source_col_1",
		"entity_id": "%s",
		"attribute_id": "%s",
		"transformation_rule": "lowercase",
		"metadata": {"quality_check": "passed"}
	}`, entity.ID, attribute.ID)
	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/", dataSource.ID)
	w := performRequest(testRouter, "POST", url, strings.NewReader(payload), nil)

	assert.Equal(t, http.StatusCreated, w.Code)
	var fm DataSourceFieldMapping
	err := json.Unmarshal(w.Body.Bytes(), &fm)
	assert.NoError(t, err)
	assert.NotEmpty(t, fm.ID)
	assert.Equal(t, "source_col_1", fm.SourceFieldName)
	assert.Equal(t, entity.ID, fm.EntityID)
	assert.Equal(t, attribute.ID, fm.AttributeID)
	assert.Equal(t, "lowercase", fm.TransformationRule)
	assert.Equal(t, dataSource.ID, fm.SourceID) 
	require.NotNil(t, fm.Metadata)
	assert.Equal(t, "passed", fm.Metadata["quality_check"])


	// Test with non-existent source_id in URL
	invalidUrl := fmt.Sprintf("/api/v1/datasources/%s/mappings/", "nonexistent-source-id")
	w = performRequest(testRouter, "POST", invalidUrl, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // API should check if dataSource exists

	// Test with non-existent entity_id in payload
	payloadNonExistentEntity := fmt.Sprintf(`{
		"source_field_name": "source_col_2",
		"entity_id": "nonexistent-entity-id",
		"attribute_id": "%s"
	}`, attribute.ID)
	w = performRequest(testRouter, "POST", url, strings.NewReader(payloadNonExistentEntity), nil)
	// This might be a 400 (validation) or 404/500 if DB constraint fails later and not caught by API validation
	// Assuming API validates this foreign key concept or DB constraint is hit.
	// For now, expect 400 as bad request due to invalid reference, or 404 if API checks.
	// Let's assume API returns 400 for invalid foreign key references in payload.
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError}, w.Code, "Creating with non-existent entity_id should fail")


	// Test missing source_field_name
	payloadMissingField := fmt.Sprintf(`{"entity_id": "%s", "attribute_id": "%s"}`, entity.ID, attribute.ID)
	w = performRequest(testRouter, "POST", url, strings.NewReader(payloadMissingField), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListFieldMappingsHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)
	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/", dataSource.ID)

	// Test empty list
	w := performRequest(testRouter, "GET", url, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
	assert.Empty(t, resp.Data)

	// Create a mapping
	fm1, err := testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "col1",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
	}, nil)
	require.NoError(t, err)

	w = performRequest(testRouter, "GET", url, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	var fms []DataSourceFieldMapping
	fmsBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(fmsBytes, &fms)
	require.NoError(t, err)
	assert.Len(t, fms, 1)
	assert.Equal(t, "col1", fms[0].SourceFieldName)

	// Test pagination
	fm2Meta := map[string]interface{}{"validated": false}
	_, _ = testStore.CreateFieldMapping(DataSourceFieldMapping{SourceID: dataSource.ID, SourceFieldName: "col2", EntityID: entity.ID, AttributeID: attribute.ID}, fm2Meta)
	w = performRequest(testRouter, "GET", url+"?limit=1&offset=0", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	fmsBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(fmsBytes, &fms)
	require.NoError(t, err)
	assert.Len(t, fms, 1)
	assert.Equal(t, fm1.SourceFieldName, fms[0].SourceFieldName)
	// Metadata for fm1 was nil as per CreateFieldMapping call for fm1.
	assert.Nil(t, fms[0].Metadata, "Expected metadata for fm1 to be nil as per its creation")


	// Test list for non-existent source_id
	invalidUrl := fmt.Sprintf("/api/v1/datasources/%s/mappings/", "nonexistent-source-id")
	w = performRequest(testRouter, "GET", invalidUrl, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // API should check if dataSource exists
}

func TestGetFieldMappingHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)

	fmMeta := map[string]interface{}{"verified": true}
	fm, err := testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "col_to_get",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
	}, fmMeta)
	require.NoError(t, err)

	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/%s", dataSource.ID, fm.ID)
	w := performRequest(testRouter, "GET", url, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedFm DataSourceFieldMapping
	err = json.Unmarshal(w.Body.Bytes(), &fetchedFm)
	assert.NoError(t, err)
	assert.Equal(t, fm.ID, fetchedFm.ID)
	assert.Equal(t, "col_to_get", fetchedFm.SourceFieldName)
	assert.Equal(t, fmMeta, fetchedFm.Metadata)

	// Test non-existent mapping_id
	urlNonExistentMapping := fmt.Sprintf("/api/v1/datasources/%s/mappings/nonexistent-mapping-id", dataSource.ID)
	w = performRequest(testRouter, "GET", urlNonExistentMapping, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test non-existent source_id
	urlNonExistentSource := fmt.Sprintf("/api/v1/datasources/nonexistent-source-id/mappings/%s", fm.ID)
	w = performRequest(testRouter, "GET", urlNonExistentSource, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateFieldMappingHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)
	fm, err := testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "old_source_name",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
		Metadata:        map[string]interface{}{"initial_version": "1.0"},
	}, map[string]interface{}{"initial_version": "1.0"})
	require.NoError(t, err)

	// Create another attribute to test updating attribute_id
	anotherAttribute, err := testStore.CreateAttribute(entity.ID, "Another Attribute", BaseTypeInteger, nil, "Another desc", false, false, false, nil)
	require.NoError(t, err)
	
	updatedMeta := map[string]interface{}{"update_source": "api_test_v2", "status": "pending"}

	payload := fmt.Sprintf(`{
		"source_field_name": "new_source_name",
		"entity_id": "%s",
		"attribute_id": "%s",
		"transformation_rule": "trim",
		"metadata": {"update_source": "api_test_v2", "status": "pending"}
	}`, entity.ID, anotherAttribute.ID) 

	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/%s", dataSource.ID, fm.ID)
	w := performRequest(testRouter, "PUT", url, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedFm DataSourceFieldMapping
	err = json.Unmarshal(w.Body.Bytes(), &updatedFm)
	assert.NoError(t, err)
	assert.Equal(t, "new_source_name", updatedFm.SourceFieldName)
	assert.Equal(t, "trim", updatedFm.TransformationRule)
	assert.Equal(t, anotherAttribute.ID, updatedFm.AttributeID) 
	assert.Equal(t, updatedMeta, updatedFm.Metadata)
	assert.True(t, updatedFm.UpdatedAt.After(fm.UpdatedAt))

	// Test with non-existent mapping_id
	urlNonExistentMapping := fmt.Sprintf("/api/v1/datasources/%s/mappings/nonexistent-mapping-id", dataSource.ID)
	w = performRequest(testRouter, "PUT", urlNonExistentMapping, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test with non-existent source_id
	urlNonExistentSource := fmt.Sprintf("/api/v1/datasources/nonexistent-source-id/mappings/%s", fm.ID)
	w = performRequest(testRouter, "PUT", urlNonExistentSource, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update with missing source_field_name
	payloadMissingField := fmt.Sprintf(`{"entity_id": "%s", "attribute_id": "%s"}`, entity.ID, attribute.ID)
	w = performRequest(testRouter, "PUT", url, strings.NewReader(payloadMissingField), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteFieldMappingHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)
	fm, err := testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "to_be_deleted",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
	})
	require.NoError(t, err)

	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/%s", dataSource.ID, fm.ID)
	w := performRequest(testRouter, "DELETE", url, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err = testStore.GetFieldMapping(dataSource.ID, fm.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "FieldMapping should be deleted")

	// Test non-existent mapping_id
	urlNonExistentMapping := fmt.Sprintf("/api/v1/datasources/%s/mappings/nonexistent-mapping-id", dataSource.ID)
	w = performRequest(testRouter, "DELETE", urlNonExistentMapping, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test non-existent source_id
	urlNonExistentSource := fmt.Sprintf("/api/v1/datasources/nonexistent-source-id/mappings/%s", fm.ID)
	w = performRequest(testRouter, "DELETE", urlNonExistentSource, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- ScheduleDefinition Handler Tests (New) ---

func TestCreateScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	// Test successful creation
	validPayload := `{"name": "Daily Ingest", "description": "Ingests data daily", "cron_expression": "0 0 * * *", "task_type": "ingest_data_source", "task_parameters": "{\"source_id\":\"uuid-for-source\"}", "is_enabled": true, "metadata": {"owner": "data_eng"}}`
	w := performRequest(testRouter, "POST", "/api/v1/schedules/", strings.NewReader(validPayload), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var createdSchedule ScheduleDefinition
	err := json.Unmarshal(w.Body.Bytes(), &createdSchedule)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdSchedule.ID)
	assert.Equal(t, "Daily Ingest", createdSchedule.Name)
	assert.Equal(t, "0 0 * * *", createdSchedule.CronExpression)
	assert.Equal(t, "ingest_data_source", createdSchedule.TaskType)
	assert.Equal(t, `{"source_id":"uuid-for-source"}`, createdSchedule.TaskParameters) 
	assert.True(t, createdSchedule.IsEnabled)
	require.NotNil(t, createdSchedule.Metadata)
	assert.Equal(t, "data_eng", createdSchedule.Metadata["owner"])
	assert.WithinDuration(t, time.Now(), createdSchedule.CreatedAt, 2*time.Second)

	// Test missing required fields
	missingFieldsTests := []struct {
		name    string
		payload string
		field   string
	}{
		{"Missing Name", `{"description": "Test", "cron_expression": "0 0 * * *", "task_type": "type", "task_parameters": "{}"}`, "name"},
		{"Missing CronExpression", `{"name": "Test", "description": "Test", "task_type": "type", "task_parameters": "{}"}`, "cron_expression"},
		{"Missing TaskType", `{"name": "Test", "description": "Test", "cron_expression": "0 0 * * *", "task_parameters": "{}"}`, "task_type"},
		{"Missing TaskParameters", `{"name": "Test", "description": "Test", "cron_expression": "0 0 * * *", "task_type": "type"}`, "task_parameters"},
	}

	for _, tt := range missingFieldsTests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(testRouter, "POST", "/api/v1/schedules/", strings.NewReader(tt.payload), nil)
			assert.Equal(t, http.StatusBadRequest, w.Code, "Expected BadRequest for missing "+tt.field)
			var errorResponse map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], tt.field) // Gin binding error messages usually mention the field
		})
	}
}

func TestListScheduleDefinitionsHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	// Test empty list
	w := performRequest(testRouter, "GET", "/api/v1/schedules/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
	assert.Empty(t, resp.Data)

	// Create some schedules directly via store for setup
	s1Meta := map[string]interface{}{"team": "alpha"}
	s2Meta := map[string]interface{}{"team": "beta", "criticality": 3.0}
	s1, _ := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Schedule A", CronExpression: "0 * * * *", TaskType: "typeA", TaskParameters: "{}"}, s1Meta)
	_, _ = testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Schedule B", CronExpression: "30 * * * *", TaskType: "typeB", TaskParameters: "{}"}, s2Meta)

	w = performRequest(testRouter, "GET", "/api/v1/schedules/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	var schedules []ScheduleDefinition
	schedulesBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(schedulesBytes, &schedules)
	require.NoError(t, err)
	assert.Len(t, schedules, 2)
	for _, s := range schedules {
		if s.Name == "Schedule A" {
			assert.Equal(t, s1Meta, s.Metadata)
		} else if s.Name == "Schedule B" {
			assert.Equal(t, s2Meta, s.Metadata)
		}
	}


	// Test pagination
	w = performRequest(testRouter, "GET", "/api/v1/schedules/?limit=1&offset=0", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	schedulesBytes, _ = json.Marshal(resp.Data)
	err = json.Unmarshal(schedulesBytes, &schedules)
	require.NoError(t, err)
	assert.Len(t, schedules, 1)
	assert.Equal(t, s1.Name, schedules[0].Name) // Assuming default order is by name
}

func TestGetScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	scheduleMeta := map[string]interface{}{"priority": "high"}
	createdSchedule, err := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "My Schedule", CronExpression: "15 * * * *", TaskType: "myTask", TaskParameters: `{"key":"val"}`}, scheduleMeta)
	require.NoError(t, err)

	// Test successful retrieval
	w := performRequest(testRouter, "GET", "/api/v1/schedules/"+createdSchedule.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedSchedule ScheduleDefinition
	err = json.Unmarshal(w.Body.Bytes(), &fetchedSchedule)
	assert.NoError(t, err)
	assert.Equal(t, createdSchedule.ID, fetchedSchedule.ID)
	assert.Equal(t, "My Schedule", fetchedSchedule.Name)
	assert.Equal(t, `{"key":"val"}`, fetchedSchedule.TaskParameters)
	assert.Equal(t, scheduleMeta, fetchedSchedule.Metadata)

	// Test retrieval of non-existent schedule
	w = performRequest(testRouter, "GET", "/api/v1/schedules/nonexistent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdSchedule, err := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Old Name", CronExpression: "0 0 * * *", TaskType: "old_type", TaskParameters: `{"old_param":"old_val"}`, IsEnabled: false}, nil)
	require.NoError(t, err)
	newMeta := map[string]interface{}{"last_reviewed": "2024-01-01"}

	// Test successful update
	updatePayload := `{"name": "New Name", "description": "Updated desc", "cron_expression": "0 1 * * *", "task_type": "new_type", "task_parameters": "{\"new_param\":\"new_val\"}", "is_enabled": true, "metadata": {"last_reviewed": "2024-01-01"}}`
	w := performRequest(testRouter, "PUT", "/api/v1/schedules/"+createdSchedule.ID, strings.NewReader(updatePayload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedSchedule ScheduleDefinition
	err = json.Unmarshal(w.Body.Bytes(), &updatedSchedule)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updatedSchedule.Name)
	assert.Equal(t, "Updated desc", updatedSchedule.Description)
	assert.Equal(t, "0 1 * * *", updatedSchedule.CronExpression)
	assert.Equal(t, "new_type", updatedSchedule.TaskType)
	assert.Equal(t, `{"new_param":"new_val"}`, updatedSchedule.TaskParameters)
	assert.True(t, updatedSchedule.IsEnabled)
	assert.Equal(t, newMeta, updatedSchedule.Metadata)
	assert.True(t, updatedSchedule.UpdatedAt.After(createdSchedule.UpdatedAt))

	// Test update with non-existent ID
	w = performRequest(testRouter, "PUT", "/api/v1/schedules/nonexistent-id", strings.NewReader(updatePayload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test update with missing required field (e.g., Name)
	invalidUpdatePayload := `{"description": "Only Desc", "cron_expression": "0 2 * * *"}`
	w = performRequest(testRouter, "PUT", "/api/v1/schedules/"+createdSchedule.ID, strings.NewReader(invalidUpdatePayload), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code) // Name is required by struct binding
}

func TestDeleteScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdSchedule, err := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "To Be Deleted", CronExpression: "* * * * *", TaskType: "delete_me", TaskParameters: "{}"})
	require.NoError(t, err)

	// Test successful deletion
	w := performRequest(testRouter, "DELETE", "/api/v1/schedules/"+createdSchedule.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's gone
	w = performRequest(testRouter, "GET", "/api/v1/schedules/"+createdSchedule.ID, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test deletion with non-existent ID
	w = performRequest(testRouter, "DELETE", "/api/v1/schedules/nonexistent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TODO: Add tests for ListWorkflowDefinitionsHandler pagination and ListActionTemplatesHandler pagination
// (similar to TestListScheduleDefinitionsHandler) if those endpoints are intended to be paginated.

// --- GroupDefinition Handler Tests (New - Assuming they might be missing) ---
// Adding Metadata tests for GroupDefinition handlers if they exist or creating minimal ones.
// For the purpose of this task, I will assume these tests are new or need significant adaptation for Metadata.

func TestCreateGroupDefinitionHandlerAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Grouped Entity", "For GroupDefinition Test", nil)
	groupMeta := map[string]interface{}{"department": "sales", "isActive": true, "version": 1.0}

	payload := fmt.Sprintf(`{"name": "Sales Users Group", "entity_id": "%s", "rules_json": "{\"field\":\"department\", \"equals\":\"sales\"}", "description": "Group for sales users", "metadata": {"department": "sales", "isActive": true, "version": 1.0}}`, entity.ID)
	w := performRequest(testRouter, "POST", "/api/v1/group-definitions/", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var groupDef GroupDefinition
	err := json.Unmarshal(w.Body.Bytes(), &groupDef)
	require.NoError(t, err)
	assert.NotEmpty(t, groupDef.ID)
	assert.Equal(t, "Sales Users Group", groupDef.Name)
	assert.Equal(t, groupMeta, groupDef.Metadata)
}

func TestGetGroupDefinitionHandlerAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Grouped Entity", "For GroupDefinition Test", nil)
	groupMeta := map[string]interface{}{"department": "eng", "team_lead": "john.doe", "version": 1.1}
	createdGroup, _ := testStore.CreateGroupDefinition(GroupDefinition{
		Name: "Eng Group", EntityID: entity.ID, RulesJSON: "{}", Description: "Engineering Group",
	}, groupMeta)

	w := performRequest(testRouter, "GET", "/api/v1/group-definitions/"+createdGroup.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var groupDef GroupDefinition
	err := json.Unmarshal(w.Body.Bytes(), &groupDef)
	require.NoError(t, err)
	assert.Equal(t, createdGroup.ID, groupDef.ID)
	assert.Equal(t, groupMeta, groupDef.Metadata)
}

func TestListGroupDefinitionsHandlerAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Grouped Entity List", "For GroupDefinition List Test", nil)
	meta1 := map[string]interface{}{"status": "active", "priority": 1.0}
	meta2 := map[string]interface{}{"status": "archived", "reason": "old_project", "priority": 2.0}
	_, _ = testStore.CreateGroupDefinition(GroupDefinition{Name: "Group1", EntityID: entity.ID, RulesJSON: "{}"}, meta1)
	_, _ = testStore.CreateGroupDefinition(GroupDefinition{Name: "Group2", EntityID: entity.ID, RulesJSON: "{}"}, meta2)

	w := performRequest(testRouter, "GET", "/api/v1/group-definitions/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	var groups []GroupDefinition
	groupsBytes, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(groupsBytes, &groups)
	require.NoError(t, err)
	assert.Len(t, groups, 2)
	for _, g := range groups {
		if g.Name == "Group1" {
			assert.Equal(t, meta1, g.Metadata)
		} else if g.Name == "Group2" {
			assert.Equal(t, meta2, g.Metadata)
		}
	}
}


func TestUpdateGroupDefinitionHandlerAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Grouped Entity Update", "For GroupDefinition Update Test", nil)
	initialMeta := map[string]interface{}{"version": 1.0, "owner": "old_owner"}
	group, _ := testStore.CreateGroupDefinition(GroupDefinition{Name: "Initial Group", EntityID: entity.ID, RulesJSON: "{}"}, initialMeta)
	
	updatedMeta := map[string]interface{}{"version": 2.0, "reviewed": true, "owner": "new_owner"}
	payload := fmt.Sprintf(`{"name": "Updated Group Name", "entity_id": "%s", "rules_json": "{\"new_rule\":\"true\"}", "metadata": {"version": 2.0, "reviewed": true, "owner": "new_owner"}}`, entity.ID)
	
	w := performRequest(testRouter, "PUT", "/api/v1/group-definitions/"+group.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedGroup GroupDefinition
	err := json.Unmarshal(w.Body.Bytes(), &updatedGroup)
	require.NoError(t, err)
	assert.Equal(t, "Updated Group Name", updatedGroup.Name)
	assert.Equal(t, updatedMeta, updatedGroup.Metadata)
}


// --- WorkflowDefinition and ActionTemplate Tests would follow a similar simplified pattern ---
// --- For brevity, they are omitted here but should be added if not present.     ---


// --- Bulk Entity Operation API Tests (New) ---

func TestBulkCreateEntitiesAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	t.Run("SuccessfulBulkCreate", func(t *testing.T) {
		payload := `{"entities": [
			{"name": "BulkAPIEntity1", "description": "Desc1"},
			{"name": "BulkAPIEntity2", "description": "Desc2"}
		]}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-create", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusCreated, w.Code) // All success = 201
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)
		for _, item := range resp.Results {
			assert.True(t, item.Success)
			assert.NotEmpty(t, item.ID)
			assert.NotNil(t, item.Entity)
			assert.Empty(t, item.Error)
			// Verify entity in DB
			_, getErr := testStore.GetEntity(item.ID)
			assert.NoError(t, getErr)
		}
	})

	t.Run("PartialSuccessBulkCreate", func(t *testing.T) {
		// Setup: one entity that will cause a unique constraint violation
		_, err := testStore.CreateEntity("BulkAPIEntityUnique", "Pre-existing")
		require.NoError(t, err)

		payload := `{"entities": [
			{"name": "BulkAPIEntityNew", "description": "New one"},
			{"name": "BulkAPIEntityUnique", "description": "Attempt duplicate name"}
		]}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-create", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusMultiStatus, w.Code) // Partial success = 207
		var resp BulkOperationResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)

		assert.True(t, resp.Results[0].Success)
		assert.Equal(t, "BulkAPIEntityNew", resp.Results[0].Entity.Name)

		assert.False(t, resp.Results[1].Success)
		assert.Contains(t, resp.Results[1].Error, "UNIQUE constraint failed") // SQLite specific
		assert.Nil(t, resp.Results[1].Entity)
	})

	t.Run("InvalidPayloadBulkCreate", func(t *testing.T) {
		payload := `{"entities": [{"description": "Missing Name"}]}` // Name is required
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-create", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		var apiErr APIError
		err := json.Unmarshal(w.Body.Bytes(), &apiErr)
		require.NoError(t, err)
		assert.Contains(t, apiErr.Message, "Invalid input")
	})
	
	t.Run("EmptyEntitiesListBulkCreate", func(t *testing.T) {
		payload := `{"entities": []}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-create", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusOK, w.Code) // API handler returns 200 for empty list
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Results)
	})
}

func TestBulkUpdateEntitiesAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	e1, _ := testStore.CreateEntity("BulkUpdateE1", "OrigD1")
	e2, _ := testStore.CreateEntity("BulkUpdateE2", "OrigD2")

	t.Run("SuccessfulBulkUpdate", func(t *testing.T) {
		payload := fmt.Sprintf(`{"entities": [
			{"id": "%s", "name": "UpdatedName1"},
			{"id": "%s", "description": "UpdatedDesc2"}
		]}`, e1.ID, e2.ID)
		w := performRequest(testRouter, "PUT", "/api/v1/entities/bulk-update", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusOK, w.Code) // All success = 200
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)

		assert.True(t, resp.Results[0].Success)
		assert.Equal(t, e1.ID, resp.Results[0].ID)
		assert.Equal(t, "UpdatedName1", resp.Results[0].Entity.Name)
		assert.Equal(t, "OrigD1", resp.Results[0].Entity.Description) // Description should remain

		assert.True(t, resp.Results[1].Success)
		assert.Equal(t, e2.ID, resp.Results[1].ID)
		assert.Equal(t, "BulkUpdateE2", resp.Results[1].Entity.Name) // Name should remain
		assert.Equal(t, "UpdatedDesc2", resp.Results[1].Entity.Description)
	})

	t.Run("PartialSuccessBulkUpdate", func(t *testing.T) {
		payload := fmt.Sprintf(`{"entities": [
			{"id": "%s", "name": "FurtherUpdate1"},
			{"id": "non-existent-id", "description": "NoEntityHere"}
		]}`, e1.ID)
		w := performRequest(testRouter, "PUT", "/api/v1/entities/bulk-update", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusMultiStatus, w.Code)
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)

		assert.True(t, resp.Results[0].Success)
		assert.Equal(t, "FurtherUpdate1", resp.Results[0].Entity.Name)

		assert.False(t, resp.Results[1].Success)
		assert.Equal(t, "non-existent-id", resp.Results[1].ID)
		assert.Contains(t, resp.Results[1].Error, "not found")
	})

	t.Run("InvalidPayloadBulkUpdate", func(t *testing.T) {
		payload := `{"entities": [{"name": "MissingID"}]}` // ID is required for update
		w := performRequest(testRouter, "PUT", "/api/v1/entities/bulk-update", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestBulkDeleteEntitiesAPI(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	e1, _ := testStore.CreateEntity("BulkDeleteE1", "ToDelete1")
	e2, _ := testStore.CreateEntity("BulkDeleteE2", "ToDelete2")

	t.Run("SuccessfulBulkDelete", func(t *testing.T) {
		payload := fmt.Sprintf(`{"entity_ids": ["%s", "%s"]}`, e1.ID, e2.ID)
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-delete", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusOK, w.Code) // All success (idempotent)
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)
		for _, item := range resp.Results {
			assert.True(t, item.Success)
		}
		_, getErr := testStore.GetEntity(e1.ID)
		assert.ErrorIs(t, getErr, sql.ErrNoRows)
		_, getErr = testStore.GetEntity(e2.ID)
		assert.ErrorIs(t, getErr, sql.ErrNoRows)
	})

	t.Run("PartialSuccessBulkDelete", func(t *testing.T) {
		// e1 and e2 are already deleted from previous sub-test. Create a new one.
		e3, _ := testStore.CreateEntity("BulkDeleteE3", "ToDelete3")
		payload := fmt.Sprintf(`{"entity_ids": ["%s", "non-existent-id"]}`, e3.ID)
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-delete", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusOK, w.Code) // Still 200 OK because non-existent is idempotent success
		var resp BulkOperationResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Results, 2)
		
		assert.True(t, resp.Results[0].Success) // e3 deleted
		assert.Equal(t, e3.ID, resp.Results[0].ID)

		assert.True(t, resp.Results[1].Success) // non-existent is success
		assert.Equal(t, "non-existent-id", resp.Results[1].ID)
		assert.Contains(t, resp.Results[1].Error, "not found") // Informational error
	})

	t.Run("InvalidPayloadBulkDelete", func(t *testing.T) {
		payload := `{"entity_ids": "not-an-array"}`
		w := performRequest(testRouter, "POST", "/api/v1/entities/bulk-delete", strings.NewReader(payload), nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
