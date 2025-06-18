-- Agnostic event store for DCB, storing events of any type with TEXT[] tags and data.
CREATE TABLE events (
                        id VARCHAR(64) PRIMARY KEY,
                        type TEXT NOT NULL,
                        tags TEXT[] NOT NULL,
                        data JSON NOT NULL,
                        position BIGSERIAL NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        causation_id VARCHAR(64) NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED,
                        correlation_id VARCHAR(64) NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED
);

-- Core indexes for essential operations (based on actual usage analysis)
CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_correlation_id ON events (correlation_id);

-- Performance optimization indexes
CREATE INDEX idx_events_type_position ON events (type, position);

-- Statistics and maintenance
ANALYZE events;

-- Optional: Add table partitioning for very large datasets (10M+ events)
-- CREATE TABLE events_partitioned (
--     id VARCHAR(64),
--     type TEXT NOT NULL,
--     tags TEXT[] NOT NULL,
--     data JSONB NOT NULL,
--     position BIGSERIAL NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
--     causation_id VARCHAR(64) NOT NULL,
--     correlation_id VARCHAR(64) NOT NULL
-- ) PARTITION BY RANGE (position);

-- Performance tuning hints for PostgreSQL
ALTER TABLE events SET (fillfactor = 90);
ALTER TABLE events SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE events SET (autovacuum_analyze_scale_factor = 0.05);