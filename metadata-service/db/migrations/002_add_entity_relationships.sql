-- Migration: Create entity_relationship_definitions table
-- Create the table
CREATE TABLE entity_relationship_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    source_entity_id UUID NOT NULL,
    target_entity_id UUID NOT NULL,
    relationship_type VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_source_entity
        FOREIGN KEY(source_entity_id)
        REFERENCES entity_definitions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_target_entity
        FOREIGN KEY(target_entity_id)
        REFERENCES entity_definitions(id)
        ON DELETE CASCADE,
    CONSTRAINT unique_source_target_name
        UNIQUE(source_entity_id, target_entity_id, name)
);

-- Create indexes for faster lookups
CREATE INDEX idx_er_source_entity_id ON entity_relationship_definitions(source_entity_id);
CREATE INDEX idx_er_target_entity_id ON entity_relationship_definitions(target_entity_id);
CREATE INDEX idx_er_relationship_type ON entity_relationship_definitions(relationship_type);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER set_timestamp_entity_relationship_definitions
BEFORE UPDATE ON entity_relationship_definitions
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
