-- Agnostic event store for DCB, storing events of any type with JSONB tags and data.
CREATE TABLE events (
                        id UUID PRIMARY KEY,
                        type TEXT NOT NULL,
                        tags JSONB NOT NULL,
                        data JSONB NOT NULL,
                        position BIGSERIAL NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        causation_id UUID REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED, -- Nullable, deferred
                        correlation_id UUID REFERENCES events(id) DEFERRABLE INITIALLY DEFERRED -- Nullable, deferred
);

CREATE INDEX idx_events_position ON events (position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_causation_id_not_null ON events (causation_id) WHERE causation_id IS NOT NULL;
CREATE INDEX idx_events_correlation_id_not_null ON events (correlation_id) WHERE correlation_id IS NOT NULL;

-- Update the append_events function to accept event types
CREATE OR REPLACE FUNCTION append_events(
    p_ids UUID[],
    p_types TEXT[],
    p_tags JSONB[],
    p_data JSONB[],
    p_query_tags JSONB,
    p_last_position BIGINT,
    p_causation_ids UUID[],
    p_correlation_ids UUID[],
    p_query_event_types TEXT[] DEFAULT NULL
) RETURNS BIGINT[] AS $$
DECLARE
v_positions BIGINT[];
    v_array_length INT;
    v_position BIGINT;
    i INT;
BEGIN
    -- Validate input array lengths
    v_array_length := array_length(p_ids, 1);
    IF v_array_length IS NULL OR v_array_length = 0 THEN
        RAISE EXCEPTION 'Input arrays are empty';
END IF;

    IF array_length(p_types, 1) != v_array_length OR
       array_length(p_tags, 1) != v_array_length OR
       array_length(p_data, 1) != v_array_length OR
       array_length(p_causation_ids, 1) != v_array_length OR
       array_length(p_correlation_ids, 1) != v_array_length THEN
        RAISE EXCEPTION 'Input arrays have inconsistent lengths';
END IF;

    -- Check for conflicting events (optimistic locking)
    IF EXISTS (
        SELECT 1
        FROM events
        WHERE position > p_last_position
          AND tags @> p_query_tags
          AND (p_query_event_types IS NULL OR
               array_length(p_query_event_types, 1) = 0 OR
               type = ANY(p_query_event_types))
    ) THEN
        RAISE EXCEPTION 'Consistency violation: new events match query since position %', p_last_position;
END IF;

    -- Insert events one by one using a loop
BEGIN
        v_positions := ARRAY[]::BIGINT[];
FOR i IN 1..v_array_length LOOP
                INSERT INTO events (id, type, tags, data, causation_id, correlation_id)
                VALUES (
                           p_ids[i],
                           p_types[i],
                           p_tags[i],
                           p_data[i],
                           p_causation_ids[i],
                           p_correlation_ids[i]
                       )
                RETURNING position INTO v_position;

                v_positions := array_append(v_positions, v_position);
END LOOP;
EXCEPTION
        WHEN foreign_key_violation THEN
            RAISE EXCEPTION 'Foreign key violation: invalid causation_id or correlation_id';
WHEN others THEN
            RAISE EXCEPTION 'Error inserting events: %', SQLERRM;
END;

RETURN v_positions;
END;
$$ LANGUAGE plpgsql;

-- Also update the append_events_batch function
CREATE OR REPLACE FUNCTION append_events_batch(
    p_ids UUID[],
    p_types TEXT[],
    p_tags JSONB[],
    p_data JSONB[],
    p_query_tags JSONB,
    p_last_position BIGINT,
    p_causation_ids UUID[],
    p_correlation_ids UUID[],
    p_query_event_types TEXT[] DEFAULT NULL
) RETURNS BIGINT[] AS $$
DECLARE
v_positions BIGINT[];
    v_array_length INT;
BEGIN
    -- Validate input array lengths
    v_array_length := array_length(p_ids, 1);
    IF v_array_length IS NULL OR v_array_length = 0 THEN
        RAISE EXCEPTION 'Input arrays are empty';
END IF;

    IF array_length(p_types, 1) != v_array_length OR
       array_length(p_tags, 1) != v_array_length OR
       array_length(p_data, 1) != v_array_length OR
       array_length(p_causation_ids, 1) != v_array_length OR
       array_length(p_correlation_ids, 1) != v_array_length THEN
        RAISE EXCEPTION 'Input arrays have inconsistent lengths';
END IF;

    -- Check for conflicting events (optimistic locking)
    IF EXISTS (
        SELECT 1
        FROM events
        WHERE position > p_last_position
          AND tags @> p_query_tags
          AND (p_query_event_types IS NULL OR
               array_length(p_query_event_types, 1) = 0 OR
               type = ANY(p_query_event_types))
    ) THEN
        RAISE EXCEPTION 'Consistency violation: new events match query since position %', p_last_position;
END IF;

    -- Insert all events in a single batch using UNNEST
BEGIN
WITH inserted AS (
INSERT INTO events (id, type, tags, data, causation_id, correlation_id)
SELECT
    unnest(p_ids) AS id,
    unnest(p_types) AS type,
    unnest(p_tags) AS tags,
    unnest(p_data) AS data,
    unnest(p_causation_ids) AS causation_id,
    unnest(p_correlation_ids) AS correlation_id
    RETURNING position
)
SELECT array_agg(position) INTO v_positions FROM inserted;
EXCEPTION
        WHEN foreign_key_violation THEN
            RAISE EXCEPTION 'Foreign key violation: invalid causation_id or correlation_id in batch';
WHEN others THEN
            RAISE EXCEPTION 'Error inserting events: %', SQLERRM;
END;

RETURN v_positions;
END;
$$ LANGUAGE plpgsql;