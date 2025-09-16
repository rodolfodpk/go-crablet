-- Agnostic event store for DCB, storing events of any type with TEXT[] tags and data.
-- Using transaction_id for proper ordering guarantees (see: https://event-driven.io/en/ordering_in_postgres_outbox/)

-- Create the default events table
CREATE TABLE events (type VARCHAR(64) NOT NULL,
                     tags TEXT[] NOT NULL,
                     data JSON NOT NULL,
                     transaction_id xid8 NOT NULL,
                     position BIGSERIAL NOT NULL PRIMARY KEY,
                     occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                     CONSTRAINT chk_event_type_length CHECK (LENGTH(type) <= 64));

-- Create the commands table for command tracking
CREATE TABLE commands (
    transaction_id xid8 NOT NULL PRIMARY KEY,
    type VARCHAR(64) NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for commands table
-- CREATE INDEX idx_commands_type ON commands (type);
-- CREATE INDEX idx_commands_target_table ON commands (target_events_table);

-- Core indexes for essential operations
CREATE INDEX idx_events_transaction_position_btree ON events (transaction_id, position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);
CREATE INDEX idx_events_type ON events (type);

-- Function to batch insert events using UNNEST for better performance
-- Always uses 'events' table for maximum performance
CREATE OR REPLACE FUNCTION append_events_batch(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[]
) RETURNS VOID AS $$
BEGIN
    -- Insert directly into events table (no dynamic table name needed)
    INSERT INTO events (type, tags, data, transaction_id)
    SELECT 
        t.type,
        t.tag_string::TEXT[], -- Cast the array literal string to TEXT[]
        t.data,
        pg_current_xact_id()
    FROM UNNEST($1, $2, $3) AS t(type, tag_string, data);
END;
$$ LANGUAGE plpgsql;

-- Optimized function that receives primitive parameters instead of JSONB parsing
-- This eliminates the JSONB parsing overhead for much better performance
CREATE OR REPLACE FUNCTION append_events_if(
    p_types TEXT[],
    p_tags TEXT[],
    p_data JSONB[],
    p_event_types TEXT[] DEFAULT NULL,
    p_condition_tags TEXT[] DEFAULT NULL,
    p_after_cursor_tx_id xid8 DEFAULT NULL,
    p_after_cursor_position BIGINT DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    condition_count INTEGER;
    result JSONB;
BEGIN
    -- Initialize result
    result := '{"success": true, "message": "condition check passed"}'::JSONB;
    
    -- Check condition using direct array comparisons (no JSONB parsing)
    IF p_event_types IS NOT NULL OR p_condition_tags IS NOT NULL THEN
        SELECT COUNT(*)
        INTO condition_count
        FROM events e
        WHERE (
            -- Check event types if specified (direct array comparison)
            (p_event_types IS NULL OR e.type = ANY(p_event_types))
            AND
            -- Check tags if specified (direct array comparison)
            (p_condition_tags IS NULL OR e.tags @> p_condition_tags)
        )
        -- Apply cursor-based after condition using (transaction_id, position)
        AND (p_after_cursor_tx_id IS NULL OR
             (e.transaction_id > p_after_cursor_tx_id) OR
             (e.transaction_id = p_after_cursor_tx_id AND e.position > p_after_cursor_position))
        -- Only consider committed transactions for proper ordering
        AND e.transaction_id < pg_snapshot_xmin(pg_current_snapshot())
        LIMIT 1;
        
        IF condition_count > 0 THEN
            -- Return failure status instead of raising exception
            result := jsonb_build_object(
                'success', false,
                'message', 'append condition violated',
                'matching_events_count', condition_count,
                'error_code', 'DCB01'
            );
            RETURN result;
        END IF;
    END IF;
    
    -- If conditions pass, insert events using UNNEST for all cases
    PERFORM append_events_batch(p_types, p_tags, p_data);
    
    -- Return success status
    RETURN jsonb_build_object(
        'success', true,
        'message', 'events appended successfully',
        'events_count', array_length(p_types, 1)
    );
END;
$$ LANGUAGE plpgsql;

