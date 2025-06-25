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

-- Optimized function to check append conditions using CTEs and better query structure
CREATE OR REPLACE FUNCTION check_append_condition(
    p_fail_if_events_match JSONB DEFAULT NULL,
    p_after_position BIGINT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    condition_count INTEGER;
BEGIN
    -- Check FailIfEventsMatch condition (with optional after position scope)
    IF p_fail_if_events_match IS NOT NULL THEN
        -- Use CTE for better performance and readability
        WITH condition_queries AS (
            SELECT 
                jsonb_array_elements(p_fail_if_events_match->'items') AS query_item
        ),
        event_matches AS (
            SELECT DISTINCT e.position
            FROM events e
            CROSS JOIN condition_queries cq
            WHERE (
                -- Check event types if specified
                (cq.query_item->'event_types' IS NULL OR 
                 e.type = ANY(SELECT jsonb_array_elements_text(cq.query_item->'event_types')))
                AND
                -- Check tags if specified (using GIN index efficiently for TEXT[] arrays)
                (cq.query_item->'tags' IS NULL OR 
                 e.tags @> (
                     SELECT array_agg((obj->>'key') || ':' || (obj->>'value'))
                     FROM jsonb_array_elements((cq.query_item->'tags')::jsonb) AS obj
                 )::TEXT[])
            )
            -- Apply after position filter within query scope (DCB compliant)
            AND (p_after_position IS NULL OR e.position > p_after_position)
        )
        SELECT COUNT(*) INTO condition_count FROM event_matches;
        
        IF condition_count > 0 THEN
            RAISE EXCEPTION 'append condition violated: % matching events found', condition_count;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to batch insert events using UNNEST for better performance
CREATE OR REPLACE FUNCTION append_events_batch(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[]
) RETURNS VOID AS $$
BEGIN
    -- Use UNNEST for efficient batch insert with proper array handling
    INSERT INTO events (type, tags, data)
    SELECT 
        t.type,
        t.tag_string::TEXT[], -- Cast the array literal string to TEXT[]
        t.data
    FROM UNNEST(p_types, p_tags, p_data) AS t(type, tag_string, data);
END;
$$ LANGUAGE plpgsql;

-- Combined function to check conditions and append events atomically
CREATE OR REPLACE FUNCTION append_events_with_condition(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    fail_if_events_match JSONB;
    after_position BIGINT;
BEGIN
    -- Extract condition parameters
    IF p_condition IS NOT NULL THEN
        fail_if_events_match := p_condition->'fail_if_events_match';
        IF p_condition->'after' IS NOT NULL AND p_condition->'after' != 'null' THEN
            after_position := (p_condition->'after')::BIGINT;
        END IF;
    END IF;
    
    -- Check append conditions first
    PERFORM check_append_condition(fail_if_events_match, after_position);
    
    -- If conditions pass, insert events using UNNEST for all cases
    PERFORM append_events_batch(p_types, p_tags, p_data);
END;
$$ LANGUAGE plpgsql;
    