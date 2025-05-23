package metadata

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store manages metadata entities, attributes, data sources, and field mappings in memory.
type Store struct {
	entities      map[string]EntityDefinition
	attributes    map[string]map[string]AttributeDefinition // Key: EntityID, Key: AttributeID
	dataSources      map[string]DataSourceConfig
	fieldMappings    map[string]map[string]DataSourceFieldMapping // Key: SourceID, Key: MappingID
	groupDefinitions map[string]GroupDefinition                   // Key: GroupDefinitionID
	mu               sync.RWMutex
}

// NewStore creates and returns a new Store.
func NewStore() *Store {
	return &Store{
		entities:         make(map[string]EntityDefinition),
		attributes:       make(map[string]map[string]AttributeDefinition),
		dataSources:      make(map[string]DataSourceConfig),
		fieldMappings:    make(map[string]map[string]DataSourceFieldMapping),
		groupDefinitions: make(map[string]GroupDefinition),
	}
}

// --- Entity Methods ---

// CreateEntity adds a new entity to the store.
func (s *Store) CreateEntity(name, description string) (EntityDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return EntityDefinition{}, fmt.Errorf("entity name cannot be empty")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	entity := EntityDefinition{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.entities[id] = entity
	s.attributes[id] = make(map[string]AttributeDefinition) // Initialize attributes map for the new entity
	return entity, nil
}

// GetEntity retrieves an entity by its ID.
func (s *Store) GetEntity(id string) (EntityDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entity, ok := s.entities[id]
	return entity, ok
}

// ListEntities retrieves all entities.
func (s *Store) ListEntities() []EntityDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]EntityDefinition, 0, len(s.entities))
	for _, entity := range s.entities {
		list = append(list, entity)
	}
	return list
}

// UpdateEntity updates an existing entity.
func (s *Store) UpdateEntity(id, name, description string) (EntityDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entity, ok := s.entities[id]
	if !ok {
		return EntityDefinition{}, fmt.Errorf("entity with ID %s not found", id)
	}

	if name == "" {
		return EntityDefinition{}, fmt.Errorf("entity name cannot be empty")
	}

	entity.Name = name
	entity.Description = description
	entity.UpdatedAt = time.Now().UTC()
	s.entities[id] = entity
	return entity, nil
}

// DeleteEntity removes an entity and its attributes from the store.
// It also removes any field mappings associated with this entity or its attributes.
func (s *Store) DeleteEntity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[id]; !ok {
		return fmt.Errorf("entity with ID %s not found", id)
	}

	// Get all attributes for the entity to be deleted
	attrsToDelete := make(map[string]struct{})
	if entityAttrs, ok := s.attributes[id]; ok {
		for attrID := range entityAttrs {
			attrsToDelete[attrID] = struct{}{}
		}
	}

	// Remove field mappings associated with the entity or its attributes
	for sourceID, mappings := range s.fieldMappings {
		for mappingID, mapping := range mappings {
			if mapping.EntityID == id { // Mapping directly to the entity (if that's a use case)
				delete(s.fieldMappings[sourceID], mappingID)
				continue
			}
			if _, isAttrToDelete := attrsToDelete[mapping.AttributeID]; isAttrToDelete {
				delete(s.fieldMappings[sourceID], mappingID)
			}
		}
		if len(s.fieldMappings[sourceID]) == 0 {
			delete(s.fieldMappings, sourceID)
		}
	}

	delete(s.entities, id)
	delete(s.attributes, id) // Also delete associated attributes
	return nil
}

// --- GroupDefinition Methods ---

// CreateGroupDefinition adds a new group definition to the store.
func (s *Store) CreateGroupDefinition(def GroupDefinition) (GroupDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if def.Name == "" {
		return GroupDefinition{}, fmt.Errorf("group definition name cannot be empty")
	}
	if def.EntityID == "" {
		return GroupDefinition{}, fmt.Errorf("group definition entity_id cannot be empty")
	}
	if _, ok := s.entities[def.EntityID]; !ok {
		return GroupDefinition{}, fmt.Errorf("entity with ID %s not found", def.EntityID)
	}
	if def.RulesJSON == "" { // Basic check for non-empty rules
		return GroupDefinition{}, fmt.Errorf("group definition rules_json cannot be empty")
	}
	// A more sophisticated JSON validation could be added here if needed.

	id := uuid.New().String()
	now := time.Now().UTC()
	def.ID = id
	def.CreatedAt = now
	def.UpdatedAt = now

	s.groupDefinitions[id] = def
	return def, nil
}

// GetGroupDefinition retrieves a group definition by its ID.
func (s *Store) GetGroupDefinition(id string) (GroupDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groupDef, ok := s.groupDefinitions[id]
	if !ok {
		return GroupDefinition{}, fmt.Errorf("group definition with ID %s not found", id)
	}
	return groupDef, nil
}

// ListGroupDefinitions retrieves all group definitions.
// Future enhancement: Add filtering by entityID if needed.
func (s *Store) ListGroupDefinitions() ([]GroupDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]GroupDefinition, 0, len(s.groupDefinitions))
	for _, gd := range s.groupDefinitions {
		list = append(list, gd)
	}
	return list, nil
}

// UpdateGroupDefinition updates an existing group definition.
func (s *Store) UpdateGroupDefinition(id string, def GroupDefinition) (GroupDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingDef, ok := s.groupDefinitions[id]
	if !ok {
		return GroupDefinition{}, fmt.Errorf("group definition with ID %s not found", id)
	}

	if def.Name == "" {
		return GroupDefinition{}, fmt.Errorf("group definition name cannot be empty")
	}
	// EntityID cannot be changed for a group definition, so we don't check/update it.
	// If changing EntityID is a requirement, ensure the new EntityID exists.
	// For now, we assume EntityID is immutable for a given group.
	if def.RulesJSON == "" {
		return GroupDefinition{}, fmt.Errorf("group definition rules_json cannot be empty")
	}

	existingDef.Name = def.Name
	existingDef.RulesJSON = def.RulesJSON
	existingDef.Description = def.Description
	existingDef.UpdatedAt = time.Now().UTC()
	s.groupDefinitions[id] = existingDef
	return existingDef, nil
}

// DeleteGroupDefinition removes a group definition from the store.
func (s *Store) DeleteGroupDefinition(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groupDefinitions[id]; !ok {
		return fmt.Errorf("group definition with ID %s not found", id)
	}
	delete(s.groupDefinitions, id)
	return nil
}

// --- Attribute Methods ---

// CreateAttribute adds a new attribute to an entity.
func (s *Store) CreateAttribute(entityID, name, dataType, description string, isFilterable bool, isPii bool, isIndexed bool) (AttributeDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[entityID]; !ok {
		return AttributeDefinition{}, fmt.Errorf("entity with ID %s not found", entityID)
	}
	if name == "" {
		return AttributeDefinition{}, fmt.Errorf("attribute name cannot be empty")
	}
	if dataType == "" {
		return AttributeDefinition{}, fmt.Errorf("attribute dataType cannot be empty")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	attr := AttributeDefinition{
		ID:           id,
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

	if _, ok := s.attributes[entityID]; !ok {
		s.attributes[entityID] = make(map[string]AttributeDefinition)
	}
	s.attributes[entityID][id] = attr
	return attr, nil
}

// GetAttribute retrieves an attribute by its ID for a given entity.
func (s *Store) GetAttribute(entityID, attributeID string) (AttributeDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entityAttributes, ok := s.attributes[entityID]
	if !ok {
		return AttributeDefinition{}, false
	}
	attribute, ok := entityAttributes[attributeID]
	return attribute, ok
}

// ListAttributes retrieves all attributes for a given entity.
func (s *Store) ListAttributes(entityID string) ([]AttributeDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.entities[entityID]; !ok {
		return nil, fmt.Errorf("entity with ID %s not found", entityID)
	}
	entityAttributes, ok := s.attributes[entityID]
	if !ok {
		// This case should ideally not happen if entities always have an attribute map initialized
		return []AttributeDefinition{}, nil
	}
	list := make([]AttributeDefinition, 0, len(entityAttributes))
	for _, attr := range entityAttributes {
		list = append(list, attr)
	}
	return list, nil
}

// UpdateAttribute updates an existing attribute.
func (s *Store) UpdateAttribute(entityID, attributeID, name, dataType, description string, isFilterable bool, isPii bool, isIndexed bool) (AttributeDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[entityID]; !ok {
		return AttributeDefinition{}, fmt.Errorf("entity with ID %s not found", entityID)
	}
	entityAttributes, ok := s.attributes[entityID]
	if !ok {
		return AttributeDefinition{}, fmt.Errorf("no attributes found for entity ID %s", entityID)
	}
	attr, ok := entityAttributes[attributeID]
	if !ok {
		return AttributeDefinition{}, fmt.Errorf("attribute with ID %s not found for entity ID %s", attributeID, entityID)
	}

	if name == "" {
		return AttributeDefinition{}, fmt.Errorf("attribute name cannot be empty")
	}
	if dataType == "" {
		return AttributeDefinition{}, fmt.Errorf("attribute dataType cannot be empty")
	}

	attr.Name = name
	attr.DataType = dataType
	attr.Description = description
	attr.IsFilterable = isFilterable
	attr.IsPii = isPii
	attr.IsIndexed = isIndexed
	attr.UpdatedAt = time.Now().UTC()
	s.attributes[entityID][attributeID] = attr
	return attr, nil
}

// DeleteAttribute removes an attribute from an entity.
// It also removes any field mappings associated with this attribute.
func (s *Store) DeleteAttribute(entityID, attributeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[entityID]; !ok {
		return fmt.Errorf("entity with ID %s not found", entityID)
	}
	entityAttributes, ok := s.attributes[entityID]
	if !ok {
		return fmt.Errorf("no attributes found for entity ID %s", entityID)
	}
	if _, ok := entityAttributes[attributeID]; !ok {
		return fmt.Errorf("attribute with ID %s not found for entity ID %s", attributeID, entityID)
	}

	// Remove field mappings associated with the attribute
	for sourceID, mappings := range s.fieldMappings {
		for mappingID, mapping := range mappings {
			if mapping.AttributeID == attributeID {
				delete(s.fieldMappings[sourceID], mappingID)
			}
		}
		if len(s.fieldMappings[sourceID]) == 0 {
			delete(s.fieldMappings, sourceID)
		}
	}

	delete(s.attributes[entityID], attributeID)
	return nil
}

// --- DataSourceConfig Methods ---

// CreateDataSource adds a new data source configuration to the store.
func (s *Store) CreateDataSource(config DataSourceConfig) (DataSourceConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Name == "" {
		return DataSourceConfig{}, fmt.Errorf("data source name cannot be empty")
	}
	if config.Type == "" {
		return DataSourceConfig{}, fmt.Errorf("data source type cannot be empty")
	}
	// Basic validation for ConnectionDetails, could be more sophisticated
	if config.ConnectionDetails == "" {
		return DataSourceConfig{}, fmt.Errorf("data source connection details cannot be empty")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	config.ID = id
	config.CreatedAt = now
	config.UpdatedAt = now

	s.dataSources[id] = config
	s.fieldMappings[id] = make(map[string]DataSourceFieldMapping) // Initialize field mappings map for this source
	return config, nil
}

// GetDataSources retrieves all data source configurations.
func (s *Store) GetDataSources() ([]DataSourceConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]DataSourceConfig, 0, len(s.dataSources))
	for _, ds := range s.dataSources {
		list = append(list, ds)
	}
	return list, nil
}

// GetDataSource retrieves a data source configuration by its ID.
func (s *Store) GetDataSource(id string) (DataSourceConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ds, ok := s.dataSources[id]
	if !ok {
		return DataSourceConfig{}, fmt.Errorf("data source with ID %s not found", id)
	}
	return ds, nil
}

// UpdateDataSource updates an existing data source configuration.
func (s *Store) UpdateDataSource(id string, config DataSourceConfig) (DataSourceConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingDs, ok := s.dataSources[id]
	if !ok {
		return DataSourceConfig{}, fmt.Errorf("data source with ID %s not found", id)
	}

	if config.Name == "" {
		return DataSourceConfig{}, fmt.Errorf("data source name cannot be empty")
	}
	if config.Type == "" {
		return DataSourceConfig{}, fmt.Errorf("data source type cannot be empty")
	}
	if config.ConnectionDetails == "" {
		return DataSourceConfig{}, fmt.Errorf("data source connection details cannot be empty")
	}

	existingDs.Name = config.Name
	existingDs.Type = config.Type
	existingDs.ConnectionDetails = config.ConnectionDetails
	existingDs.EntityID = config.EntityID // Update EntityID
	existingDs.UpdatedAt = time.Now().UTC()
	s.dataSources[id] = existingDs
	return existingDs, nil
}

// DeleteDataSource removes a data source configuration and its associated field mappings.
func (s *Store) DeleteDataSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.dataSources[id]; !ok {
		return fmt.Errorf("data source with ID %s not found", id)
	}
	delete(s.dataSources, id)
	delete(s.fieldMappings, id) // Also delete all associated field mappings
	return nil
}

// --- DataSourceFieldMapping Methods ---

// CreateFieldMapping adds a new field mapping to a data source.
func (s *Store) CreateFieldMapping(mapping DataSourceFieldMapping) (DataSourceFieldMapping, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.dataSources[mapping.SourceID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("data source with ID %s not found", mapping.SourceID)
	}
	if _, ok := s.entities[mapping.EntityID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("entity with ID %s not found", mapping.EntityID)
	}
	if entityAttrs, ok := s.attributes[mapping.EntityID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("no attributes found for entity ID %s", mapping.EntityID)
	} else if _, ok := entityAttrs[mapping.AttributeID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("attribute with ID %s not found for entity ID %s", mapping.AttributeID, mapping.EntityID)
	}

	if mapping.SourceFieldName == "" {
		return DataSourceFieldMapping{}, fmt.Errorf("source field name cannot be empty")
	}

	id := uuid.New().String()
	now := time.Now().UTC()
	mapping.ID = id
	mapping.CreatedAt = now
	mapping.UpdatedAt = now

	if _, ok := s.fieldMappings[mapping.SourceID]; !ok {
		s.fieldMappings[mapping.SourceID] = make(map[string]DataSourceFieldMapping)
	}
	s.fieldMappings[mapping.SourceID][id] = mapping
	return mapping, nil
}

// GetFieldMappings retrieves all field mappings for a given data source.
func (s *Store) GetFieldMappings(sourceID string) ([]DataSourceFieldMapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.dataSources[sourceID]; !ok {
		return nil, fmt.Errorf("data source with ID %s not found", sourceID)
	}

	sourceMappings, ok := s.fieldMappings[sourceID]
	if !ok {
		return []DataSourceFieldMapping{}, nil // No mappings yet for this source
	}

	list := make([]DataSourceFieldMapping, 0, len(sourceMappings))
	for _, fm := range sourceMappings {
		list = append(list, fm)
	}
	return list, nil
}

// GetFieldMapping retrieves a specific field mapping by its ID for a given data source.
func (s *Store) GetFieldMapping(sourceID, mappingID string) (DataSourceFieldMapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.dataSources[sourceID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("data source with ID %s not found", sourceID)
	}
	sourceMappings, ok := s.fieldMappings[sourceID]
	if !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("no field mappings found for data source ID %s", sourceID)
	}
	fm, ok := sourceMappings[mappingID]
	if !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("field mapping with ID %s not found for data source ID %s", mappingID, sourceID)
	}
	return fm, nil
}

// UpdateFieldMapping updates an existing field mapping.
func (s *Store) UpdateFieldMapping(sourceID, mappingID string, mapping DataSourceFieldMapping) (DataSourceFieldMapping, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.dataSources[sourceID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("data source with ID %s not found", sourceID)
	}
	if _, ok := s.entities[mapping.EntityID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("entity with ID %s not found", mapping.EntityID)
	}
	if entityAttrs, ok := s.attributes[mapping.EntityID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("no attributes found for entity ID %s", mapping.EntityID)
	} else if _, ok := entityAttrs[mapping.AttributeID]; !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("attribute with ID %s not found for entity ID %s", mapping.AttributeID, mapping.EntityID)
	}

	sourceMappings, ok := s.fieldMappings[sourceID]
	if !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("no field mappings found for data source ID %s", sourceID)
	}
	existingFm, ok := sourceMappings[mappingID]
	if !ok {
		return DataSourceFieldMapping{}, fmt.Errorf("field mapping with ID %s not found for data source ID %s", mappingID, sourceID)
	}

	if mapping.SourceFieldName == "" {
		return DataSourceFieldMapping{}, fmt.Errorf("source field name cannot be empty")
	}

	existingFm.SourceFieldName = mapping.SourceFieldName
	existingFm.EntityID = mapping.EntityID // Allow changing entity/attribute target
	existingFm.AttributeID = mapping.AttributeID
	existingFm.TransformationRule = mapping.TransformationRule
	existingFm.UpdatedAt = time.Now().UTC()
	s.fieldMappings[sourceID][mappingID] = existingFm
	return existingFm, nil
}

// DeleteFieldMapping removes a field mapping from a data source.
func (s *Store) DeleteFieldMapping(sourceID, mappingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.dataSources[sourceID]; !ok {
		return fmt.Errorf("data source with ID %s not found", sourceID)
	}
	sourceMappings, ok := s.fieldMappings[sourceID]
	if !ok {
		return fmt.Errorf("no field mappings found for data source ID %s", sourceID)
	}
	if _, ok := sourceMappings[mappingID]; !ok {
		return fmt.Errorf("field mapping with ID %s not found for data source ID %s", mappingID, sourceID)
	}
	delete(s.fieldMappings[sourceID], mappingID)
	if len(s.fieldMappings[sourceID]) == 0 { // Clean up map if no mappings left for this source
		delete(s.fieldMappings, sourceID)
	}
	return nil
}
