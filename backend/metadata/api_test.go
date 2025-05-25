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
		// With CASCADE, the order is less strict, but it's good practice to be mindful.
		"action_templates",
		"workflow_definitions",
		"group_definitions",
		"data_source_field_mappings",
		"schedule_definitions", // Added schedule_definitions
		"action_templates",
		"workflow_definitions",
		"group_definitions",
		"data_source_field_mappings",
		"data_source_configs",
		"attribute_definitions",
		"entity_definitions",
	}

	for _, table := range tables {
		// Using TRUNCATE ... RESTART IDENTITY CASCADE to ensure clean slate and reset sequences.
		// CASCADE will also handle dependent tables if any were missed or order was wrong.
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", table)
		_, err := store.DB.Exec(query)
		if err != nil {
			// Log error but attempt to continue cleaning other tables.
			log.Printf("Failed to truncate table %s: %v. This might be okay if the table doesn't exist yet on first run or due to CASCADE.", table, err)
			// If a table doesn't exist (e.g. schema not fully up on first error), pq will error.
			// We can check for specific pq error codes if we want to be more granular.
		}
	}
	return nil // Return nil as we log errors but try to continue. A single failure might not be fatal for all tests.
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

	payload := `{"name": "Test Entity", "description": "A test entity"}`
	w := performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payload), nil)

	assert.Equal(t, http.StatusCreated, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.NotEmpty(t, entity.ID)
	assert.Equal(t, "Test Entity", entity.Name)
	assert.WithinDuration(t, time.Now(), entity.CreatedAt, 2*time.Second) // Check timestamp

	// Test missing name
	payload = `{"description": "Another test entity"}`
	w = performRequest(testRouter, "POST", "/api/v1/entities/", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListEntitiesHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	_, _ = testStore.CreateEntity("Entity 1", "Desc 1")
	_, _ = testStore.CreateEntity("Entity 2", "Desc 2")

	w := performRequest(testRouter, "GET", "/api/v1/entities/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var entities []EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entities)
	assert.NoError(t, err)
	assert.Len(t, entities, 2)
}

func TestGetEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdEntity, _ := testStore.CreateEntity("Test Entity", "Desc")

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+createdEntity.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, createdEntity.ID, entity.ID)

	w = performRequest(testRouter, "GET", "/api/v1/entities/nonexistent-uuid-format", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // Changed from StatusBadRequest due to how PostgresStore handles errors
}

func TestUpdateEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdEntity, _ := testStore.CreateEntity("Old Name", "Old Desc")

	payload := `{"name": "New Name", "description": "New Desc"}`
	w := performRequest(testRouter, "PUT", "/api/v1/entities/"+createdEntity.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var entity EntityDefinition
	err := json.Unmarshal(w.Body.Bytes(), &entity)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", entity.Name)
	assert.True(t, entity.UpdatedAt.After(createdEntity.UpdatedAt))

	w = performRequest(testRouter, "PUT", "/api/v1/entities/nonexistent-id", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEntityHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdEntity, _ := testStore.CreateEntity("To Be Deleted", "Desc")
	_, _ = testStore.CreateAttribute(createdEntity.ID, "Attr1", "string", "Test Attr", false, false, false)

	w := performRequest(testRouter, "DELETE", "/api/v1/entities/"+createdEntity.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := testStore.GetEntity(createdEntity.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "Entity should be deleted")

	attrs, err := testStore.ListAttributes(createdEntity.ID)
	assert.NoError(t, err, "Listing attributes for a deleted entity ID should not error but return empty")
	assert.Empty(t, attrs, "Attributes should be cascade deleted")

	w = performRequest(testRouter, "DELETE", "/api/v1/entities/nonexistent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Attribute Handler Tests (Adapted) ---

func TestCreateAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity For Attr", "Entity Desc")

	payload := `{"name": "Test Attr", "data_type": "string", "description": "A test attribute", "is_filterable": true, "is_pii": false, "is_indexed": true}`
	w := performRequest(testRouter, "POST", "/api/v1/entities/"+entity.ID+"/attributes/", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var attr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attr)
	assert.NoError(t, err)
	assert.Equal(t, "Test Attr", attr.Name)
	assert.True(t, attr.IsFilterable)
	assert.True(t, attr.IsIndexed)

	w = performRequest(testRouter, "POST", "/api/v1/entities/nonexistententity/attributes/", strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // API should check if entity exists
}

func TestListAttributesHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc")
	_, _ = testStore.CreateAttribute(entity.ID, "Attr1", "string", "Desc1", false, false, false)

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+entity.ID+"/attributes/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var attrs []AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &attrs)
	assert.NoError(t, err)
	assert.Len(t, attrs, 1)
}

func TestGetAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc")
	attr, _ := testStore.CreateAttribute(entity.ID, "TestAttr", "string", "Desc", false, false, false)

	w := performRequest(testRouter, "GET", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &fetchedAttr)
	assert.NoError(t, err)
	assert.Equal(t, attr.ID, fetchedAttr.ID)
}

func TestUpdateAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc")
	attr, _ := testStore.CreateAttribute(entity.ID, "Old Attr Name", "string", "Old Attr Desc", false, false, false)

	payload := `{"name": "New Attr Name", "data_type": "integer", "description": "New Attr Desc", "is_pii": true}`
	w := performRequest(testRouter, "PUT", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedAttr AttributeDefinition
	err := json.Unmarshal(w.Body.Bytes(), &updatedAttr)
	assert.NoError(t, err)
	assert.Equal(t, "New Attr Name", updatedAttr.Name)
	assert.Equal(t, "integer", updatedAttr.DataType)
	assert.True(t, updatedAttr.IsPii)
	assert.True(t, updatedAttr.UpdatedAt.After(attr.UpdatedAt))

	w = performRequest(testRouter, "PUT", "/api/v1/entities/nonexistententity/attributes/"+attr.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAttributeHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Test Entity", "Desc")
	attr, _ := testStore.CreateAttribute(entity.ID, "To Be Deleted Attr", "string", "Desc", false, false, false)

	w := performRequest(testRouter, "DELETE", "/api/v1/entities/"+entity.ID+"/attributes/"+attr.ID, nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)

	_, err := testStore.GetAttribute(entity.ID, attr.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "Attribute should be deleted")
}

// --- DataSourceConfig Handler Tests (New) ---

func TestCreateDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Related Entity", "For DataSource Test")

	// Test successful creation with EntityID
	payloadWithEntity := `{"name": "Test DS With Entity", "type": "PostgreSQL", "connection_details": "{\"host\":\"localhost\"}", "entity_id": "` + entity.ID + `"}`
	w := performRequest(testRouter, "POST", "/api/v1/datasources/", strings.NewReader(payloadWithEntity), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var dsWithEntity DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &dsWithEntity)
	assert.NoError(t, err)
	assert.NotEmpty(t, dsWithEntity.ID)
	assert.Equal(t, "Test DS With Entity", dsWithEntity.Name)
	assert.Equal(t, entity.ID, dsWithEntity.EntityID)
	assert.WithinDuration(t, time.Now(), dsWithEntity.CreatedAt, 2*time.Second)

	// Test successful creation without EntityID
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
	var emptyDs []DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &emptyDs)
	assert.NoError(t, err)
	assert.Len(t, emptyDs, 0)

	// Create some data sources
	_, _ = testStore.CreateDataSource(DataSourceConfig{Name: "DS1", Type: "PostgreSQL", ConnectionDetails: "{}", EntityID: ""})
	_, _ = testStore.CreateDataSource(DataSourceConfig{Name: "DS2", Type: "CSV", ConnectionDetails: "{}", EntityID: ""})

	w = performRequest(testRouter, "GET", "/api/v1/datasources/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var dss []DataSourceConfig
	err = json.Unmarshal(w.Body.Bytes(), &dss)
	assert.NoError(t, err)
	assert.Len(t, dss, 2)
}

func TestGetDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdDS, _ := testStore.CreateDataSource(DataSourceConfig{Name: "Test DS", Type: "API", ConnectionDetails: "{\"url\":\"http://api.com\"}"})

	w := performRequest(testRouter, "GET", "/api/v1/datasources/"+createdDS.ID, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var ds DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &ds)
	assert.NoError(t, err)
	assert.Equal(t, createdDS.ID, ds.ID)
	assert.Equal(t, "Test DS", ds.Name)

	w = performRequest(testRouter, "GET", "/api/v1/datasources/nonexistent-ds-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateDataSourceConfigHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, _ := testStore.CreateEntity("Referenced Entity", "For DS Update")
	createdDS, _ := testStore.CreateDataSource(DataSourceConfig{Name: "Old DS Name", Type: "OldType", ConnectionDetails: "{\"old\":\"details\"}"})

	// Test successful update with EntityID
	payload := `{"name": "New DS Name", "type": "NewType", "connection_details": "{\"new\":\"details\"}", "entity_id": "` + entity.ID + `"}`
	w := performRequest(testRouter, "PUT", "/api/v1/datasources/"+createdDS.ID, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var ds DataSourceConfig
	err := json.Unmarshal(w.Body.Bytes(), &ds)
	assert.NoError(t, err)
	assert.Equal(t, "New DS Name", ds.Name)
	assert.Equal(t, "NewType", ds.Type)
	assert.Equal(t, entity.ID, ds.EntityID)
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

	attribute, err := testStore.CreateAttribute(entity.ID, "FM Test Attribute", "string", "Attribute for FieldMapping tests", false, false, false)
	require.NoError(t, err)

	dataSource, err := testStore.CreateDataSource(DataSourceConfig{
		Name:              "FM Test DataSource",
		Type:              "CSV",
		ConnectionDetails: `{"path":"/dev/null"}`,
	})
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
		"transformation_rule": "lowercase"
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
	assert.Equal(t, dataSource.ID, fm.SourceID) // SourceID should be set by API

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
	var fms []DataSourceFieldMapping
	err := json.Unmarshal(w.Body.Bytes(), &fms)
	assert.NoError(t, err)
	assert.Len(t, fms, 0)

	// Create a mapping
	_, err = testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "col1",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
	})
	require.NoError(t, err)

	w = performRequest(testRouter, "GET", url, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &fms)
	assert.NoError(t, err)
	assert.Len(t, fms, 1)
	assert.Equal(t, "col1", fms[0].SourceFieldName)

	// Test list for non-existent source_id
	invalidUrl := fmt.Sprintf("/api/v1/datasources/%s/mappings/", "nonexistent-source-id")
	w = performRequest(testRouter, "GET", invalidUrl, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code) // API should check if dataSource exists
}

func TestGetFieldMappingHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	entity, attribute, dataSource := setupPrerequisitesForFieldMappingTests(t)

	fm, err := testStore.CreateFieldMapping(DataSourceFieldMapping{
		SourceID:        dataSource.ID,
		SourceFieldName: "col_to_get",
		EntityID:        entity.ID,
		AttributeID:     attribute.ID,
	})
	require.NoError(t, err)

	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/%s", dataSource.ID, fm.ID)
	w := performRequest(testRouter, "GET", url, nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var fetchedFm DataSourceFieldMapping
	err = json.Unmarshal(w.Body.Bytes(), &fetchedFm)
	assert.NoError(t, err)
	assert.Equal(t, fm.ID, fetchedFm.ID)
	assert.Equal(t, "col_to_get", fetchedFm.SourceFieldName)

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
	})
	require.NoError(t, err)

	// Create another attribute to test updating attribute_id
	anotherAttribute, err := testStore.CreateAttribute(entity.ID, "Another Attribute", "integer", "Another desc", false, false, false)
	require.NoError(t, err)

	payload := fmt.Sprintf(`{
		"source_field_name": "new_source_name",
		"entity_id": "%s",
		"attribute_id": "%s",
		"transformation_rule": "trim"
	}`, entity.ID, anotherAttribute.ID) // Use the new attribute ID

	url := fmt.Sprintf("/api/v1/datasources/%s/mappings/%s", dataSource.ID, fm.ID)
	w := performRequest(testRouter, "PUT", url, strings.NewReader(payload), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var updatedFm DataSourceFieldMapping
	err = json.Unmarshal(w.Body.Bytes(), &updatedFm)
	assert.NoError(t, err)
	assert.Equal(t, "new_source_name", updatedFm.SourceFieldName)
	assert.Equal(t, "trim", updatedFm.TransformationRule)
	assert.Equal(t, anotherAttribute.ID, updatedFm.AttributeID) // Check if attribute_id was updated
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
	validPayload := `{"name": "Daily Ingest", "description": "Ingests data daily", "cron_expression": "0 0 * * *", "task_type": "ingest_data_source", "task_parameters": "{\"source_id\":\"uuid-for-source\"}", "is_enabled": true}`
	w := performRequest(testRouter, "POST", "/api/v1/schedules/", strings.NewReader(validPayload), nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	var createdSchedule ScheduleDefinition
	err := json.Unmarshal(w.Body.Bytes(), &createdSchedule)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdSchedule.ID)
	assert.Equal(t, "Daily Ingest", createdSchedule.Name)
	assert.Equal(t, "0 0 * * *", createdSchedule.CronExpression)
	assert.Equal(t, "ingest_data_source", createdSchedule.TaskType)
	assert.Equal(t, `{"source_id":"uuid-for-source"}`, createdSchedule.TaskParameters) // Stored as string
	assert.True(t, createdSchedule.IsEnabled)
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
	var schedules []ScheduleDefinition
	err := json.Unmarshal(w.Body.Bytes(), &schedules)
	assert.NoError(t, err)
	assert.Len(t, schedules, 0)

	// Create some schedules directly via store for setup
	_, _ = testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Schedule A", CronExpression: "0 * * * *", TaskType: "typeA", TaskParameters: "{}"})
	_, _ = testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Schedule B", CronExpression: "30 * * * *", TaskType: "typeB", TaskParameters: "{}"})

	w = performRequest(testRouter, "GET", "/api/v1/schedules/", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &schedules)
	assert.NoError(t, err)
	assert.Len(t, schedules, 2)
}

func TestGetScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")

	createdSchedule, err := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "My Schedule", CronExpression: "15 * * * *", TaskType: "myTask", TaskParameters: `{"key":"val"}`})
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

	// Test retrieval of non-existent schedule
	w = performRequest(testRouter, "GET", "/api/v1/schedules/nonexistent-id", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateScheduleDefinitionHandler(t *testing.T) {
	require.NoError(t, clearAllTables(testStore), "Failed to clear tables before test")
	createdSchedule, err := testStore.CreateScheduleDefinition(ScheduleDefinition{Name: "Old Name", CronExpression: "0 0 * * *", TaskType: "old_type", TaskParameters: `{"old_param":"old_val"}`, IsEnabled: false})
	require.NoError(t, err)

	// Test successful update
	updatePayload := `{"name": "New Name", "description": "Updated desc", "cron_expression": "0 1 * * *", "task_type": "new_type", "task_parameters": "{\"new_param\":\"new_val\"}", "is_enabled": true}`
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
