package metadata

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresStore implements the Store interface using a PostgreSQL database.
type PostgresStore struct {
	DB *sql.DB
}

// NewPostgresStore creates a new PostgresStore, connects to the database,
// and ensures the schema is initialized.
func NewPostgresStore(dataSourceName string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PostgresStore{DB: db}
	if err = store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL and initialized schema.")
	return store, nil
}

// initSchema creates the necessary tables if they don't exist.
func (s *PostgresStore) initSchema() error {
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for schema initialization: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	schemaStatements := []string{
		`CREATE TABLE IF NOT EXISTS entity_definitions (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entity_definitions_name ON entity_definitions(name)`,

		`CREATE TABLE IF NOT EXISTS attribute_definitions (
			id TEXT PRIMARY KEY,
			entity_id TEXT NOT NULL REFERENCES entity_definitions(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			data_type VARCHAR(100) NOT NULL,
			description TEXT,
			is_filterable BOOLEAN DEFAULT FALSE,
			is_pii BOOLEAN DEFAULT FALSE,
			is_indexed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (entity_id, name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attribute_definitions_entity_id ON attribute_definitions(entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attribute_definitions_name ON attribute_definitions(name)`,

		`CREATE TABLE IF NOT EXISTS data_source_configs (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(100) NOT NULL,
			connection_details TEXT,
			entity_id TEXT REFERENCES entity_definitions(id) ON DELETE SET NULL, -- Can be null if not directly tied to one entity
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_data_source_configs_name ON data_source_configs(name)`,
		`CREATE INDEX IF NOT EXISTS idx_data_source_configs_entity_id ON data_source_configs(entity_id)`,
		
		`CREATE TABLE IF NOT EXISTS data_source_field_mappings (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL REFERENCES data_source_configs(id) ON DELETE CASCADE,
			source_field_name VARCHAR(255) NOT NULL,
			entity_id TEXT NOT NULL REFERENCES entity_definitions(id) ON DELETE CASCADE,
			attribute_id TEXT NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
			transformation_rule TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (source_id, source_field_name, attribute_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_data_source_field_mappings_source_id ON data_source_field_mappings(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_data_source_field_mappings_attribute_id ON data_source_field_mappings(attribute_id)`,

		`CREATE TABLE IF NOT EXISTS group_definitions (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			entity_id TEXT NOT NULL REFERENCES entity_definitions(id) ON DELETE CASCADE,
			rules_json TEXT,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_group_definitions_name ON group_definitions(name)`,
		`CREATE INDEX IF NOT EXISTS idx_group_definitions_entity_id ON group_definitions(entity_id)`,

		`CREATE TABLE IF NOT EXISTS workflow_definitions (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			trigger_type VARCHAR(100),
			trigger_config TEXT,
			action_sequence_json TEXT,
			is_enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_definitions_name ON workflow_definitions(name)`,

		`CREATE TABLE IF NOT EXISTS action_templates (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			action_type VARCHAR(100),
			template_content TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_action_templates_name ON action_templates(name)`,

		`CREATE TABLE IF NOT EXISTS schedule_definitions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			cron_expression TEXT NOT NULL,
			task_type TEXT NOT NULL,
			task_parameters JSONB NOT NULL,
			is_enabled BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sd_name ON schedule_definitions(name)`,
		`CREATE INDEX IF NOT EXISTS idx_sd_task_type ON schedule_definitions(task_type)`,
		`CREATE INDEX IF NOT EXISTS idx_sd_is_enabled ON schedule_definitions(is_enabled)`,

		`CREATE TABLE IF NOT EXISTS entity_relationship_definitions (
			id TEXT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			source_entity_id TEXT NOT NULL REFERENCES entity_definitions(id) ON DELETE CASCADE,
			source_attribute_id TEXT NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
			target_entity_id TEXT NOT NULL REFERENCES entity_definitions(id) ON DELETE CASCADE,
			target_attribute_id TEXT NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
			relationship_type VARCHAR(50) NOT NULL, -- ONE_TO_ONE, ONE_TO_MANY, MANY_TO_ONE
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (name, source_entity_id, target_entity_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_er_source_entity_id ON entity_relationship_definitions(source_entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_target_entity_id ON entity_relationship_definitions(target_entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_source_attribute_id ON entity_relationship_definitions(source_attribute_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_target_attribute_id ON entity_relationship_definitions(target_attribute_id)`,
		`CREATE INDEX IF NOT EXISTS idx_er_relationship_type ON entity_relationship_definitions(relationship_type)`,
	}

	for _, stmt := range schemaStatements {
		_, err := tx.Exec(stmt)
		if err != nil {
			return fmt.Errorf("failed to execute schema statement: %s\nError: %w", stmt, err)
		}
	}

	return tx.Commit()
}

// --- EntityDefinition Methods ---

func (s *PostgresStore) CreateEntity(name, description string) (EntityDefinition, error) {
	now := time.Now().UTC()
	entity := EntityDefinition{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	query := `INSERT INTO entity_definitions (id, name, description, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5)`
	_, err := s.DB.Exec(query, entity.ID, entity.Name, entity.Description, entity.CreatedAt, entity.UpdatedAt)
	if err != nil {
		return EntityDefinition{}, fmt.Errorf("CreateEntity failed: %w", err)
	}
	return entity, nil
}

func (s *PostgresStore) GetEntity(id string) (EntityDefinition, error) {
	var entity EntityDefinition
	query := `SELECT id, name, description, created_at, updated_at FROM entity_definitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&entity.ID, &entity.Name, &entity.Description, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EntityDefinition{}, sql.ErrNoRows
		}
		return EntityDefinition{}, fmt.Errorf("GetEntity failed: %w", err)
	}
	return entity, nil
}

func (s *PostgresStore) ListEntities(params ListParams) ([]EntityDefinition, int64, error) {
	var entities []EntityDefinition
	var totalCount int64

	// Base query for counting
	countQueryStr := "SELECT COUNT(*) FROM entity_definitions"
	// Base query for selecting data
	selectQueryStr := "SELECT id, name, description, created_at, updated_at FROM entity_definitions"

	// args will hold query parameters
	var args []interface{}
	var countArgs []interface{}

	// TODO: Implement filtering based on params.Filters if any
	// Example: if nameFilter, ok := params.Filters["name"].(string); ok && nameFilter != "" {
	//    countQueryStr += " WHERE name ILIKE $1"
	//    selectQueryStr += " WHERE name ILIKE $1"
	//    countArgs = append(countArgs, "%"+nameFilter+"%")
	//    args = append(args, "%"+nameFilter+"%")
	// }


	// Get total count
	err := s.DB.QueryRow(countQueryStr, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListEntities count query failed: %w", err)
	}

	if totalCount == 0 {
		return []EntityDefinition{}, 0, nil
	}

	// Add ordering, limit, and offset for selecting data
	selectQueryStr += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(selectQueryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListEntities query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entity EntityDefinition
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.Description, &entity.CreatedAt, &entity.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("ListEntities row scan failed: %w", err)
		}
		entities = append(entities, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListEntities rows iteration error: %w", err)
	}
	return entities, totalCount, nil
}

func (s *PostgresStore) UpdateEntity(id, name, description string) (EntityDefinition, error) {
	now := time.Now().UTC()
	query := `UPDATE entity_definitions SET name = $1, description = $2, updated_at = $3 WHERE id = $4
              RETURNING id, name, description, created_at, updated_at`
	var entity EntityDefinition
	err := s.DB.QueryRow(query, name, description, now, id).Scan(&entity.ID, &entity.Name, &entity.Description, &entity.CreatedAt, &entity.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EntityDefinition{}, sql.ErrNoRows // Or a custom "not found" error
		}
		return EntityDefinition{}, fmt.Errorf("UpdateEntity failed: %w", err)
	}
	return entity, nil
}

func (s *PostgresStore) DeleteEntity(id string) error {
	query := `DELETE FROM entity_definitions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteEntity failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteEntity failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows // Or a custom "not found" error
	}
	return nil
}

// --- EntityRelationshipDefinition Methods ---

func (s *PostgresStore) CreateEntityRelationship(def EntityRelationshipDefinition) (EntityRelationshipDefinition, error) {
	now := time.Now().UTC()
	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	def.CreatedAt = now
	def.UpdatedAt = now

	query := `INSERT INTO entity_relationship_definitions 
              (id, name, description, source_entity_id, source_attribute_id, target_entity_id, target_attribute_id, relationship_type, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.DB.Exec(query, def.ID, def.Name, def.Description, def.SourceEntityID, def.SourceAttributeID, def.TargetEntityID, def.TargetAttributeID, string(def.RelationshipType), def.CreatedAt, def.UpdatedAt)
	if err != nil {
		return EntityRelationshipDefinition{}, fmt.Errorf("CreateEntityRelationship failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) GetEntityRelationship(id string) (EntityRelationshipDefinition, error) {
	var def EntityRelationshipDefinition
	query := `SELECT id, name, description, source_entity_id, source_attribute_id, target_entity_id, target_attribute_id, relationship_type, created_at, updated_at 
              FROM entity_relationship_definitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(
		&def.ID, &def.Name, &def.Description, &def.SourceEntityID, &def.SourceAttributeID, &def.TargetEntityID, &def.TargetAttributeID, &def.RelationshipType, &def.CreatedAt, &def.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EntityRelationshipDefinition{}, sql.ErrNoRows
		}
		return EntityRelationshipDefinition{}, fmt.Errorf("GetEntityRelationship failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) GetEntityRelationshipsBySourceEntity(sourceEntityID string) ([]EntityRelationshipDefinition, error) {
	var defs []EntityRelationshipDefinition
	query := `SELECT id, name, description, source_entity_id, source_attribute_id, target_entity_id, target_attribute_id, relationship_type, created_at, updated_at 
              FROM entity_relationship_definitions WHERE source_entity_id = $1 ORDER BY name`
	rows, err := s.DB.Query(query, sourceEntityID)
	if err != nil {
		return nil, fmt.Errorf("GetEntityRelationshipsBySourceEntity failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var def EntityRelationshipDefinition
		if err := rows.Scan(
			&def.ID, &def.Name, &def.Description, &def.SourceEntityID, &def.SourceAttributeID, &def.TargetEntityID, &def.TargetAttributeID, &def.RelationshipType, &def.CreatedAt, &def.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("GetEntityRelationshipsBySourceEntity row scan failed: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetEntityRelationshipsBySourceEntity rows iteration error: %w", err)
	}
	return defs, nil
}

func (s *PostgresStore) ListEntityRelationships(params ListParams) ([]EntityRelationshipDefinition, int64, error) {
	var defs []EntityRelationshipDefinition
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM entity_relationship_definitions"
	baseSelectQuery := `SELECT id, name, description, source_entity_id, source_attribute_id, 
						target_entity_id, target_attribute_id, relationship_type, created_at, updated_at 
						FROM entity_relationship_definitions`

	var whereClauses []string
	var args []interface{}
	var countArgs []interface{}
	argCounter := 1

	if sourceEntityID, ok := params.Filters["source_entity_id"].(string); ok && sourceEntityID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_entity_id = $%d", argCounter))
		args = append(args, sourceEntityID)
		countArgs = append(countArgs, sourceEntityID)
		argCounter++
	}

	// TODO: Add other filters here as needed, incrementing argCounter

	if len(whereClauses) > 0 {
		whereStr := " WHERE " + strings.Join(whereClauses, " AND ")
		baseCountQuery += whereStr
		baseSelectQuery += whereStr
	}

	err := s.DB.QueryRow(baseCountQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListEntityRelationships count query failed: %w", err)
	}

	if totalCount == 0 {
		return []EntityRelationshipDefinition{}, 0, nil
	}

	baseSelectQuery += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListEntityRelationships query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var def EntityRelationshipDefinition
		if err := rows.Scan(
			&def.ID, &def.Name, &def.Description, &def.SourceEntityID, &def.SourceAttributeID,
			&def.TargetEntityID, &def.TargetAttributeID, &def.RelationshipType, &def.CreatedAt, &def.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("ListEntityRelationships row scan failed: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListEntityRelationships rows iteration error: %w", err)
	}
	return defs, totalCount, nil
}


func (s *PostgresStore) UpdateEntityRelationship(id string, def EntityRelationshipDefinition) (EntityRelationshipDefinition, error) {
	now := time.Now().UTC()
	def.UpdatedAt = now
	def.ID = id // Ensure ID is the one from path param

	query := `UPDATE entity_relationship_definitions 
              SET name = $1, description = $2, source_entity_id = $3, source_attribute_id = $4, target_entity_id = $5, target_attribute_id = $6, relationship_type = $7, updated_at = $8 
              WHERE id = $9
              RETURNING id, name, description, source_entity_id, source_attribute_id, target_entity_id, target_attribute_id, relationship_type, created_at, updated_at`
	var updatedDef EntityRelationshipDefinition
	err := s.DB.QueryRow(query, def.Name, def.Description, def.SourceEntityID, def.SourceAttributeID, def.TargetEntityID, def.TargetAttributeID, string(def.RelationshipType), def.UpdatedAt, def.ID).Scan(
		&updatedDef.ID, &updatedDef.Name, &updatedDef.Description, &updatedDef.SourceEntityID, &updatedDef.SourceAttributeID, &updatedDef.TargetEntityID, &updatedDef.TargetAttributeID, &updatedDef.RelationshipType, &updatedDef.CreatedAt, &updatedDef.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EntityRelationshipDefinition{}, sql.ErrNoRows
		}
		return EntityRelationshipDefinition{}, fmt.Errorf("UpdateEntityRelationship failed: %w", err)
	}
	return updatedDef, nil
}

func (s *PostgresStore) DeleteEntityRelationship(id string) error {
	query := `DELETE FROM entity_relationship_definitions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteEntityRelationship failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteEntityRelationship failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}


// --- ScheduleDefinition Methods ---

func (s *PostgresStore) CreateScheduleDefinition(def ScheduleDefinition) (ScheduleDefinition, error) {
	now := time.Now().UTC()
	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	def.CreatedAt = now
	def.UpdatedAt = now

	query := `INSERT INTO schedule_definitions 
              (id, name, description, cron_expression, task_type, task_parameters, is_enabled, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.DB.Exec(query, def.ID, def.Name, def.Description, def.CronExpression, def.TaskType, def.TaskParameters, def.IsEnabled, def.CreatedAt, def.UpdatedAt)
	if err != nil {
		return ScheduleDefinition{}, fmt.Errorf("CreateScheduleDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) GetScheduleDefinition(id string) (ScheduleDefinition, error) {
	var def ScheduleDefinition
	query := `SELECT id, name, description, cron_expression, task_type, task_parameters, is_enabled, created_at, updated_at 
              FROM schedule_definitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(
		&def.ID, &def.Name, &def.Description, &def.CronExpression, &def.TaskType, &def.TaskParameters, &def.IsEnabled, &def.CreatedAt, &def.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScheduleDefinition{}, sql.ErrNoRows
		}
		return ScheduleDefinition{}, fmt.Errorf("GetScheduleDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) ListScheduleDefinitions(params ListParams) ([]ScheduleDefinition, int64, error) {
	var defs []ScheduleDefinition
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM schedule_definitions"
	baseSelectQuery := `SELECT id, name, description, cron_expression, task_type, 
						task_parameters, is_enabled, created_at, updated_at 
						FROM schedule_definitions`
	
	var args []interface{}
	// var countArgs []interface{} // If filters were added

	// TODO: Implement filtering based on params.Filters if any
	// Add WHERE clauses to baseCountQuery and baseSelectQuery, and populate countArgs/args

	err := s.DB.QueryRow(baseCountQuery /*, countArgs... */).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListScheduleDefinitions count query failed: %w", err)
	}

	if totalCount == 0 {
		return []ScheduleDefinition{}, 0, nil
	}
	
	baseSelectQuery += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListScheduleDefinitions query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var def ScheduleDefinition
		if err := rows.Scan(
			&def.ID, &def.Name, &def.Description, &def.CronExpression, &def.TaskType, 
			&def.TaskParameters, &def.IsEnabled, &def.CreatedAt, &def.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("ListScheduleDefinitions row scan failed: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListScheduleDefinitions rows iteration error: %w", err)
	}
	return defs, totalCount, nil
}

func (s *PostgresStore) UpdateScheduleDefinition(id string, def ScheduleDefinition) (ScheduleDefinition, error) {
	now := time.Now().UTC()
	def.UpdatedAt = now
	def.ID = id // Ensure ID is the one from path param, not from payload if they differ

	query := `UPDATE schedule_definitions 
              SET name = $1, description = $2, cron_expression = $3, task_type = $4, task_parameters = $5, is_enabled = $6, updated_at = $7 
              WHERE id = $8
              RETURNING id, name, description, cron_expression, task_type, task_parameters, is_enabled, created_at, updated_at`
	var updatedDef ScheduleDefinition
	err := s.DB.QueryRow(query, def.Name, def.Description, def.CronExpression, def.TaskType, def.TaskParameters, def.IsEnabled, def.UpdatedAt, def.ID).Scan(
		&updatedDef.ID, &updatedDef.Name, &updatedDef.Description, &updatedDef.CronExpression, &updatedDef.TaskType, &updatedDef.TaskParameters, &updatedDef.IsEnabled, &updatedDef.CreatedAt, &updatedDef.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScheduleDefinition{}, sql.ErrNoRows
		}
		return ScheduleDefinition{}, fmt.Errorf("UpdateScheduleDefinition failed: %w", err)
	}
	return updatedDef, nil
}

func (s *PostgresStore) DeleteScheduleDefinition(id string) error {
	query := `DELETE FROM schedule_definitions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteScheduleDefinition failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteScheduleDefinition failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- AttributeDefinition Methods ---

func (s *PostgresStore) CreateAttribute(entityID, name, dataType, description string, isFilterable bool, isPii bool, isIndexed bool) (AttributeDefinition, error) {
	now := time.Now().UTC()
	attr := AttributeDefinition{
		ID:           uuid.NewString(),
		EntityID:     entityID,
		Name:         name,
		DataType:     dataType,
		Description:  description,
		IsFilterable: isFilterable,
		IsPii:        isPii,
		IsIndexed:    isIndexed,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	query := `INSERT INTO attribute_definitions 
              (id, entity_id, name, data_type, description, is_filterable, is_pii, is_indexed, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.DB.Exec(query, attr.ID, attr.EntityID, attr.Name, attr.DataType, attr.Description, attr.IsFilterable, attr.IsPii, attr.IsIndexed, attr.CreatedAt, attr.UpdatedAt)
	if err != nil {
		return AttributeDefinition{}, fmt.Errorf("CreateAttribute failed: %w", err)
	}
	return attr, nil
}

func (s *PostgresStore) GetAttribute(entityID, attributeID string) (AttributeDefinition, error) {
	var attr AttributeDefinition
	query := `SELECT id, entity_id, name, data_type, description, is_filterable, is_pii, is_indexed, created_at, updated_at 
              FROM attribute_definitions WHERE entity_id = $1 AND id = $2`
	err := s.DB.QueryRow(query, entityID, attributeID).Scan(&attr.ID, &attr.EntityID, &attr.Name, &attr.DataType, &attr.Description, &attr.IsFilterable, &attr.IsPii, &attr.IsIndexed, &attr.CreatedAt, &attr.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AttributeDefinition{}, sql.ErrNoRows
		}
		return AttributeDefinition{}, fmt.Errorf("GetAttribute failed: %w", err)
	}
	return attr, nil
}

func (s *PostgresStore) ListAttributes(entityID string, params ListParams) ([]AttributeDefinition, int64, error) {
	var attrs []AttributeDefinition
	var totalCount int64

	// Base query for counting
	countQueryStr := "SELECT COUNT(*) FROM attribute_definitions WHERE entity_id = $1"
	// Base query for selecting data
	selectQueryStr := `SELECT id, entity_id, name, data_type, description, is_filterable, 
						 is_pii, is_indexed, created_at, updated_at 
						 FROM attribute_definitions WHERE entity_id = $1`

	var args []interface{}
	var countArgs []interface{}{entityID}
	argCounter := 2 // Start after entity_id

	// TODO: Implement filtering based on params.Filters if any
	// Example: if nameFilter, ok := params.Filters["name"].(string); ok && nameFilter != "" {
	//    countQueryStr += fmt.Sprintf(" AND name ILIKE $%d", argCounter)
	//    selectQueryStr += fmt.Sprintf(" AND name ILIKE $%d", argCounter)
	//    countArgs = append(countArgs, "%"+nameFilter+"%")
	//    args = append(args, "%"+nameFilter+"%")
	//    argCounter++
	// }
	
	err := s.DB.QueryRow(countQueryStr, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAttributes count query for entityID %s failed: %w", entityID, err)
	}

	if totalCount == 0 {
		return []AttributeDefinition{}, 0, nil
	}
	
	args = append(args, entityID) // First arg for select is entity_id
	selectQueryStr += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	args = append(args, params.GetLimit(), params.GetOffset())


	rows, err := s.DB.Query(selectQueryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAttributes query for entityID %s failed: %w", entityID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var attr AttributeDefinition
		if err := rows.Scan(&attr.ID, &attr.EntityID, &attr.Name, &attr.DataType, &attr.Description, 
			&attr.IsFilterable, &attr.IsPii, &attr.IsIndexed, &attr.CreatedAt, &attr.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("ListAttributes row scan for entityID %s failed: %w", entityID, err)
		}
		attrs = append(attrs, attr)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListAttributes rows iteration error for entityID %s: %w", entityID, err)
	}
	return attrs, totalCount, nil
}

func (s *PostgresStore) UpdateAttribute(entityID, attributeID, name, dataType, description string, isFilterable bool, isPii bool, isIndexed bool) (AttributeDefinition, error) {
	now := time.Now().UTC()
	query := `UPDATE attribute_definitions 
              SET name = $1, data_type = $2, description = $3, is_filterable = $4, is_pii = $5, is_indexed = $6, updated_at = $7 
              WHERE entity_id = $8 AND id = $9
              RETURNING id, entity_id, name, data_type, description, is_filterable, is_pii, is_indexed, created_at, updated_at`
	var attr AttributeDefinition
	err := s.DB.QueryRow(query, name, dataType, description, isFilterable, isPii, isIndexed, now, entityID, attributeID).Scan(
		&attr.ID, &attr.EntityID, &attr.Name, &attr.DataType, &attr.Description, &attr.IsFilterable, &attr.IsPii, &attr.IsIndexed, &attr.CreatedAt, &attr.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AttributeDefinition{}, sql.ErrNoRows
		}
		return AttributeDefinition{}, fmt.Errorf("UpdateAttribute failed: %w", err)
	}
	return attr, nil
}

func (s *PostgresStore) DeleteAttribute(entityID, attributeID string) error {
	query := `DELETE FROM attribute_definitions WHERE entity_id = $1 AND id = $2`
	result, err := s.DB.Exec(query, entityID, attributeID)
	if err != nil {
		return fmt.Errorf("DeleteAttribute failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteAttribute failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- DataSourceConfig Methods ---

func (s *PostgresStore) CreateDataSource(config DataSourceConfig) (DataSourceConfig, error) {
	now := time.Now().UTC()
	if config.ID == "" {
		config.ID = uuid.NewString()
	}
	config.CreatedAt = now
	config.UpdatedAt = now

	query := `INSERT INTO data_source_configs (id, name, type, connection_details, entity_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.DB.Exec(query, config.ID, config.Name, config.Type, config.ConnectionDetails, sql.NullString{String: config.EntityID, Valid: config.EntityID != ""}, config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return DataSourceConfig{}, fmt.Errorf("CreateDataSource failed: %w", err)
	}
	return config, nil
}

func (s *PostgresStore) GetDataSources(params ListParams) ([]DataSourceConfig, int64, error) {
	var configs []DataSourceConfig
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM data_source_configs"
	baseSelectQuery := `SELECT id, name, type, connection_details, entity_id, 
						created_at, updated_at FROM data_source_configs`
	
	var args []interface{}
	// var countArgs []interface{} // If filters were added

	// TODO: Implement filtering based on params.Filters if any
	
	err := s.DB.QueryRow(baseCountQuery /*, countArgs... */).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("GetDataSources count query failed: %w", err)
	}

	if totalCount == 0 {
		return []DataSourceConfig{}, 0, nil
	}

	baseSelectQuery += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())
	
	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("GetDataSources query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var config DataSourceConfig
		var entityID sql.NullString
		if err := rows.Scan(&config.ID, &config.Name, &config.Type, &config.ConnectionDetails, 
			&entityID, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("GetDataSources row scan failed: %w", err)
		}
		if entityID.Valid {
			config.EntityID = entityID.String
		}
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("GetDataSources rows iteration error: %w", err)
	}
	return configs, totalCount, nil
}

func (s *PostgresStore) GetDataSource(id string) (DataSourceConfig, error) {
	var config DataSourceConfig
	var entityID sql.NullString
	query := `SELECT id, name, type, connection_details, entity_id, created_at, updated_at FROM data_source_configs WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&config.ID, &config.Name, &config.Type, &config.ConnectionDetails, &entityID, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DataSourceConfig{}, sql.ErrNoRows
		}
		return DataSourceConfig{}, fmt.Errorf("GetDataSource failed: %w", err)
	}
	if entityID.Valid {
		config.EntityID = entityID.String
	}
	return config, nil
}

func (s *PostgresStore) UpdateDataSource(id string, config DataSourceConfig) (DataSourceConfig, error) {
	now := time.Now().UTC()
	config.UpdatedAt = now
	config.ID = id // Ensure ID is the one from path param

	query := `UPDATE data_source_configs 
              SET name = $1, type = $2, connection_details = $3, entity_id = $4, updated_at = $5 
              WHERE id = $6
              RETURNING id, name, type, connection_details, entity_id, created_at, updated_at`
	var updatedConfig DataSourceConfig
	var entityID sql.NullString
	err := s.DB.QueryRow(query, config.Name, config.Type, config.ConnectionDetails, sql.NullString{String: config.EntityID, Valid: config.EntityID != ""}, config.UpdatedAt, config.ID).Scan(
		&updatedConfig.ID, &updatedConfig.Name, &updatedConfig.Type, &updatedConfig.ConnectionDetails, &entityID, &updatedConfig.CreatedAt, &updatedConfig.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DataSourceConfig{}, sql.ErrNoRows
		}
		return DataSourceConfig{}, fmt.Errorf("UpdateDataSource failed: %w", err)
	}
	if entityID.Valid {
		updatedConfig.EntityID = entityID.String
	}
	return updatedConfig, nil
}

func (s *PostgresStore) DeleteDataSource(id string) error {
	query := `DELETE FROM data_source_configs WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteDataSource failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteDataSource failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- DataSourceFieldMapping Methods ---

func (s *PostgresStore) CreateFieldMapping(mapping DataSourceFieldMapping) (DataSourceFieldMapping, error) {
	now := time.Now().UTC()
	if mapping.ID == "" {
		mapping.ID = uuid.NewString()
	}
	mapping.CreatedAt = now
	mapping.UpdatedAt = now

	query := `INSERT INTO data_source_field_mappings 
              (id, source_id, source_field_name, entity_id, attribute_id, transformation_rule, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := s.DB.Exec(query, mapping.ID, mapping.SourceID, mapping.SourceFieldName, mapping.EntityID, mapping.AttributeID, mapping.TransformationRule, mapping.CreatedAt, mapping.UpdatedAt)
	if err != nil {
		return DataSourceFieldMapping{}, fmt.Errorf("CreateFieldMapping failed: %w", err)
	}
	return mapping, nil
}

func (s *PostgresStore) GetFieldMappings(sourceID string, params ListParams) ([]DataSourceFieldMapping, int64, error) {
	var mappings []DataSourceFieldMapping
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM data_source_field_mappings WHERE source_id = $1"
	baseSelectQuery := `SELECT id, source_id, source_field_name, entity_id, attribute_id, 
						transformation_rule, created_at, updated_at 
						FROM data_source_field_mappings WHERE source_id = $1`

	var args []interface{}
	var countArgs []interface{}{sourceID}
	argCounter := 2 // Start after source_id for select query, count query already has source_id

	// TODO: Implement filtering based on params.Filters if any
	
	err := s.DB.QueryRow(baseCountQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("GetFieldMappings count for sourceID %s failed: %w", sourceID, err)
	}

	if totalCount == 0 {
		return []DataSourceFieldMapping{}, 0, nil
	}
	
	args = append(args, sourceID) // First arg for select is source_id
	baseSelectQuery += fmt.Sprintf(" ORDER BY source_field_name LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("GetFieldMappings query for sourceID %s failed: %w", sourceID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var m DataSourceFieldMapping
		if err := rows.Scan(&m.ID, &m.SourceID, &m.SourceFieldName, &m.EntityID, 
			&m.AttributeID, &m.TransformationRule, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("GetFieldMappings row scan for sourceID %s failed: %w", sourceID, err)
		}
		mappings = append(mappings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("GetFieldMappings rows iteration for sourceID %s error: %w", sourceID, err)
	}
	return mappings, totalCount, nil
}

func (s *PostgresStore) GetFieldMapping(sourceID, mappingID string) (DataSourceFieldMapping, error) {
	var m DataSourceFieldMapping
	query := `SELECT id, source_id, source_field_name, entity_id, attribute_id, transformation_rule, created_at, updated_at 
              FROM data_source_field_mappings WHERE source_id = $1 AND id = $2`
	err := s.DB.QueryRow(query, sourceID, mappingID).Scan(&m.ID, &m.SourceID, &m.SourceFieldName, &m.EntityID, &m.AttributeID, &m.TransformationRule, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DataSourceFieldMapping{}, sql.ErrNoRows
		}
		return DataSourceFieldMapping{}, fmt.Errorf("GetFieldMapping failed: %w", err)
	}
	return m, nil
}

func (s *PostgresStore) UpdateFieldMapping(sourceID, mappingID string, mapping DataSourceFieldMapping) (DataSourceFieldMapping, error) {
	now := time.Now().UTC()
	mapping.UpdatedAt = now
	mapping.ID = mappingID
	mapping.SourceID = sourceID

	query := `UPDATE data_source_field_mappings 
              SET source_field_name = $1, entity_id = $2, attribute_id = $3, transformation_rule = $4, updated_at = $5 
              WHERE source_id = $6 AND id = $7
              RETURNING id, source_id, source_field_name, entity_id, attribute_id, transformation_rule, created_at, updated_at`
	var updatedMapping DataSourceFieldMapping
	err := s.DB.QueryRow(query, mapping.SourceFieldName, mapping.EntityID, mapping.AttributeID, mapping.TransformationRule, mapping.UpdatedAt, mapping.SourceID, mapping.ID).Scan(
		&updatedMapping.ID, &updatedMapping.SourceID, &updatedMapping.SourceFieldName, &updatedMapping.EntityID, &updatedMapping.AttributeID, &updatedMapping.TransformationRule, &updatedMapping.CreatedAt, &updatedMapping.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DataSourceFieldMapping{}, sql.ErrNoRows
		}
		return DataSourceFieldMapping{}, fmt.Errorf("UpdateFieldMapping failed: %w", err)
	}
	return updatedMapping, nil
}

func (s *PostgresStore) DeleteFieldMapping(sourceID, mappingID string) error {
	query := `DELETE FROM data_source_field_mappings WHERE source_id = $1 AND id = $2`
	result, err := s.DB.Exec(query, sourceID, mappingID)
	if err != nil {
		return fmt.Errorf("DeleteFieldMapping failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteFieldMapping failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- GroupDefinition Methods ---

func (s *PostgresStore) CreateGroupDefinition(def GroupDefinition) (GroupDefinition, error) {
	now := time.Now().UTC()
	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	def.CreatedAt = now
	def.UpdatedAt = now

	query := `INSERT INTO group_definitions (id, name, entity_id, rules_json, description, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.DB.Exec(query, def.ID, def.Name, def.EntityID, def.RulesJSON, def.Description, def.CreatedAt, def.UpdatedAt)
	if err != nil {
		return GroupDefinition{}, fmt.Errorf("CreateGroupDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) GetGroupDefinition(id string) (GroupDefinition, error) {
	var def GroupDefinition
	query := `SELECT id, name, entity_id, rules_json, description, created_at, updated_at FROM group_definitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&def.ID, &def.Name, &def.EntityID, &def.RulesJSON, &def.Description, &def.CreatedAt, &def.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GroupDefinition{}, sql.ErrNoRows
		}
		return GroupDefinition{}, fmt.Errorf("GetGroupDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) ListGroupDefinitions(params ListParams) ([]GroupDefinition, int64, error) {
	var defs []GroupDefinition
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM group_definitions"
	baseSelectQuery := `SELECT id, name, entity_id, rules_json, description, 
						created_at, updated_at FROM group_definitions`
	
	var args []interface{}
	// var countArgs []interface{} // If filters were added

	// TODO: Implement filtering based on params.Filters if any

	err := s.DB.QueryRow(baseCountQuery /*, countArgs... */).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListGroupDefinitions count query failed: %w", err)
	}

	if totalCount == 0 {
		return []GroupDefinition{}, 0, nil
	}

	baseSelectQuery += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())
	
	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListGroupDefinitions query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var def GroupDefinition
		if err := rows.Scan(&def.ID, &def.Name, &def.EntityID, &def.RulesJSON, 
			&def.Description, &def.CreatedAt, &def.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("ListGroupDefinitions row scan failed: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListGroupDefinitions rows iteration error: %w", err)
	}
	return defs, totalCount, nil
}

func (s *PostgresStore) UpdateGroupDefinition(id string, def GroupDefinition) (GroupDefinition, error) {
	now := time.Now().UTC()
	def.UpdatedAt = now
	def.ID = id

	query := `UPDATE group_definitions 
              SET name = $1, entity_id = $2, rules_json = $3, description = $4, updated_at = $5 
              WHERE id = $6
              RETURNING id, name, entity_id, rules_json, description, created_at, updated_at`
	var updatedDef GroupDefinition
	err := s.DB.QueryRow(query, def.Name, def.EntityID, def.RulesJSON, def.Description, def.UpdatedAt, def.ID).Scan(
		&updatedDef.ID, &updatedDef.Name, &updatedDef.EntityID, &updatedDef.RulesJSON, &updatedDef.Description, &updatedDef.CreatedAt, &updatedDef.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GroupDefinition{}, sql.ErrNoRows
		}
		return GroupDefinition{}, fmt.Errorf("UpdateGroupDefinition failed: %w", err)
	}
	return updatedDef, nil
}

func (s *PostgresStore) DeleteGroupDefinition(id string) error {
	query := `DELETE FROM group_definitions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteGroupDefinition failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteGroupDefinition failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- WorkflowDefinition Methods ---

func (s *PostgresStore) CreateWorkflowDefinition(def WorkflowDefinition) (WorkflowDefinition, error) {
	now := time.Now().UTC()
	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	def.CreatedAt = now
	def.UpdatedAt = now

	query := `INSERT INTO workflow_definitions 
              (id, name, description, trigger_type, trigger_config, action_sequence_json, is_enabled, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.DB.Exec(query, def.ID, def.Name, def.Description, def.TriggerType, def.TriggerConfig, def.ActionSequenceJSON, def.IsEnabled, def.CreatedAt, def.UpdatedAt)
	if err != nil {
		return WorkflowDefinition{}, fmt.Errorf("CreateWorkflowDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) GetWorkflowDefinition(id string) (WorkflowDefinition, error) {
	var def WorkflowDefinition
	query := `SELECT id, name, description, trigger_type, trigger_config, action_sequence_json, is_enabled, created_at, updated_at 
              FROM workflow_definitions WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(
		&def.ID, &def.Name, &def.Description, &def.TriggerType, &def.TriggerConfig, &def.ActionSequenceJSON, &def.IsEnabled, &def.CreatedAt, &def.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkflowDefinition{}, sql.ErrNoRows
		}
		return WorkflowDefinition{}, fmt.Errorf("GetWorkflowDefinition failed: %w", err)
	}
	return def, nil
}

func (s *PostgresStore) ListWorkflowDefinitions(params ListParams) ([]WorkflowDefinition, int64, error) {
	var defs []WorkflowDefinition
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM workflow_definitions"
	baseSelectQuery := `SELECT id, name, description, trigger_type, trigger_config, 
						action_sequence_json, is_enabled, created_at, updated_at 
						FROM workflow_definitions`
	
	var args []interface{}
	// var countArgs []interface{} // If filters were added

	// TODO: Implement filtering based on params.Filters if any

	err := s.DB.QueryRow(baseCountQuery /*, countArgs... */).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListWorkflowDefinitions count query failed: %w", err)
	}

	if totalCount == 0 {
		return []WorkflowDefinition{}, 0, nil
	}
	
	baseSelectQuery += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListWorkflowDefinitions query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var def WorkflowDefinition
		if err := rows.Scan(
			&def.ID, &def.Name, &def.Description, &def.TriggerType, &def.TriggerConfig, 
			&def.ActionSequenceJSON, &def.IsEnabled, &def.CreatedAt, &def.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("ListWorkflowDefinitions row scan failed: %w", err)
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListWorkflowDefinitions rows iteration error: %w", err)
	}
	return defs, totalCount, nil
}

func (s *PostgresStore) UpdateWorkflowDefinition(id string, def WorkflowDefinition) (WorkflowDefinition, error) {
	now := time.Now().UTC()
	def.UpdatedAt = now
	def.ID = id

	query := `UPDATE workflow_definitions 
              SET name = $1, description = $2, trigger_type = $3, trigger_config = $4, action_sequence_json = $5, is_enabled = $6, updated_at = $7 
              WHERE id = $8
              RETURNING id, name, description, trigger_type, trigger_config, action_sequence_json, is_enabled, created_at, updated_at`
	var updatedDef WorkflowDefinition
	err := s.DB.QueryRow(query, def.Name, def.Description, def.TriggerType, def.TriggerConfig, def.ActionSequenceJSON, def.IsEnabled, def.UpdatedAt, def.ID).Scan(
		&updatedDef.ID, &updatedDef.Name, &updatedDef.Description, &updatedDef.TriggerType, &updatedDef.TriggerConfig, &updatedDef.ActionSequenceJSON, &updatedDef.IsEnabled, &updatedDef.CreatedAt, &updatedDef.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorkflowDefinition{}, sql.ErrNoRows
		}
		return WorkflowDefinition{}, fmt.Errorf("UpdateWorkflowDefinition failed: %w", err)
	}
	return updatedDef, nil
}

func (s *PostgresStore) DeleteWorkflowDefinition(id string) error {
	query := `DELETE FROM workflow_definitions WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteWorkflowDefinition failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteWorkflowDefinition failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- ActionTemplate Methods ---

func (s *PostgresStore) CreateActionTemplate(tmpl ActionTemplate) (ActionTemplate, error) {
	now := time.Now().UTC()
	if tmpl.ID == "" {
		tmpl.ID = uuid.NewString()
	}
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now

	query := `INSERT INTO action_templates (id, name, description, action_type, template_content, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.DB.Exec(query, tmpl.ID, tmpl.Name, tmpl.Description, tmpl.ActionType, tmpl.TemplateContent, tmpl.CreatedAt, tmpl.UpdatedAt)
	if err != nil {
		return ActionTemplate{}, fmt.Errorf("CreateActionTemplate failed: %w", err)
	}
	return tmpl, nil
}

func (s *PostgresStore) GetActionTemplate(id string) (ActionTemplate, error) {
	var tmpl ActionTemplate
	query := `SELECT id, name, description, action_type, template_content, created_at, updated_at FROM action_templates WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(
		&tmpl.ID, &tmpl.Name, &tmpl.Description, &tmpl.ActionType, &tmpl.TemplateContent, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActionTemplate{}, sql.ErrNoRows
		}
		return ActionTemplate{}, fmt.Errorf("GetActionTemplate failed: %w", err)
	}
	return tmpl, nil
}

func (s *PostgresStore) ListActionTemplates(params ListParams) ([]ActionTemplate, int64, error) {
	var tmpls []ActionTemplate
	var totalCount int64

	baseCountQuery := "SELECT COUNT(*) FROM action_templates"
	baseSelectQuery := `SELECT id, name, description, action_type, template_content, 
						created_at, updated_at FROM action_templates`
	
	var args []interface{}
	// var countArgs []interface{} // If filters were added

	// TODO: Implement filtering based on params.Filters if any
	
	err := s.DB.QueryRow(baseCountQuery /*, countArgs... */).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("ListActionTemplates count query failed: %w", err)
	}

	if totalCount == 0 {
		return []ActionTemplate{}, 0, nil
	}

	baseSelectQuery += " ORDER BY name LIMIT $1 OFFSET $2"
	args = append(args, params.GetLimit(), params.GetOffset())

	rows, err := s.DB.Query(baseSelectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListActionTemplates query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tmpl ActionTemplate
		if err := rows.Scan(
			&tmpl.ID, &tmpl.Name, &tmpl.Description, &tmpl.ActionType, 
			&tmpl.TemplateContent, &tmpl.CreatedAt, &tmpl.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("ListActionTemplates row scan failed: %w", err)
		}
		tmpls = append(tmpls, tmpl)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListActionTemplates rows iteration error: %w", err)
	}
	return tmpls, totalCount, nil
}

func (s *PostgresStore) UpdateActionTemplate(id string, tmpl ActionTemplate) (ActionTemplate, error) {
	now := time.Now().UTC()
	tmpl.UpdatedAt = now
	tmpl.ID = id

	query := `UPDATE action_templates 
              SET name = $1, description = $2, action_type = $3, template_content = $4, updated_at = $5 
              WHERE id = $6
              RETURNING id, name, description, action_type, template_content, created_at, updated_at`
	var updatedTmpl ActionTemplate
	err := s.DB.QueryRow(query, tmpl.Name, tmpl.Description, tmpl.ActionType, tmpl.TemplateContent, tmpl.UpdatedAt, tmpl.ID).Scan(
		&updatedTmpl.ID, &updatedTmpl.Name, &updatedTmpl.Description, &updatedTmpl.ActionType, &updatedTmpl.TemplateContent, &updatedTmpl.CreatedAt, &updatedTmpl.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActionTemplate{}, sql.ErrNoRows
		}
		return ActionTemplate{}, fmt.Errorf("UpdateActionTemplate failed: %w", err)
	}
	return updatedTmpl, nil
}

func (s *PostgresStore) DeleteActionTemplate(id string) error {
	query := `DELETE FROM action_templates WHERE id = $1`
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("DeleteActionTemplate failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteActionTemplate failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Interface assertion (optional, but good for ensuring correctness)
// var _ Store = (*PostgresStore)(nil)
// The above line would require the Store interface to be defined in this file or package.
// For now, we assume the interface matches the methods implemented.
// The actual `Store` interface is defined in `api.go`, so this check can't be done here directly
// without causing import cycles or refactoring interface definition location.
// It will be implicitly checked at compile time if `PostgresStore` is used where `Store` is expected.

// --- Bulk EntityDefinition Methods ---

// BulkCreateEntities attempts to create multiple entities.
// It processes each entity individually and reports success or failure for each.
func (s *PostgresStore) BulkCreateEntities(entities []EntityCreateData) ([]BulkOperationResultItem, error) {
	results := make([]BulkOperationResultItem, 0, len(entities))

	for _, item := range entities {
		result := BulkOperationResultItem{} // ID will be set upon successful creation

		entity, err := s.CreateEntity(item.Name, item.Description)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			// For create, the input item doesn't have an ID, so we can't set result.ID if creation fails.
			// We could potentially use the Name as a temporary reference if needed, but the spec for BulkOperationResultItem has ID as omitempty.
		} else {
			result.Success = true
			result.ID = entity.ID
			result.Entity = &entity
		}
		results = append(results, result)
	}
	return results, nil // Overall method error is nil unless a catastrophic failure not tied to individual items occurs.
}

// BulkUpdateEntities attempts to update multiple entities.
// It processes each entity individually and reports success or failure for each.
// For partial updates, ensure that EntityUpdateData fields are pointers or use a similar mechanism
// if empty strings are valid values and should not be skipped. Current EntityUpdateData uses omitempty,
// so empty fields are fine for not updating.
func (s *PostgresStore) BulkUpdateEntities(entities []EntityUpdateData) ([]BulkOperationResultItem, error) {
	results := make([]BulkOperationResultItem, 0, len(entities))

	for _, item := range entities {
		result := BulkOperationResultItem{ID: item.ID}

		// Get current entity to ensure it exists and to have its full state if needed.
		// The current UpdateEntity method in store.go doesn't require pre-fetching
		// if we are okay with its behavior of updating specified fields.
		// However, to ensure we only update existing, a GetEntity check is good.
		existingEntity, err := s.GetEntity(item.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				result.Success = false
				result.Error = fmt.Sprintf("entity with ID %s not found", item.ID)
			} else {
				result.Success = false
				result.Error = fmt.Sprintf("failed to retrieve entity %s for update: %v", item.ID, err)
			}
			results = append(results, result)
			continue
		}

		// Use existing values if new ones are not provided (respecting omitempty behavior)
		name := existingEntity.Name
		if item.Name != "" {
			name = item.Name
		}
		description := existingEntity.Description
		if item.Description != "" { // Assuming empty string for description means "do not update" if it was previously set.
			description = item.Description
		}
		
		// The existing UpdateEntity method updates both name and description.
		// If item.Name or item.Description is empty, it will update the field to be empty.
		// This might not be desired for partial updates where empty means "don't change".
		// For true partial update with current UpdateEntity, we'd pass existing values if item fields are empty.
		updatedEntity, err := s.UpdateEntity(item.ID, name, description)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Entity = &updatedEntity
		}
		results = append(results, result)
	}
	return results, nil
}

// BulkDeleteEntities attempts to delete multiple entities by their IDs.
// It processes each ID individually and reports success or failure.
// Deletion is considered successful if the entity was deleted or if it was already not found (idempotency).
func (s *PostgresStore) BulkDeleteEntities(entityIDs []string) ([]BulkOperationResultItem, error) {
	results := make([]BulkOperationResultItem, 0, len(entityIDs))

	for _, id := range entityIDs {
		result := BulkOperationResultItem{ID: id}
		err := s.DeleteEntity(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Considered success for idempotency: entity already deleted or never existed.
				result.Success = true
				result.Error = "entity not found (considered successful as it's already gone)"
			} else {
				result.Success = false
				result.Error = err.Error()
			}
		} else {
			result.Success = true
		}
		results = append(results, result)
	}
	return results, nil
}


func (s *PostgresStore) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}
