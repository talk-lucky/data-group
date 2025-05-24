package ingestion

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock MetadataServiceAPIClient ---
type MockMetadataServiceClient struct {
	GetDataSourceConfigFunc func(sourceID string) (*DataSourceConfig, error)
}

func (m *MockMetadataServiceClient) GetDataSourceConfig(sourceID string) (*DataSourceConfig, error) {
	if m.GetDataSourceConfigFunc != nil {
		return m.GetDataSourceConfigFunc(sourceID)
	}
	return nil, fmt.Errorf("GetDataSourceConfigFunc not implemented")
}

// --- Mock ProcessingServiceAPIClient ---
type MockProcessingServiceClient struct {
	CallProcessDataFunc   func(payload ProcessDataRequest) error
	CapturedProcessDataRequest *ProcessDataRequest
}

func (m *MockProcessingServiceClient) CallProcessData(payload ProcessDataRequest) error {
	m.CapturedProcessDataRequest = &payload // Capture the payload
	if m.CallProcessDataFunc != nil {
		return m.CallProcessDataFunc(payload)
	}
	return nil // Default to success
}

// Helper to create a temporary CSV file for testing
func createTempCSV(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_*.csv")
	require.NoError(t, err, "Failed to create temp CSV file")

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err, "Failed to write to temp CSV file")
	err = tmpFile.Close()
	require.NoError(t, err, "Failed to close temp CSV file")

	// t.Cleanup() will automatically remove TempDir after test, so direct file removal isn't strictly needed here.
	// However, if not using t.TempDir(), direct cleanup is good:
	// t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	return tmpFile.Name()
}

func TestIngestData_CSV(t *testing.T) {
	mockMetaClient := &MockMetadataServiceClient{}
	mockProcClient := &MockProcessingServiceClient{}
	service := NewIngestionService(mockMetaClient, mockProcClient)

	t.Run("Successful CSV Ingestion", func(t *testing.T) {
		csvContent := "id,name,value\n1,productA,100\n2,productB,200"
		csvFilePath := createTempCSV(t, csvContent)

		connDetails := fmt.Sprintf(`{"filepath": "%s"}`, csvFilePath) // Use valid JSON string
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{
				ID:                "csvSource1",
				Name:              "Test CSV Source",
				Type:              "csv",
				ConnectionDetails: connDetails,
				EntityID:          "OrderEvents", // Used as EntityTypeName
			}, nil
		}
		mockProcClient.CallProcessDataFunc = func(payload ProcessDataRequest) error {
			return nil // Success
		}
		mockProcClient.CapturedProcessDataRequest = nil // Reset capture

		sourceID := "csvSource1"
		results, err := service.IngestData(sourceID)

		require.NoError(t, err)
		require.NotNil(t, results)
		require.Len(t, results, 2)

		assert.Equal(t, "1", results[0]["id"])
		assert.Equal(t, "productA", results[0]["name"])
		assert.Equal(t, "100", results[0]["value"])

		assert.Equal(t, "2", results[1]["id"])
		assert.Equal(t, "productB", results[1]["name"])
		assert.Equal(t, "200", results[1]["value"])

		require.NotNil(t, mockProcClient.CapturedProcessDataRequest, "Processing client should have been called")
		assert.Equal(t, sourceID, mockProcClient.CapturedProcessDataRequest.SourceID)
		assert.Equal(t, "OrderEvents", mockProcClient.CapturedProcessDataRequest.EntityTypeName)
		assert.Equal(t, results, mockProcClient.CapturedProcessDataRequest.RawData)
	})

	t.Run("CSV File Not Found", func(t *testing.T) {
		nonExistentFilePath := "/tmp/this/path/should/really/not/exist/file.csv"
		connDetails := fmt.Sprintf(`{"filepath": "%s"}`, nonExistentFilePath)
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{
				ID:                "csvSource_nonexistent",
				Type:              "csv",
				ConnectionDetails: connDetails,
			}, nil
		}

		_, err := service.IngestData("csvSource_nonexistent")
		require.Error(t, err)
		// Check if the error is related to file opening
		assert.True(t, strings.Contains(err.Error(), "failed to open CSV file") || os.IsNotExist(err), "Error should indicate file not found")
	})

	t.Run("Empty CSV File (Only Headers)", func(t *testing.T) {
		csvContent := "id,name\n" // Headers only
		csvFilePath := createTempCSV(t, csvContent)
		connDetails := fmt.Sprintf(`{"filepath": "%s"}`, csvFilePath)

		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: "csvSource_emptyheaders", Type: "csv", ConnectionDetails: connDetails, EntityID: "TestEntity"}, nil
		}
		mockProcClient.CapturedProcessDataRequest = nil // Reset

		results, err := service.IngestData("csvSource_emptyheaders")
		require.NoError(t, err)
		assert.Empty(t, results, "Results should be empty for a CSV with only headers")
		
		// Based on current IngestData logic, CallProcessData is skipped if len(results) == 0
		assert.Nil(t, mockProcClient.CapturedProcessDataRequest, "Processing client should not be called for empty results")
	})

	t.Run("CSV File with No Rows (Empty File)", func(t *testing.T) {
		csvFilePath := createTempCSV(t, "") // Empty content
		connDetails := fmt.Sprintf(`{"filepath": "%s"}`, csvFilePath)

		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: "csvSource_emptyfile", Type: "csv", ConnectionDetails: connDetails}, nil
		}

		results, err := service.IngestData("csvSource_emptyfile")
		// The current implementation returns empty results and no error if the file is completely empty
		// because reader.Read() for headers returns io.EOF immediately.
		// Let's adjust the assertion if that's the intended behavior from service.go
		// The service.go has:
		//   if err == io.EOF { log.Printf(...); return results, nil }
		// So, no error is expected, and results will be empty.
		require.NoError(t, err, "Error should be nil for a completely empty CSV file due to EOF on header read")
		assert.Empty(t, results, "Results should be empty for a completely empty CSV file")
	})

	t.Run("Invalid JSON in ConnectionDetails", func(t *testing.T) {
		invalidConnDetails := `{"filepath": "/path/to/file.csv"` // Missing closing brace
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: "csvSource_badjson", Type: "csv", ConnectionDetails: invalidConnDetails}, nil
		}

		_, err := service.IngestData("csvSource_badjson")
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to parse CSV connection details"), "Error should indicate JSON parsing failure")
	})

	t.Run("Missing Filepath in ConnectionDetails", func(t *testing.T) {
		missingFilepathConnDetails := `{"some_other_param": "value"}` // Filepath key is missing
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: "csvSource_nofilepath", Type: "csv", ConnectionDetails: missingFilepathConnDetails}, nil
		}

		_, err := service.IngestData("csvSource_nofilepath")
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "filepath is required for CSV data source type"), "Error should indicate missing filepath")
	})

	t.Run("Unsupported DataSource Type", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return &DataSourceConfig{ID: "unsupportedSource", Type: "SQLITE", ConnectionDetails: "{}"}, nil
		}

		_, err := service.IngestData("unsupportedSource")
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "unsupported data source type: SQLITE"), "Error should indicate unsupported type")
	})

	t.Run("GetDataSourceConfig Fails", func(t *testing.T) {
		mockMetaClient.GetDataSourceConfigFunc = func(sourceID string) (*DataSourceConfig, error) {
			return nil, fmt.Errorf("mock metadata service error")
		}
		_, err := service.IngestData("anySourceID")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get data source config")
		assert.Contains(t, err.Error(), "mock metadata service error")
	})
}
