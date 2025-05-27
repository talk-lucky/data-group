-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Table for Entity Definitions
CREATE TABLE entity_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table for Attribute Definitions
CREATE TABLE attribute_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID REFERENCES entity_definitions(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    data_type VARCHAR(50) NOT NULL, -- e.g., STRING, INT, BOOLEAN, DATETIME
    description TEXT,
    is_filterable BOOLEAN DEFAULT FALSE,
    is_pii BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (entity_id, name) -- Ensure attribute names are unique within an entity
);

-- Indexes
CREATE INDEX idx_entity_definitions_name ON entity_definitions(name);
CREATE INDEX idx_attribute_definitions_entity_id ON attribute_definitions(entity_id);
CREATE INDEX idx_attribute_definitions_name ON attribute_definitions(name);

-- Trigger function to update 'updated_at' timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for 'updated_at'
CREATE TRIGGER set_timestamp_entity_definitions
BEFORE UPDATE ON entity_definitions
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp_attribute_definitions
BEFORE UPDATE ON attribute_definitions
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
