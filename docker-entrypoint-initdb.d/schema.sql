-- Create the database (PostgreSQL doesn't support IF NOT EXISTS for CREATE DATABASE)
-- This will be run in the default postgres database
SELECT 'CREATE DATABASE dcb_app' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'dcb_app')\gexec

-- Use the database
\c dcb_app;

-- Agnostic event store for DCB, storing events of any type with TEXT[] tags and data.
-- Using transaction_id for proper ordering guarantees (see: https://event-driven.io/en/ordering_in_postgres_outbox/)
CREATE TABLE events (type VARCHAR(64) NOT NULL,
                     tags TEXT[] NOT NULL,
                     data JSON NOT NULL,
                     position BIGSERIAL NOT NULL,
                     transaction_id xid8 NOT NULL,
                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                     CONSTRAINT chk_event_type_length CHECK (LENGTH(type) <= 64));

-- Core indexes for essential operations (based on actual usage analysis)
CREATE INDEX idx_events_transaction_id ON events (transaction_id);
CREATE INDEX idx_events_transaction_position ON events (transaction_id, position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);

-- Performance optimization indexes
CREATE INDEX idx_events_type_position ON events (type, position);

-- Additional indexes for better read performance
CREATE INDEX idx_events_type ON events (type);

-- Optimized function to check append conditions using transaction_id for proper ordering
CREATE OR REPLACE FUNCTION check_append_condition(
    p_fail_if_events_match JSONB DEFAULT NULL,
    p_after_cursor JSONB DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    condition_count INTEGER;
BEGIN
    -- Check FailIfEventsMatch condition (with optional after cursor scope)
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
            -- Apply cursor-based after condition using (transaction_id, position)
            AND (p_after_cursor IS NULL OR 
                 (e.transaction_id > (p_after_cursor->>'transaction_id')::xid8) OR
                 (e.transaction_id = (p_after_cursor->>'transaction_id')::xid8 AND e.position > (p_after_cursor->>'position')::BIGINT))
            -- Only consider committed transactions for proper ordering
            AND e.transaction_id < pg_snapshot_xmin(pg_current_snapshot())
        )
        SELECT COUNT(*) INTO condition_count FROM event_matches;
        
        IF condition_count > 0 THEN
            RAISE EXCEPTION 'append condition violated: % matching events found', condition_count USING ERRCODE = 'DCB01', HINT = 'This is a concurrency violation - events matching the condition were found';
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
    INSERT INTO events (type, tags, data, transaction_id)
    SELECT 
        t.type,
        t.tag_string::TEXT[], -- Cast the array literal string to TEXT[]
        t.data,
        pg_current_xact_id()
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
    after_cursor JSONB;
BEGIN
    -- Extract condition parameters
    IF p_condition IS NOT NULL THEN
        fail_if_events_match := p_condition->'fail_if_events_match';
        IF p_condition->'after_cursor' IS NOT NULL AND p_condition->'after_cursor' != 'null' THEN
            after_cursor := p_condition->'after_cursor';
        END IF;
    END IF;
    
    -- Check append conditions first
    PERFORM check_append_condition(fail_if_events_match, after_cursor);
    
    -- If conditions pass, insert events using UNNEST for all cases
    PERFORM append_events_batch(p_types, p_tags, p_data);
END;
$$ LANGUAGE plpgsql;
    