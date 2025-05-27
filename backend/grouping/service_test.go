package grouping

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// Assuming metadata types are accessible if they were defined in grouping/service.go
	// If they are in a separate metadata package, an import like:
	// metadata_pkg "example.com/project/backend/metadata"
	// would be needed, and types prefixed e.g., metadata_pkg.AttributeDefinition
	// For this exercise, assuming types like AttributeDefinition are defined within the grouping package
	// or are implicitly available (e.g. if they were moved into grouping/service.go previously)
)

// --- Mock MetadataServiceAPIClient ---
type MockMetadataServiceClient struct {
	GetGroupDefinitionFunc     func(groupID string) (*GroupDefinition, error)
	GetEntityDefinitionFunc    func(entityID string) (*EntityDefinition, error)
	GetAttributeDefinitionFunc func(entityID string, attributeID string) (*AttributeDefinition, error)
	ListWorkflowsFunc          func() ([]WorkflowDefinition, error)
}

func (m *MockMetadataServiceClient) GetGroupDefinition(groupID string) (*GroupDefinition, error) {
	if m.GetGroupDefinitionFunc != nil {
		return m.GetGroupDefinitionFunc(groupID)
	}
	return nil, fmt.Errorf("GetGroupDefinitionFunc not implemented")
}

func (m *MockMetadataServiceClient) GetEntityDefinition(entityID string) (*EntityDefinition, error) {
	if m.GetEntityDefinitionFunc != nil {
		return m.GetEntityDefinitionFunc(entityID)
	}
	return nil, fmt.Errorf("GetEntityDefinitionFunc not implemented")
}

func (m *MockMetadataServiceClient) GetAttributeDefinition(entityID string, attributeID string) (*AttributeDefinition, error) {
	if m.GetAttributeDefinitionFunc != nil {
		return m.GetAttributeDefinitionFunc(entityID, attributeID)
	}
	return nil, fmt.Errorf("GetAttributeDefinitionFunc not implemented")
}

func (m *MockMetadataServiceClient) ListWorkflows() ([]WorkflowDefinition, error) {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc()
	}
	return nil, fmt.Errorf("ListWorkflowsFunc not implemented")
}

// --- Mock OrchestrationServiceClient ---
type MockOrchestrationServiceClient struct {
	TriggerWorkflowFunc func(workflowID string) error
	TriggerWorkflowCalled bool
	LastTriggeredWorkflowID string
}

func (m *MockOrchestrationServiceClient) TriggerWorkflow(workflowID string) error {
	m.TriggerWorkflowCalled = true
	m.LastTriggeredWorkflowID = workflowID
	if m.TriggerWorkflowFunc != nil {
		return m.TriggerWorkflowFunc(workflowID)
	}
	return nil
}

// Helper to create json.RawMessage from a rule struct
func mustMarshalJSONRaw(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return json.RawMessage(raw)
}


// --- Tests for buildWhereClauseRecursive ---
func TestBuildWhereClauseRecursive(t *testing.T) {
	attrDefsMap := map[string]*AttributeDefinition{
		"age_attr_id":    {ID: "age_attr_id", Name: "Age", DataType: "integer"},
		"country_attr_id":{ID: "country_attr_id", Name: "Country", DataType: "string"},
		"active_attr_id": {ID: "active_attr_id", Name: "IsActive", DataType: "boolean"},
		"reg_attr_id":    {ID: "reg_attr_id", Name: "RegistrationDate", DataType: "datetime"},
		"tags_attr_id":   {ID: "tags_attr_id", Name: "Tags", DataType: "string"}, // for LIKE
		"desc_attr_id":   {ID: "desc_attr_id", EntityID: "user_entity", Name: "Description", DataType: "string"}, // for IS NULL
		"category_attr_id": {ID: "category_attr_id", EntityID: "user_entity", Name: "Category", DataType: "string"}, // for IN
		// Attributes for a related "Order" entity
		"order_amount_attr_id": {ID: "order_amount_attr_id", EntityID: "order_entity", Name: "OrderAmount", DataType: "numeric"},
		"order_date_attr_id":   {ID: "order_date_attr_id", EntityID: "order_entity", Name: "OrderDate", DataType: "datetime"},
		"user_fk_attr_id":      {ID: "user_fk_attr_id", EntityID: "order_entity", Name: "UserID", DataType: "string"}, // Foreign key in Order entity
		"user_pk_attr_id":      {ID: "user_pk_attr_id", EntityID: "user_entity", Name: "ID", DataType: "string"},   // Primary key in User entity
	}
	// Add EntityID to existing attributes
	for k, v := range attrDefsMap {
		if v.EntityID == "" { // Default to user_entity if not specified for older tests
			v.EntityID = "user_entity"
			attrDefsMap[k] = v
		}
	}


	relationshipDefsMap := map[string]*EntityRelationshipDefinition{
		"user_orders_rel_id": {
			ID:                "user_orders_rel_id",
			Name:              "UserOrders",
			SourceEntityID:    "user_entity",
			SourceAttributeID: "user_pk_attr_id", // User.ID
			TargetEntityID:    "order_entity",
			TargetAttributeID: "user_fk_attr_id", // Order.UserID
			RelationshipType:  OneToMany,
		},
	}

	// Mock metadata client for tests that might involve fetching relationships if not in map
	// (though for buildWhereClauseRecursive, we primarily rely on pre-fetched maps)
	mockMetaClient := &MockMetadataServiceClient{
		GetEntityRelationshipFunc: func(relationshipID string) (*EntityRelationshipDefinition, error) {
			if rel, ok := relationshipDefsMap[relationshipID]; ok {
				return rel, nil
			}
			return nil, fmt.Errorf("mock: relationship %s not found", relationshipID)
		},
		// Other funcs can be defined if needed by tests hitting CalculateGroup directly
	}
	_ = mockMetaClient // Avoid unused error for now, might be used in CalculateGroup tests later

	// Helper for alias generation in tests
	defaultAliasGenerator := func() func() string {
		c := 0
		return func() string {
			c++
			return fmt.Sprintf("pe%d", c)
		}
	}


	t.Run("Simple Integer Condition (Age >= 30)", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: ">=", Value: 30, ValueType: "integer"}),
			},
		}
		params := make([]interface{}, 0)
		paramCounter := 1
		// For simple conditions, currentTableAlias is the primary alias, groupEntityID is the ruleGroup.EntityID
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Age')::bigint >= $1", sql)
		assert.Equal(t, []interface{}{30}, params)
		assert.Equal(t, 2, paramCounter)
	})

	t.Run("String Equality (Country = USA)", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "USA", ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Country') = $1", sql)
		assert.Equal(t, []interface{}{"USA"}, params)
		assert.Equal(t, 2, paramCounter)
	})

	t.Run("Boolean Comparison (IsActive = true)", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "active_attr_id", AttributeName: "IsActive", Operator: "=", Value: true, ValueType: "boolean"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'IsActive')::boolean = $1", sql)
		assert.Equal(t, []interface{}{true}, params)
	})

	t.Run("Datetime Comparison (RegistrationDate < value)", func(t *testing.T) {
		dtValue := "2023-01-01T00:00:00Z"
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "reg_attr_id", AttributeName: "RegistrationDate", Operator: "<", Value: dtValue, ValueType: "datetime"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'RegistrationDate')::timestamptz < $1", sql)
		assert.Equal(t, []interface{}{dtValue}, params)
	})

	t.Run("LIKE (Tags contains 'tech')", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "tags_attr_id", AttributeName: "Tags", Operator: "contains", Value: "tech", ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Tags') LIKE $1", sql)
		assert.Equal(t, []interface{}{"%tech%"}, params)
	})

	t.Run("ILIKE (Tags ilike 'Tech')", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "tags_attr_id", AttributeName: "Tags", Operator: "ilike", Value: "Tech", ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Tags') ilike $1", sql)
		assert.Equal(t, []interface{}{"Tech"}, params)
	})

	t.Run("IN (Category IN ('A', 'B'))", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "category_attr_id", AttributeName: "Category", Operator: "in", Value: []interface{}{"A", "B"}, ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Category') IN ($1, $2)", sql)
		assert.Equal(t, []interface{}{"A", "B"}, params)
		assert.Equal(t, 3, paramCounter)
	})

	t.Run("IN with empty list (Category IN ())", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "category_attr_id", AttributeName: "Category", Operator: "in", Value: []interface{}{}, ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "FALSE", sql)
		assert.Empty(t, params)
		assert.Equal(t, 1, paramCounter) 
	})

	t.Run("IS NULL (Description IS NULL)", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "desc_attr_id", AttributeName: "Description", Operator: "is null", ValueType: "string"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Description') IS NULL", sql)
		assert.Empty(t, params)
	})

	t.Run("Nested Groups (Age > 30 AND (Country = USA OR Country = CAN))", func(t *testing.T) {
		nestedOrGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity", // Same entity for nested group
			LogicalOperator: "OR",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "USA", ValueType: "string"}),
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "CAN", ValueType: "string"}),
			},
		}
		mainAndGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: ">", Value: 30, ValueType: "integer"}),
				mustMarshalJSONRaw(t, nestedOrGroup),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(mainAndGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		expectedSQL := "(pe1.attributes->>'Age')::bigint > $1 AND ((pe1.attributes->>'Country') = $2 OR (pe1.attributes->>'Country') = $3)"
		assert.Equal(t, expectedSQL, sql)
		assert.Equal(t, []interface{}{30, "USA", "CAN"}, params)
		assert.Equal(t, 4, paramCounter)
	})

	t.Run("Invalid Operator", func(t *testing.T) {
		ruleGroup := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "AND", Rules: []json.RawMessage{
			mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: "INVALID_OP", Value: 30, ValueType: "integer"}),
		}}
		params := make([]interface{}, 0); paramCounter := 1
		_, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported operator: INVALID_OP")
	})

	t.Run("Missing Attribute Definition", func(t *testing.T) {
		ruleGroup := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "AND", Rules: []json.RawMessage{
			mustMarshalJSONRaw(t, RuleCondition{AttributeID: "non_existent_attr_id", AttributeName: "NonExistent", Operator: "=", Value: "X", ValueType: "string"}),
		}}
		params := make([]interface{}, 0); paramCounter := 1
		_, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "attribute definition not found for ID: 'non_existent_attr_id' (Name: 'NonExistent', EntityID: 'user_entity')")
	})

	t.Run("Group with single condition", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND", 
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: "<", Value: 10, ValueType: "integer"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Age')::bigint < $1", sql)
		assert.Equal(t, []interface{}{10}, params)
	})

	t.Run("Group with single nested group", func(t *testing.T) {
		nestedGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "OR",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "FR", ValueType: "string"}),
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "active_attr_id", AttributeName: "IsActive", Operator: "=", Value: false, ValueType: "boolean"}),
			},
		}
		mainGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND", 
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, nestedGroup),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(mainGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "((pe1.attributes->>'Country') = $1 OR (pe1.attributes->>'IsActive')::boolean = $2)", sql)
		assert.Equal(t, []interface{}{"FR", false}, params)
	})

	t.Run("Empty rules array in a group", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules:           []json.RawMessage{},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "", sql) 
		assert.Empty(t, params)
	})
	
	t.Run("Group with empty nested group", func(t *testing.T) {
		emptyNestedGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "OR",
			Rules:           []json.RawMessage{},
		}
		mainGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: "=", Value: 50}),
				mustMarshalJSONRaw(t, emptyNestedGroup), 
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "DE"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(mainGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Age')::bigint = $1 AND (pe1.attributes->>'Country') = $2", sql)
		assert.Equal(t, []interface{}{50, "DE"}, params)
	})


	t.Run("Deeply Nested Groups (4 levels)", func(t *testing.T) {
		level4Cond := RuleCondition{AttributeID: "active_attr_id", AttributeName: "IsActive", Operator: "=", Value: true, ValueType: "boolean"}
		level3Group := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "OR", Rules: []json.RawMessage{mustMarshalJSONRaw(t, level4Cond)}}
		level2Cond := RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "!=", Value: "US", ValueType: "string"}
		level2Group := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "AND", Rules: []json.RawMessage{mustMarshalJSONRaw(t, level2Cond), mustMarshalJSONRaw(t, level3Group)}}
		level1Cond := RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: ">", Value: 20, ValueType: "integer"}
		rootGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, level1Cond),
				mustMarshalJSONRaw(t, level2Group),
			},
		}

		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(rootGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		expectedSQL := "(pe1.attributes->>'Age')::bigint > $1 AND ((pe1.attributes->>'Country') != $2 AND ((pe1.attributes->>'IsActive')::boolean = $3))"
		assert.Equal(t, expectedSQL, sql)
		assert.Equal(t, []interface{}{20, "US", true}, params)
		assert.Equal(t, 4, paramCounter)
	})
	
	t.Run("Error: Malformed Rule - Unknown Type", func(t *testing.T) {
		malformedRule := `{"type": "unknown_type"}`
		ruleGroup := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "AND", Rules: []json.RawMessage{json.RawMessage(malformedRule)}}
		params := make([]interface{}, 0); paramCounter := 1
		_, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown rule type: 'unknown_type'")
	})
	
	t.Run("Error: Malformed RuleCondition - Missing AttributeID", func(t *testing.T) {
		ruleGroup := RuleGroup{Type: "group", EntityID: "user_entity", LogicalOperator: "AND", Rules: []json.RawMessage{
			mustMarshalJSONRaw(t, RuleCondition{AttributeName: "Age", Operator: ">=", Value: 30}), 
		}}
		params := make([]interface{}, 0); paramCounter := 1
		_, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "attribute definition not found for ID:") 
	})

	t.Run("Error: Malformed RuleGroup - Invalid LogicalOperator (e.g. empty)", func(t *testing.T) {
		ruleGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "", 
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: ">", Value: 20}),
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "country_attr_id", AttributeName: "Country", Operator: "=", Value: "CA"}),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(ruleGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		assert.Equal(t, "(pe1.attributes->>'Age')::bigint > $1 AND (pe1.attributes->>'Country') = $2", sql)
		assert.Equal(t, []interface{}{20, "CA"}, params)
	})


	// --- Tests for relationship_group ---
	t.Run("Simple relationship_group with one condition", func(t *testing.T) {
		relatedRules := RuleGroup{
			Type:     "group",
			EntityID: "order_entity", // Explicitly order_entity
			LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "order_amount_attr_id", AttributeName: "OrderAmount", Operator: ">", Value: 100, ValueType: "numeric"}),
			},
		}
		relationshipNode := RelationshipGroupNode{
			Type:               "relationship_group",
			RelationshipID:     "user_orders_rel_id",
			RelatedEntityRules: mustMarshalJSONRaw(t, relatedRules),
		}
		rootGroup := RuleGroup{
			Type:            "group",
			EntityID:        "user_entity",
			LogicalOperator: "AND",
			Rules:           []json.RawMessage{mustMarshalJSONRaw(t, relationshipNode)},
		}

		params := make([]interface{}, 0)
		paramCounter := 1
		sql, err := buildWhereClauseRecursive(rootGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		
		// Expected: EXISTS (SELECT 1 FROM processed_entities pe2 WHERE pe2.entity_definition_id = $1 AND (pe1.attributes->>'ID') = (pe2.attributes->>'UserID') AND ((pe2.attributes->>'OrderAmount')::numeric > $2))
		expectedSQL := "EXISTS (SELECT 1 FROM processed_entities pe2 WHERE pe2.entity_definition_id = $1 AND (pe1.attributes->>'ID') = (pe2.attributes->>'UserID') AND ((pe2.attributes->>'OrderAmount')::numeric > $2))"
		assert.Equal(t, expectedSQL, sql)
		assert.Equal(t, []interface{}{"order_entity", 100}, params)
		assert.Equal(t, 3, paramCounter) // $1 for entity_id, $2 for amount
	})

	t.Run("Relationship_group with nested group in related_entity_rules", func(t *testing.T) {
		nestedRelatedCond := RuleCondition{AttributeID: "order_date_attr_id", AttributeName: "OrderDate", Operator: ">", Value: "2023-01-01T00:00:00Z", ValueType: "datetime"}
		nestedRelatedGroup := RuleGroup{
			Type: "group", EntityID: "order_entity", LogicalOperator: "AND", 
			Rules: []json.RawMessage{mustMarshalJSONRaw(t, nestedRelatedCond)},
		}
		relatedRules := RuleGroup{
			Type:     "group",
			EntityID: "order_entity",
			LogicalOperator: "OR",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "order_amount_attr_id", AttributeName: "OrderAmount", Operator: "<", Value: 10, ValueType: "numeric"}),
				mustMarshalJSONRaw(t, nestedRelatedGroup),
			},
		}
		relationshipNode := RelationshipGroupNode{
			Type:               "relationship_group",
			RelationshipID:     "user_orders_rel_id",
			RelatedEntityRules: mustMarshalJSONRaw(t, relatedRules),
		}
		rootGroup := RuleGroup{
			Type: "group", EntityID: "user_entity", LogicalOperator: "AND",
			Rules: []json.RawMessage{mustMarshalJSONRaw(t, relationshipNode)},
		}

		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(rootGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)

		expectedSQL := "EXISTS (SELECT 1 FROM processed_entities pe2 WHERE pe2.entity_definition_id = $1 AND (pe1.attributes->>'ID') = (pe2.attributes->>'UserID') AND (((pe2.attributes->>'OrderAmount')::numeric < $2) OR ((pe2.attributes->>'OrderDate')::timestamptz > $3)))"
		assert.Equal(t, expectedSQL, sql)
		assert.Equal(t, []interface{}{"order_entity", 10, "2023-01-01T00:00:00Z"}, params)
		assert.Equal(t, 4, paramCounter)
	})

	t.Run("Primary entity conditions AND relationship_group", func(t *testing.T) {
		primaryCond := RuleCondition{AttributeID: "age_attr_id", AttributeName: "Age", Operator: "=", Value: 42, ValueType: "integer"}
		relatedRules := RuleGroup{
			Type: "group", EntityID: "order_entity", LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, RuleCondition{AttributeID: "order_amount_attr_id", AttributeName: "OrderAmount", Operator: ">", Value: 50}),
			},
		}
		relationshipNode := RelationshipGroupNode{
			Type: "relationship_group", RelationshipID: "user_orders_rel_id",
			RelatedEntityRules: mustMarshalJSONRaw(t, relatedRules),
		}
		rootGroup := RuleGroup{
			Type: "group", EntityID: "user_entity", LogicalOperator: "AND",
			Rules: []json.RawMessage{
				mustMarshalJSONRaw(t, primaryCond),
				mustMarshalJSONRaw(t, relationshipNode),
			},
		}
		params := make([]interface{}, 0); paramCounter := 1
		sql, err := buildWhereClauseRecursive(rootGroup, attrDefsMap, relationshipDefsMap, &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClient)
		require.NoError(t, err)
		
		expectedSQL := "(pe1.attributes->>'Age')::bigint = $1 AND EXISTS (SELECT 1 FROM processed_entities pe2 WHERE pe2.entity_definition_id = $2 AND (pe1.attributes->>'ID') = (pe2.attributes->>'UserID') AND ((pe2.attributes->>'OrderAmount')::numeric > $3))"
		assert.Equal(t, expectedSQL, sql)
		assert.Equal(t, []interface{}{42, "order_entity", 50}, params)
		assert.Equal(t, 4, paramCounter)
	})

	t.Run("Error: relationship_id not found in map", func(t *testing.T) {
		// This test relies on mockMetaClient returning an error if GetEntityRelationship is called for "non_existent_rel_id"
		// because it's not in relationshipDefsMap passed to buildWhereClauseRecursive.
		mockMetaClientLocal := &MockMetadataServiceClient{
			GetEntityRelationshipFunc: func(relationshipID string) (*EntityRelationshipDefinition, error) {
				// This mock will be called by buildWhereClauseRecursive if relDef not in map
				return nil, fmt.Errorf("mock: relationship %s definitely not found", relationshipID)
			},
		}

		relationshipNode := RelationshipGroupNode{
			Type: "relationship_group", RelationshipID: "non_existent_rel_id",
			RelatedEntityRules: mustMarshalJSONRaw(t, RuleGroup{Type:"group", EntityID:"order_entity", LogicalOperator:"AND", Rules:[]json.RawMessage{}}),
		}
		rootGroup := RuleGroup{
			Type: "group", EntityID: "user_entity", LogicalOperator: "AND",
			Rules: []json.RawMessage{mustMarshalJSONRaw(t, relationshipNode)},
		}
		params := make([]interface{}, 0); paramCounter := 1
		// Pass an empty relationshipDefsMap to force a fetch attempt via mockMetaClientLocal
		_, err := buildWhereClauseRecursive(rootGroup, attrDefsMap, make(map[string]*EntityRelationshipDefinition), &params, &paramCounter, "pe1", defaultAliasGenerator(), "user_entity", mockMetaClientLocal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "relationship definition non_existent_rel_id not found and failed to fetch")
		assert.Contains(t, err.Error(), "mock: relationship non_existent_rel_id definitely not found")
	})
}


// --- Tests for CalculateGroup (DB Interaction with sqlmock) ---
func TestCalculateGroup(t *testing.T) {
	mockMetaClient := &MockMetadataServiceClient{}
	mockOrchClient := &MockOrchestrationServiceClient{}

	defaultGroupDef := &GroupDefinition{
		ID:        "group1",
		Name:      "Test Group",
		EntityID:  "entity1",
		RulesJSON: `{"type": "group", "logical_operator": "AND", "rules": [{"type": "condition", "attribute_id": "attr1", "attribute_name": "Age", "entity_id": "entity1", "operator": ">=", "value": 30, "value_type": "integer"}]}`,
	}
	// EntityID in RuleCondition is "entity1", matching group's EntityID.
	defaultAttrDef := &AttributeDefinition{ID: "attr1", EntityID: "entity1", Name: "Age", DataType: "integer"}

	t.Run("Successful Calculation (Multiple Members)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		service := NewGroupingService(mockMetaClient, mockOrchClient, db)

		mockMetaClient.GetGroupDefinitionFunc = func(groupID string) (*GroupDefinition, error) { return defaultGroupDef, nil }
		mockMetaClient.GetAttributeDefinitionFunc = func(entityID string, attributeID string) (*AttributeDefinition, error) { return defaultAttrDef, nil }
		mockOrchClient.TriggerWorkflowCalled = false // Reset

		mock.ExpectBegin()
		// UPSERT log for 'CALCULATING'
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_calculation_logs (group_definition_id, entity_definition_id, calculated_at, member_count, status, error_message) VALUES ($1, $2, NOW(), $3, $4, $5) ON CONFLICT (group_definition_id) DO UPDATE SET")).
			WithArgs(defaultGroupDef.ID, defaultGroupDef.EntityID, 0, "CALCULATING", sqlmock.AnyArg()). // error_message is sql.NullString
			WillReturnResult(sqlmock.NewResult(1, 1))
		// DELETE old members
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM group_memberships WHERE group_definition_id = $1")).
			WithArgs(defaultGroupDef.ID).WillReturnResult(sqlmock.NewResult(0, 5)) // Say 5 old members were deleted
		
		// Main Query for members
		rows := sqlmock.NewRows([]string{"id"}).AddRow("uuid-1").AddRow("uuid-2")
		expectedSQL := regexp.QuoteMeta("SELECT id FROM processed_entities WHERE entity_definition_id = $1 AND ((attributes->>'Age')::bigint >= $2)")
		mock.ExpectQuery(expectedSQL).WithArgs(defaultGroupDef.EntityID, 30).WillReturnRows(rows)

		// INSERT new members
		mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO group_memberships (group_definition_id, processed_entity_instance_id) VALUES ($1, $2)"))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_memberships")).WithArgs(defaultGroupDef.ID, "uuid-1").WillReturnResult(sqlmock.NewResult(1,1))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_memberships")).WithArgs(defaultGroupDef.ID, "uuid-2").WillReturnResult(sqlmock.NewResult(1,1))
		
		// UPDATE log for 'COMPLETED'
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_calculation_logs (group_definition_id, entity_definition_id, calculated_at, member_count, status, error_message) VALUES ($1, $2, NOW(), $3, $4, $5) ON CONFLICT (group_definition_id) DO UPDATE SET")).
			WithArgs(defaultGroupDef.ID, defaultGroupDef.EntityID, 2, "COMPLETED", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		ids, err := service.CalculateGroup("group1")
		require.NoError(t, err)
		assert.Equal(t, []string{"uuid-1", "uuid-2"}, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
		// Orchestration client is called in a goroutine, direct check might be racy.
		// For unit tests, consider making it synchronous or using channels/waitgroups if testing orchestration call is critical.
		// For now, we assume it's called if commit is successful.
	})
	
	t.Run("Query Returns No Members", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		service := NewGroupingService(mockMetaClient, mockOrchClient, db)

		mockMetaClient.GetGroupDefinitionFunc = func(groupID string) (*GroupDefinition, error) { return defaultGroupDef, nil }
		mockMetaClient.GetAttributeDefinitionFunc = func(entityID string, attributeID string) (*AttributeDefinition, error) { return defaultAttrDef, nil }

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO group_calculation_logs").WillReturnResult(sqlmock.NewResult(1,1))
		mock.ExpectExec("DELETE FROM group_memberships").WillReturnResult(sqlmock.NewResult(0,0))
		
		rows := sqlmock.NewRows([]string{"id"}) // No rows added
		mock.ExpectQuery("SELECT id FROM processed_entities").WillReturnRows(rows)
		
		// No INSERT INTO group_memberships expected
		mock.ExpectExec("INSERT INTO group_calculation_logs").WithArgs(defaultGroupDef.ID, defaultGroupDef.EntityID, 0, "COMPLETED", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1,1))
		mock.ExpectCommit()

		ids, err := service.CalculateGroup("group1")
		require.NoError(t, err)
		assert.Empty(t, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Failure in Main Member Query", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		service := NewGroupingService(mockMetaClient, mockOrchClient, db)

		mockMetaClient.GetGroupDefinitionFunc = func(groupID string) (*GroupDefinition, error) { return defaultGroupDef, nil }
		mockMetaClient.GetAttributeDefinitionFunc = func(entityID string, attributeID string) (*AttributeDefinition, error) { return defaultAttrDef, nil }
		dbError := fmt.Errorf("DB query failed")

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO group_calculation_logs").WillReturnResult(sqlmock.NewResult(1,1)) // CALCULATING
		mock.ExpectExec("DELETE FROM group_memberships").WillReturnResult(sqlmock.NewResult(0,0))
		
		mock.ExpectQuery("SELECT id FROM processed_entities").WillReturnError(dbError)
		
		// Expect log update to FAILED
		mock.ExpectExec("INSERT INTO group_calculation_logs").WithArgs(defaultGroupDef.ID, defaultGroupDef.EntityID, 0, "FAILED", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1,1))
		mock.ExpectCommit() // Commit the FAILED status

		_, err = service.CalculateGroup("group1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB query failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("Failure Storing Members (First INSERT fails)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		service := NewGroupingService(mockMetaClient, mockOrchClient, db)

		mockMetaClient.GetGroupDefinitionFunc = func(groupID string) (*GroupDefinition, error) { return defaultGroupDef, nil }
		mockMetaClient.GetAttributeDefinitionFunc = func(entityID string, attributeID string) (*AttributeDefinition, error) { return defaultAttrDef, nil }
		insertError := fmt.Errorf("member insert failed")

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO group_calculation_logs").WillReturnResult(sqlmock.NewResult(1,1)) // CALCULATING
		mock.ExpectExec("DELETE FROM group_memberships").WillReturnResult(sqlmock.NewResult(0,0))
		
		rows := sqlmock.NewRows([]string{"id"}).AddRow("uuid-1").AddRow("uuid-2")
		mock.ExpectQuery("SELECT id FROM processed_entities").WillReturnRows(rows)

		mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO group_memberships (group_definition_id, processed_entity_instance_id) VALUES ($1, $2)"))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_memberships")).WithArgs(defaultGroupDef.ID, "uuid-1").WillReturnError(insertError)
		
		// Expect log update to FAILED
		// The member_count here is tricky, it might be the count *before* the error, or the total expected.
		// Current code logs len(entityInstanceIDs) which is 2 in this case.
		mock.ExpectExec("INSERT INTO group_calculation_logs").WithArgs(defaultGroupDef.ID, defaultGroupDef.EntityID, 2, "FAILED", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1,1))
		mock.ExpectCommit()

		_, err = service.CalculateGroup("group1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member insert failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Malformed RulesJSON (causes error in buildWhereClauseRecursive)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		service := NewGroupingService(mockMetaClient, mockOrchClient, db)

		malformedRulesGroupDef := &GroupDefinition{
			ID:        "groupMalformed",
			Name:      "Malformed Group",
			EntityID:  "entity1",
			RulesJSON: `{"type": "group", "logical_operator": "AND", "rules": [{"type": "condition", "attribute_id": "attr1", "attribute_name": "Age", "operator": "INVALID_OP", "value": 30}]}`, // INVALID_OP
		}
		// EntityID for attr1 is not specified in rule, will use group's EntityID.

		mockMetaClient.GetGroupDefinitionFunc = func(groupID string) (*GroupDefinition, error) {
			return malformedRulesGroupDef, nil
		}
		// GetAttributeDefinition will be called by getAllAttributeIDsAndNamesRecursive
		mockMetaClient.GetAttributeDefinitionFunc = func(entityID string, attributeID string) (*AttributeDefinition, error) {
			if entityID == "entity1" && attributeID == "attr1" {
				return &AttributeDefinition{ID: "attr1", EntityID: "entity1", Name: "Age", DataType: "integer"}, nil
			}
			return nil, fmt.Errorf("unexpected attribute fetch: %s/%s", entityID, attributeID)
		}


		mock.ExpectBegin()
		// UPSERT log for 'CALCULATING'
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_calculation_logs")).
			WithArgs(malformedRulesGroupDef.ID, malformedRulesGroupDef.EntityID, 0, "CALCULATING", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		// DELETE old members
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM group_memberships")).
			WithArgs(malformedRulesGroupDef.ID).
			WillReturnResult(sqlmock.NewResult(0, 0)) 
		
		// Expect log update to FAILED due to buildWhereClauseRecursive error
		// The error message will contain "unsupported operator: INVALID_OP"
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO group_calculation_logs")).
			WithArgs(malformedRulesGroupDef.ID, malformedRulesGroupDef.EntityID, 0, "FAILED", sqlmock.ที่มีArg(driver.Value(".*unsupported operator: INVALID_OP.*"))).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		_, calcErr := service.CalculateGroup(malformedRulesGroupDef.ID)
		require.Error(t, calcErr)
		assert.Contains(t, calcErr.Error(), "unsupported operator: INVALID_OP")
		assert.NoError(t, mock.ExpectationsWereMet())

	})

}


// --- Tests for GetGroupResults (DB Interaction with sqlmock) ---
func TestGetGroupResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	
	mockMetaClient := &MockMetadataServiceClient{} // Not used by GetGroupResults
	mockOrchClient := &MockOrchestrationServiceClient{} // Not used by GetGroupResults
	service := NewGroupingService(mockMetaClient, mockOrchClient, db)
	
	groupID := "groupTestGetResults"
	entityDefID := "entityTestGetResults"
	now := time.Now()

	t.Run("Successful Retrieval", func(t *testing.T) {
		logRows := sqlmock.NewRows([]string{"calculated_at", "status", "member_count"}).
			AddRow(now, "COMPLETED", 2)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT calculated_at, status, member_count FROM group_calculation_logs WHERE group_definition_id = $1 ORDER BY calculated_at DESC LIMIT 1")).
			WithArgs(groupID).WillReturnRows(logRows)

		memberRows := sqlmock.NewRows([]string{"processed_entity_instance_id"}).
			AddRow("uuid-member-1").AddRow("uuid-member-2")
		mock.ExpectQuery(regexp.QuoteMeta("SELECT processed_entity_instance_id FROM group_memberships WHERE group_definition_id = $1")).
			WithArgs(groupID).WillReturnRows(memberRows)
			
		ids, calcAt, err := service.GetGroupResults(groupID)
		require.NoError(t, err)
		assert.Equal(t, []string{"uuid-member-1", "uuid-member-2"}, ids)
		assert.WithinDuration(t, now, calcAt, time.Second) // Check timestamp equality within a second
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("Log Status Not COMPLETED", func(t *testing.T) {
		logRows := sqlmock.NewRows([]string{"calculated_at", "status", "member_count"}).
			AddRow(now, "FAILED", 0)
		mock.ExpectQuery("SELECT .* FROM group_calculation_logs").WithArgs(groupID).WillReturnRows(logRows)
		
		// No query to group_memberships should be made
		
		ids, calcAt, err := service.GetGroupResults(groupID)
		require.Error(t, err)
		assert.Empty(t, ids)
		assert.WithinDuration(t, now, calcAt, time.Second) // calcAt from log is still returned
		assert.Contains(t, err.Error(), "last calculation for group groupTestGetResults was not successful (status: FAILED)")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("No Log Entry Found", func(t *testing.T) {
		logRows := sqlmock.NewRows([]string{"calculated_at", "status", "member_count"}) // Empty rows
		mock.ExpectQuery("SELECT .* FROM group_calculation_logs").WithArgs(groupID).WillReturnRows(logRows)
		
		ids, calcAt, err := service.GetGroupResults(groupID)
		require.NoError(t, err) // Current implementation returns no error, just empty results
		assert.Empty(t, ids)
		assert.True(t, calcAt.IsZero())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
