-- Agnostic event store for DCB, storing events of any type with JSONB tags and data.
CREATE TABLE events (
                        id VARCHAR(64) PRIMARY KEY,
                        type TEXT NOT NULL,
                        tags JSONB NOT NULL,
                        data JSONB NOT NULL,
                        position BIGSERIAL NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        causation_id VARCHAR(64) NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED,
                        correlation_id VARCHAR(64) NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED
);

-- Core indexes for essential operations
CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_causation_id ON events (causation_id);
CREATE INDEX idx_events_correlation_id ON events (correlation_id);

-- Performance optimization indexes
CREATE INDEX idx_events_type_position ON events (type, position);
CREATE INDEX idx_events_created_at ON events (created_at);

-- Composite indexes for common query patterns
-- Note: GIN indexes don't support INCLUDE, so we use a regular B-tree index for this pattern
CREATE INDEX idx_events_tags_position ON events (position, type) WHERE tags IS NOT NULL;

-- Partial indexes for specific event types (optional - add based on your most common event types)
-- CREATE INDEX idx_events_course_events ON events (position) WHERE type IN ('CourseCreated', 'StudentEnrolledInCourse', 'StudentUnenrolledFromCourse');
-- CREATE INDEX idx_events_student_events ON events (position) WHERE type IN ('StudentRegistered', 'StudentEnrolledInCourse', 'StudentUnenrolledFromCourse');

-- Statistics and maintenance
ANALYZE events;

-- Optional: Add table partitioning for very large datasets (10M+ events)
-- CREATE TABLE events_partitioned (
--     id VARCHAR(64),
--     type TEXT NOT NULL,
--     tags JSONB NOT NULL,
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