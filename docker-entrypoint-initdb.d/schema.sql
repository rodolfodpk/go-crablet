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
