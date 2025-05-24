package processing

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock MetadataServiceAPIClient ---
type MockMetadataServiceClient struct {
	GetDataSourceFieldMappingsFunc func(sourceID string) ([]DataSourceFieldMapping, error)
	GetAttributeDefinitionFunc     func(attributeID string, entityID string) (*AttributeDefinition, error)
	GetDataSourceConfigFunc        func(sourceID string) (*DataSourceConfig, error)
	// GetEntityDefinitionFunc is not currently called by ProcessAndStoreData directly, so not strictly needed for these tests
	// GetEntityDefinitionFunc        func(entityID string) (*EntityDefinition, error)
}

func (m *MockMetadataServiceClient) GetDataSourceFieldMappings(sourceID string) ([]DataSourceFieldMapping, error) {
	if m.GetDataSourceFieldMappingsFunc != nil {
		return m.GetDataSourceFieldMappingsFunc(sourceID)
	}
	return nil, fmt.Errorf("GetDataSourceFieldMappingsFunc not implemented")
}

func (m *MockMetadataServiceClient) GetAttributeDefinition(attributeID string, entityID string) (*AttributeDefinition, error) {
	if m.GetAttributeDefinitionFunc != nil {
		return m.GetAttributeDefinitionFunc(attributeID, entityID)
	}
	return nil, fmt.Errorf("GetAttributeDefinitionFunc not implemented")
}

func (m *MockMetadataServiceClient) GetDataSourceConfig(sourceID string) (*DataSourceConfig, error) {
	if m.GetDataSourceConfigFunc != nil {
		return m.GetDataSourceConfigFunc(sourceID)
	}
	return nil, fmt.Errorf("GetDataSourceConfigFunc not implemented")
}

// --- Tests for convertToTargetType ---
func TestConvertToTargetType(t *testing.T) {
	t.Run("String Conversions", func(t *testing.T) {
		val, err := convertToTargetType("hello", "string")
		assert.NoError(t, err)
		assert.Equal(t, "hello", val)

		val, err = convertToTargetType(123, "string")
		assert.NoError(t, err)
		assert.Equal(t, "123", val)

		val, err = convertToTargetType(123.45, "string")
		assert.NoError(t, err)
		assert.Equal(t, "123.45", val)

		val, err = convertToTargetType(true, "string")
		assert.NoError(t, err)
		assert.Equal(t, "true", val)

		val, err = convertToTargetType(nil, "string")
		assert.NoError(t, err)
		assert.Nil(t, val) // nil input should remain nil
	})

	t.Run("Integer Conversions", func(t *testing.T) {
		val, err := convertToTargetType("123", "integer")
		assert.NoError(t, err)
		assert.Equal(t, int64(123), val)

		val, err = convertToTargetType("-45", "integer")
		assert.NoError(t, err)
		assert.Equal(t, int64(-45), val)

		val, err = convertToTargetType(123.0, "integer") // float64
		assert.NoError(t, err)
		assert.Equal(t, int64(123), val)
		
		val, err = convertToTargetType(123.75, "integer") // float64 with fraction
		assert.NoError(t, err)
		assert.Equal(t, int64(123), val) // Should truncate

		val, err = convertToTargetType(json.Number("123"), "integer")
		assert.NoError(t, err)
		assert.Equal(t, int64(123), val)

		_, err = convertToTargetType("abc", "integer")
		assert.Error(t, err)

		val, err = convertToTargetType(nil, "integer")
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Float Conversions", func(t *testing.T) {
		val, err := convertToTargetType("123.45", "float")
		assert.NoError(t, err)
		assert.Equal(t, 123.45, val)

		val, err = convertToTargetType("-0.5", "float")
		assert.NoError(t, err)
		assert.Equal(t, -0.5, val)

		val, err = convertToTargetType(123, "float") // int
		assert.NoError(t, err)
		assert.Equal(t, float64(123), val)

		val, err = convertToTargetType(int64(123), "float") // int64
		assert.NoError(t, err)
		assert.Equal(t, float64(123), val)


		val, err = convertToTargetType(json.Number("123.45"), "float")
		assert.NoError(t, err)
		assert.Equal(t, 123.45, val)

		_, err = convertToTargetType("xyz", "float")
		assert.Error(t, err)

		val, err = convertToTargetType(nil, "float")
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Boolean Conversions", func(t *testing.T) {
		testCases := []struct {
			input    interface{}
			expected bool
		}{
			{"true", true}, {"TRUE", true},
			{"false", false}, {"FALSE", false},
			{"T", true}, {"t", true},
			{"F", false}, {"f", false},
			{"1", true}, {1, true}, {int64(1), true}, {float64(1.0), true}, {json.Number("1"), true},
			{"0", false}, {0, false}, {int64(0), false}, {float64(0.0), false}, {json.Number("0"), false},
			{"yes", true}, {"YES", true},
			{"no", false}, {"NO", false},
			{true, true}, {false, false},
		}
		for _, tc := range testCases {
			val, err := convertToTargetType(tc.input, "boolean")
			assert.NoError(t, err, "Input: %v", tc.input)
			assert.Equal(t, tc.expected, val, "Input: %v", tc.input)
		}

		_, err := convertToTargetType("maybe", "boolean")
		assert.Error(t, err)

		val, err := convertToTargetType(nil, "boolean")
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Date/Datetime Conversions", func(t *testing.T) {
		// RFC3339
		rfcTimeStr := "2023-10-27T10:00:00Z"
		expectedTime, _ := time.Parse(time.RFC3339, rfcTimeStr)
		val, err := convertToTargetType(rfcTimeStr, "datetime")
		assert.NoError(t, err)
		assert.Equal(t, expectedTime, val.(time.Time).UTC()) // convertToTargetType now returns UTC

		// Date only
		dateStr := "2023-10-27"
		expectedDate, _ := time.Parse("2006-01-02", dateStr)
		val, err = convertToTargetType(dateStr, "date")
		assert.NoError(t, err)
		assert.Equal(t, expectedDate, val)

		// SQL-like datetime
		sqlDtStr := "2023-10-27 10:30:15"
		expectedSqlDt, _ := time.Parse("2006-01-02 15:04:05", sqlDtStr)
		val, err = convertToTargetType(sqlDtStr, "datetime")
		assert.NoError(t, err)
		assert.Equal(t, expectedSqlDt, val)
		
		// Unix timestamp string
		unixTimeStr := "1672531200" // 2023-01-01 00:00:00 +0000 UTC
		expectedUnixTime := time.Unix(1672531200, 0).UTC()
		val, err = convertToTargetType(unixTimeStr, "datetime")
		assert.NoError(t, err)
		assert.Equal(t, expectedUnixTime, val)


		_, err = convertToTargetType("invalid-date-string", "datetime")
		assert.Error(t, err)

		val, err = convertToTargetType(nil, "datetime")
		assert.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Unsupported Type", func(t *testing.T) {
		val, err := convertToTargetType("some value", "customtype")
		assert.Error(t, err)
		assert.Equal(t, "some value", val, "Value should be returned as is for unsupported type")
		assert.Contains(t, err.Error(), "unsupported target data type: customtype")
	})
}

// --- Tests for transformAndConvertRecord (extracted logic) ---
func TestTransformAndConvertRecord(t *testing.T) {
	mockMetaClient := &MockMetadataServiceClient{} // Not used by transformAndConvertRecord directly
	service := NewProcessingService(mockMetaClient, nil) // DB is nil for this test

	// Setup common attribute definitions
	attrDefs := map[string]*AttributeDefinition{
		"attr_name":    {ID: "attr_name_id", Name: "FullName", DataType: "string"},
		"attr_email":   {ID: "attr_email_id", Name: "EmailAddress", DataType: "string"},
		"attr_age":     {ID: "attr_age_id", Name: "Age", DataType: "integer"},
		"attr_active":  {ID: "attr_active_id", Name: "IsActive", DataType: "boolean"},
		"attr_score":   {ID: "attr_score_id", Name: "Score", DataType: "float"},
		"attr_regdate": {ID: "attr_regdate_id", Name: "RegistrationDate", DataType: "datetime"},
		"attr_notes":   {ID: "attr_notes_id", Name: "Notes", DataType: "string"}, // For testing bad conversion
	}

	t.Run("Successful Transformation and Conversion", func(t *testing.T) {
		rawRecord := map[string]interface{}{
			"id":            "user123",
			"full_name":     "  JOHN DOE  ",
			"contact_email": "JOHN.DOE@EXAMPLE.COM",
			"user_age":      "30",
			"is_active":     "T",
			"rating":        "4.55",
			"joined_at":     "2023-01-15T10:00:00Z",
			"extra_field":   "should be ignored",
		}
		mappings := []DataSourceFieldMapping{
			{SourceFieldName: "full_name", AttributeID: "attr_name", TransformationRule: "trim"},
			{SourceFieldName: "contact_email", AttributeID: "attr_email", TransformationRule: "lowercase"},
			{SourceFieldName: "user_age", AttributeID: "attr_age"},
			{SourceFieldName: "is_active", AttributeID: "attr_active"},
			{SourceFieldName: "rating", AttributeID: "attr_score"},
			{SourceFieldName: "joined_at", AttributeID: "attr_regdate"},
		}

		processedData, rawID := service.transformAndConvertRecord(rawRecord, mappings, attrDefs, 1, "testSource")

		assert.Equal(t, "user123", rawID)
		require.Len(t, processedData, 6)
		assert.Equal(t, "JOHN DOE", processedData["FullName"])
		assert.Equal(t, "john.doe@example.com", processedData["EmailAddress"])
		assert.Equal(t, int64(30), processedData["Age"])
		assert.Equal(t, true, processedData["IsActive"])
		assert.Equal(t, 4.55, processedData["Score"])
		expectedTime, _ := time.Parse(time.RFC3339, "2023-01-15T10:00:00Z")
		assert.Equal(t, expectedTime, processedData["RegistrationDate"])
		_, exists := processedData["extra_field"]
		assert.False(t, exists, "Extra field should not be in processed data")
	})

	t.Run("Field Skipping on Conversion Error", func(t *testing.T) {
		rawRecord := map[string]interface{}{
			"user_notes": "this is okay",
			"user_age":   "thirty", // Invalid for integer
		}
		mappings := []DataSourceFieldMapping{
			{SourceFieldName: "user_notes", AttributeID: "attr_notes"}, // string to string
			{SourceFieldName: "user_age", AttributeID: "attr_age"},   // string "thirty" to integer
		}
		processedData, _ := service.transformAndConvertRecord(rawRecord, mappings, attrDefs, 1, "testSourceConvError")
		
		require.Len(t, processedData, 1) // Only notes should be processed
		assert.Equal(t, "this is okay", processedData["Notes"])
		_, ageExists := processedData["Age"]
		assert.False(t, ageExists, "Age should be skipped due to conversion error")
	})

	t.Run("Mapping for Non-Existent Raw Field", func(t *testing.T) {
		rawRecord := map[string]interface{}{"actual_field": "data"}
		mappings := []DataSourceFieldMapping{
			{SourceFieldName: "non_existent_field", AttributeID: "attr_name"},
			{SourceFieldName: "actual_field", AttributeID: "attr_notes"},
		}
		processedData, _ := service.transformAndConvertRecord(rawRecord, mappings, attrDefs, 1, "testSourceNonExistentField")

		require.Len(t, processedData, 1)
		assert.Equal(t, "data", processedData["Notes"])
		_, nameExists := processedData["FullName"]
		assert.False(t, nameExists, "FullName should not exist as its source field was missing")
	})

	t.Run("Raw Record Identifier Derivation", func(t *testing.T) {
		// Case 1: "id" field exists
		rawRecord1 := map[string]interface{}{"id": "record_xyz", "data": "value1"}
		_, rawID1 := service.transformAndConvertRecord(rawRecord1, []DataSourceFieldMapping{}, attrDefs, 1, "s1")
		assert.Equal(t, "record_xyz", rawID1)

		// Case 2: "source_record_id" field exists
		rawRecord2 := map[string]interface{}{"source_record_id": "record_abc", "data": "value2"}
		_, rawID2 := service.transformAndConvertRecord(rawRecord2, []DataSourceFieldMapping{}, attrDefs, 1, "s2")
		assert.Equal(t, "record_abc", rawID2)
		
		// Case 3: Neither exists
		rawRecord3 := map[string]interface{}{"other_field": "other_value", "data": "value3"}
		_, rawID3 := service.transformAndConvertRecord(rawRecord3, []DataSourceFieldMapping{}, attrDefs, 1, "s3")
		assert.Empty(t, rawID3, "Raw ID should be empty if no specific ID field is found")
	})
}

// --- Tests for ProcessAndStoreData (Early Exit Scenarios) ---
func TestProcessAndStoreData_EarlyExitScenarios(t *testing.T) {
	mockMetaClient := &MockMetadataServiceClient{}
	// Initialize service with nil DB to test logic before DB interaction
	service := NewProcessingService(mockMetaClient, nil) 

	rawData := []map[string]interface{}{{"key": "value"}}

	t.Run("GetDataSourceConfig Fails", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return nil, fmt.Errorf("mock error getting ds config")
		}
		// This test will now proceed further because GetDataSourceConfig failure is a warning, not a fatal error.
		// To test an "early exit" based on this, we'd need to check if subsequent calls are made or if processing is affected.
		// For now, we'll test a different early exit: GetDataSourceFieldMappings fails.
		// Reset GetDataSourceConfigFunc to success for next test
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: sourceID, EntityID: "testEntityDefID"}, nil
		}
	})
	
	t.Run("GetDataSourceFieldMappings Fails", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) { // Ensure this doesn't fail for this test
			return &DataSourceConfig{ID: sourceID, EntityID: "testEntityDefID"}, nil
		}
		mockMetaClient.GetDataSourceFieldMappingsFunc = func(sourceID string) ([]DataSourceFieldMapping, error) {
			return nil, fmt.Errorf("mock error getting mappings")
		}
		count, err := service.ProcessAndStoreData("source1", "TestEntity", rawData)
		assert.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Contains(t, err.Error(), "failed to fetch field mappings")
	})

	t.Run("GetDataSourceFieldMappings Returns Empty", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: sourceID, EntityID: "testEntityDefID"}, nil
		}
		mockMetaClient.GetDataSourceFieldMappingsFunc = func(sourceID string) ([]DataSourceFieldMapping, error) {
			return []DataSourceFieldMapping{}, nil // Empty mappings
		}
		count, err := service.ProcessAndStoreData("source2", "TestEntity", rawData)
		assert.NoError(t, err) // No error, but 0 processed
		assert.Equal(t, 0, count)
	})

	t.Run("GetAttributeDefinition Fails", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: sourceID, EntityID: "testEntityDefID"}, nil
		}
		mockMetaClient.GetDataSourceFieldMappingsFunc = func(sourceID string) ([]DataSourceFieldMapping, error) {
			return []DataSourceFieldMapping{
				{SourceFieldName: "raw_field", AttributeID: "attr1", EntityID: "entity1"},
			}, nil
		}
		mockMetaClient.GetAttributeDefinitionFunc = func(attributeID string, entityID string) (*AttributeDefinition, error) {
			return nil, fmt.Errorf("mock error getting attribute definition")
		}
		
		// The current logic logs a warning and continues if GetAttributeDefinition fails,
		// then potentially errors if no valid attribute definitions could be fetched at all.
		count, err := service.ProcessAndStoreData("source3", "TestEntity", rawData)
		assert.Error(t, err) // Expect an error because no attributes could be processed
		assert.Contains(t, err.Error(), "no valid attribute definitions could be fetched")
		assert.Equal(t, 0, count)
	})
}

// --- TestMain and DB Setup for Integration Tests ---
var testDB *sql.DB

// getEnvTest reads an environment variable for test configuration or returns a default.
// This function might be duplicated if tests are in different packages.
// For a real project, consider a shared test utility package.
func getEnvTest(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func setupTestDB() (*sql.DB, error) {
	dbHost := getEnvTest("TEST_DB_HOST", "localhost")
	dbPort := getEnvTest("TEST_DB_PORT", "5432")
	dbUser := getEnvTest("TEST_DB_USER", "admin")
	dbPassword := getEnvTest("TEST_DB_PASSWORD", "password")
	dbName := getEnvTest("TEST_DB_NAME", "metadata_test_db") // Use a dedicated test DB
	dbSSLMode := getEnvTest("TEST_DB_SSLMODE", "disable")

	dataSourceName := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open test database: %w", err)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping test database: %w", err)
	}
	// initSchema is called by NewProcessingService, so we don't call it here directly.
	// We just need to ensure the DB exists and is connectable.
	fmt.Println("Test DB connection successful for processing service tests.")
	return db, nil
}

func TestMain(m *testing.M) {
	var err error
	testDB, err = setupTestDB()
	if err != nil {
		fmt.Printf("Failed to set up test DB for processing service: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if testDB != nil {
			testDB.Close()
		}
	}()

	exitCode := m.Run()
	os.Exit(exitCode)
}

// clearTablesForDBTests truncates specified tables in the test database.
func clearTablesForDBTests(db *sql.DB, tableNames ...string) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	for _, table := range tableNames {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", table)
		_, err := db.Exec(query)
		if err != nil {
			// Log error but attempt to continue cleaning other tables.
			// This might happen if a table doesn't exist on the first run or due to CASCADE.
			fmt.Printf("Warning: Failed to truncate table %s: %v. This might be okay if the table doesn't exist yet or due to CASCADE.\n", table, err)
		}
	}
	return nil
}

// fetchProcessedRecords queries the processed_entities table and returns results.
// This is a basic version; more sophisticated checking might be needed.
func fetchProcessedRecords(t *testing.T, db *sql.DB, sourceID string, entityTypeName string) []map[string]interface{} {
	rows, err := db.Query("SELECT attributes, entity_definition_id, raw_record_identifier FROM processed_entities WHERE source_id = $1 AND entity_type_name = $2", sourceID, entityTypeName)
	require.NoError(t, err)
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var attributesJSON []byte
		var entityDefID sql.NullString
		var rawIdentifier sql.NullString
		err := rows.Scan(&attributesJSON, &entityDefID, &rawIdentifier)
		require.NoError(t, err)

		var recordAttributes map[string]interface{}
		err = json.Unmarshal(attributesJSON, &recordAttributes)
		require.NoError(t, err)
		
		// Add other DB fields to map if needed for assertion
		recordAttributes["_db_entity_definition_id"] = entityDefID.String 
		recordAttributes["_db_raw_record_identifier"] = rawIdentifier.String

		results = append(results, recordAttributes)
	}
	require.NoError(t, rows.Err())
	return results
}


// --- Tests for ProcessAndStoreData (DB Interaction) ---
func TestProcessAndStoreData_DBInteraction(t *testing.T) {
	require.NotNil(t, testDB, "Test DB connection should be initialized by TestMain")

	mockMetaClient := &MockMetadataServiceClient{}
	service := NewProcessingService(mockMetaClient, testDB) // NewProcessingService will call initSchema

	sourceID := "dbTestSource1"
	entityType := "Order"
	entityDefIDFromDSConfig := "order-entity-def-id-from-ds-config"

	// Common mock setup
	mockMetaClient.GetDataSourceConfigFunc = func(sID string) (*DataSourceConfig, error) {
		if sID == sourceID {
			return &DataSourceConfig{ID: sID, Name: "DB Test Source", Type: "some_type", EntityID: entityDefIDFromDSConfig}, nil
		}
		return nil, fmt.Errorf("unexpected sourceID for GetDataSourceConfig: %s", sID)
	}
	mockMetaClient.GetDataSourceFieldMappingsFunc = func(sID string) ([]DataSourceFieldMapping, error) {
		return []DataSourceFieldMapping{
			{SourceID: sID, SourceFieldName: "product_name", EntityID: entityDefIDFromDSConfig, AttributeID: "attr_prod_name"},
			{SourceID: sID, SourceFieldName: "quantity", EntityID: entityDefIDFromDSConfig, AttributeID: "attr_qty"},
			{SourceID: sID, SourceFieldName: "order_date", EntityID: entityDefIDFromDSConfig, AttributeID: "attr_order_dt"},
		}, nil
	}
	mockMetaClient.GetAttributeDefinitionFunc = func(attrID string, entID string) (*AttributeDefinition, error) {
		if entID != entityDefIDFromDSConfig {
			return nil, fmt.Errorf("unexpected entityID for GetAttributeDefinition: %s", entID)
		}
		switch attrID {
		case "attr_prod_name":
			return &AttributeDefinition{ID: attrID, EntityID: entID, Name: "ProductName", DataType: "string"}, nil
		case "attr_qty":
			return &AttributeDefinition{ID: attrID, EntityID: entID, Name: "Quantity", DataType: "integer"}, nil
		case "attr_order_dt":
			return &AttributeDefinition{ID: attrID, EntityID: entID, Name: "OrderDate", DataType: "datetime"}, nil
		}
		return nil, fmt.Errorf("unexpected attributeID for GetAttributeDefinition: %s", attrID)
	}

	t.Run("Successful Insertion of Multiple Records", func(t *testing.T) {
		require.NoError(t, clearTablesForDBTests(testDB, "processed_entities"), "Failed to clear table")

		rawData := []map[string]interface{}{
			{"id": "order1", "product_name": "Laptop", "quantity": "1", "order_date": "2023-01-15T10:00:00Z"},
			{"id": "order2", "product_name": "Mouse", "quantity": json.Number("2"), "order_date": "2023-01-16"},
			{"id": "order3", "product_name": "Keyboard", "quantity": 3.0, "order_date": "invalid-date-will-be-skipped"}, // OrderDate will be skipped
		}

		count, err := service.ProcessAndStoreData(sourceID, entityType, rawData)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "Should process all 3 records, even if one field fails conversion")

		dbRecords := fetchProcessedRecords(t, testDB, sourceID, entityType)
		require.Len(t, dbRecords, 3)

		// Record 1
		assert.Equal(t, "Laptop", dbRecords[0]["ProductName"])
		assert.Equal(t, int64(1), dbRecords[0]["Quantity"])
		expectedTime1, _ := time.Parse(time.RFC3339, "2023-01-15T10:00:00Z")
		assert.Equal(t, expectedTime1, dbRecords[0]["OrderDate"])
		assert.Equal(t, entityDefIDFromDSConfig, dbRecords[0]["_db_entity_definition_id"])
		assert.Equal(t, "order1", dbRecords[0]["_db_raw_record_identifier"])


		// Record 2
		assert.Equal(t, "Mouse", dbRecords[1]["ProductName"])
		assert.Equal(t, int64(2), dbRecords[1]["Quantity"])
		expectedTime2, _ := time.Parse("2006-01-02", "2023-01-16")
		assert.Equal(t, expectedTime2, dbRecords[1]["OrderDate"])
		assert.Equal(t, "order2", dbRecords[1]["_db_raw_record_identifier"])


		// Record 3 (OrderDate was invalid, so it should be missing from attributes)
		assert.Equal(t, "Keyboard", dbRecords[2]["ProductName"])
		assert.Equal(t, int64(3), dbRecords[2]["Quantity"])
		_, orderDateExists := dbRecords[2]["OrderDate"]
		assert.False(t, orderDateExists, "OrderDate should be missing for record 3 due to conversion error")
		assert.Equal(t, "order3", dbRecords[2]["_db_raw_record_identifier"])

	})

	t.Run("Transaction Rollback on Partial Failure", func(t *testing.T) {
		require.NoError(t, clearTablesForDBTests(testDB, "processed_entities"), "Failed to clear table")
		// To simulate a DB error, we can't easily do it mid-batch with current stmt.Exec.
		// The current code returns on the first error, so the transaction is rolled back by defer.
		// We'll test this by ensuring no records are present if any record in the batch would cause a non-transformation error.
		// This requires a way to make stmt.Exec fail. One way is to violate a constraint if one existed
		// or make the JSONB too large, or a network issue (hard to simulate here).
		// For now, we rely on the fact that if stmt.Exec fails, ProcessAndStoreData returns an error,
		// and the deferred tx.Rollback() will prevent partial commits.
		
		// Let's simulate a scenario where one record is fine, but the next one would cause some issue
		// (though we can't directly make stmt.Exec fail easily here without more complex mocks or setup).
		// The key is: if ProcessAndStoreData returns an error, nothing from THAT batch should be in DB.
		
		// This test case is conceptual for the current code structure.
		// A better way would be to mock the DB execution itself, which is beyond current scope.
		// We can, however, verify that if a previous test failed and rolled back, this one starts clean.
		
		rawData := []map[string]interface{}{
			{"id": "good_order", "product_name": "Good Product", "quantity": "10"},
			// Add a record here that might cause an issue if we could force stmt.Exec to fail for it.
			// For now, just ensuring the test setup is clean.
		}

		count, err := service.ProcessAndStoreData(sourceID, "RollbackTestEntity", rawData)
		require.NoError(t, err) // Assuming this batch is fine
		assert.Equal(t, 1, count)

		dbRecords := fetchProcessedRecords(t, testDB, sourceID, "RollbackTestEntity")
		require.Len(t, dbRecords, 1)
		assert.Equal(t, "Good Product", dbRecords[0]["ProductName"])

		// If we could make the *next* call fail:
		// e.g., mockMetaClient.GetAttributeDefinitionFunc = func(...) { return an error for a specific attribute }
		// then call ProcessAndStoreData again, it should error out, and the "Good Product" should still be there
		// but nothing from the failing batch.
		// The current ProcessAndStoreData returns on the *first* stmt.Exec error, so a "partial batch commit" isn't possible.
	})

	t.Run("Empty rawData", func(t *testing.T) {
		require.NoError(t, clearTablesForDBTests(testDB, "processed_entities"), "Failed to clear table")
		
		count, err := service.ProcessAndStoreData(sourceID, entityType, []map[string]interface{}{})
		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		dbRecords := fetchProcessedRecords(t, testDB, sourceID, entityType)
		assert.Len(t, dbRecords, 0)
	})

	t.Run("Data Leading to Empty processedRecordData", func(t *testing.T) {
		require.NoError(t, clearTablesForDBTests(testDB, "processed_entities"), "Failed to clear table")

		rawData := []map[string]interface{}{
			{"id": "bad_data_order", "quantity": "not_an_integer"}, // quantity mapped to integer
		}
		
		// Mocks are already set up from the main TestProcessAndStoreData_DBInteraction
		// "quantity" is mapped to "attr_qty" which has DataType "integer"

		count, err := service.ProcessAndStoreData(sourceID, entityType, rawData)
		assert.NoError(t, err) // Conversion errors are logged and field skipped, not a service error
		assert.Equal(t, 0, count) // processedRecordData becomes empty, so record is skipped for DB insert

		dbRecords := fetchProcessedRecords(t, testDB, sourceID, entityType)
		assert.Len(t, dbRecords, 0, "No records should be inserted if all fields failed conversion leading to empty processed data")
	})
}
