-- Create the database (PostgreSQL doesn't support IF NOT EXISTS for CREATE DATABASE)
-- This will be run in the default postgres database
SELECT 'CREATE DATABASE dcb_app' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'dcb_app')\gexec

-- Use the database
\c dcb_app;

-- Agnostic event store for DCB, storing events of any type with TEXT[] tags and data.
CREATE TABLE events (type TEXT NOT NULL,
                     tags TEXT[] NOT NULL,
                     data JSON NOT NULL,
                     position BIGSERIAL NOT NULL,
                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP);

-- Core indexes for essential operations (based on actual usage analysis)
CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);

-- Performance optimization indexes
CREATE INDEX idx_events_type_position ON events (type, position);

-- Additional indexes for better read performance
CREATE INDEX idx_events_type ON events (type);
