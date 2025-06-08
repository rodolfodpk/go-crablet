-- Agnostic event store for DCB, storing events of any type with JSONB tags and data.
CREATE TABLE events (
                        id UUID PRIMARY KEY,
                        type TEXT NOT NULL,
                        tags JSONB NOT NULL,
                        data JSONB NOT NULL,
                        position BIGSERIAL NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        causation_id UUID NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED,
                        correlation_id UUID NOT NULL REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_causation_id ON events (causation_id);
CREATE INDEX idx_events_correlation_id ON events (correlation_id);