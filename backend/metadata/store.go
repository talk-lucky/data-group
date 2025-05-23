package metadata

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store manages metadata entities and attributes in memory.
type Store struct {
	entities   map[string]EntityDefinition
	attributes map[string]map[string]AttributeDefinition // Key: EntityID, Key: AttributeID
	mu         sync.RWMutex
}

// NewStore creates and returns a new Store.
func NewStore() *Store {
	return &Store{
		entities:   make(map[string]EntityDefinition),
		attributes: make(map[string]map[string]AttributeDefinition),
	}
}

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
func (s *Store) DeleteEntity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[id]; !ok {
		return fmt.Errorf("entity with ID %s not found", id)
	}
	delete(s.entities, id)
	delete(s.attributes, id) // Also delete associated attributes
	return nil
}

// CreateAttribute adds a new attribute to an entity.
func (s *Store) CreateAttribute(entityID, name, dataType, description string) (AttributeDefinition, error) {
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
		ID:          id,
		EntityID:    entityID,
		Name:        name,
		DataType:    dataType,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
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
func (s *Store) UpdateAttribute(entityID, attributeID, name, dataType, description string) (AttributeDefinition, error) {
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
	attr.UpdatedAt = time.Now().UTC()
	s.attributes[entityID][attributeID] = attr
	return attr, nil
}

// DeleteAttribute removes an attribute from an entity.
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
	delete(s.attributes[entityID], attributeID)
	return nil
}
