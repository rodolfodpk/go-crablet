-- Create the database (PostgreSQL doesn't support IF NOT EXISTS for CREATE DATABASE)
-- This will be run in the default postgres database
SELECT 'CREATE DATABASE crablet' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'crablet')\gexec

-- Use the database
\c crablet;

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

-- Optimized function to check append conditions using transaction_id for proper ordering
-- Returns JSONB with status instead of raising exceptions for better performance
-- Always uses 'events' table for maximum performance
CREATE OR REPLACE FUNCTION check_append_condition(
    p_fail_if_events_match JSONB DEFAULT NULL,
    p_after_cursor JSONB DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    condition_count INTEGER;
    query_text TEXT;
    result JSONB;
BEGIN
    -- Initialize result
    result := '{"success": true, "message": "condition check passed"}'::JSONB;
    
    -- Check FailIfEventsMatch condition (with optional after cursor scope)
    IF p_fail_if_events_match IS NOT NULL THEN
        -- Build query for events table (no dynamic table name needed)
        query_text := '
            WITH condition_queries AS (
                SELECT 
                    jsonb_array_elements($1->''items'') AS query_item
            ),
            event_matches AS (
                SELECT DISTINCT e.position
                FROM events e
                CROSS JOIN condition_queries cq
                WHERE (
                    -- Check event types if specified
                    (cq.query_item->''event_types'' IS NULL OR 
                     e.type = ANY(SELECT jsonb_array_elements_text(cq.query_item->''event_types'')))
                    AND
                    -- Check tags if specified (using GIN index efficiently for TEXT[] arrays)
                    (cq.query_item->''tags'' IS NULL OR 
                     e.tags @> (
                         SELECT array_agg((obj->>''key'') || '':'' || (obj->>''value''))
                         FROM jsonb_array_elements((cq.query_item->''tags'')::jsonb) AS obj
                     )::TEXT[])
                )
                -- Apply cursor-based after condition using (transaction_id, position)
                AND ($2 IS NULL OR 
                     (e.transaction_id > ($2->>''transaction_id'')::xid8) OR
                     (e.transaction_id = ($2->>''transaction_id'')::xid8 AND e.position > ($2->>''position'')::BIGINT))
                -- Only consider committed transactions for proper ordering
                AND e.transaction_id < pg_snapshot_xmin(pg_current_snapshot())
            )
            SELECT COUNT(*) FROM event_matches
        ';
        
        EXECUTE query_text INTO condition_count USING p_fail_if_events_match, p_after_cursor;
        
        IF condition_count > 0 THEN
            -- Return failure status instead of raising exception
            result := jsonb_build_object(
                'success', false,
                'message', 'append condition violated',
                'matching_events_count', condition_count,
                'error_code', 'DCB01'
            );
        END IF;
    END IF;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

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

-- Combined function to check conditions and append events atomically
-- Returns JSONB with status instead of raising exceptions for better performance
-- Always uses 'events' table for maximum performance
CREATE OR REPLACE FUNCTION append_events_with_condition(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL
) RETURNS JSONB AS $$
DECLARE
    fail_if_events_match JSONB;
    after_cursor JSONB;
    condition_result JSONB;
BEGIN
    -- Extract condition parameters
    IF p_condition IS NOT NULL THEN
        fail_if_events_match := p_condition->'fail_if_events_match';
        IF p_condition->'after_cursor' IS NOT NULL AND p_condition->'after_cursor' != 'null' THEN
            after_cursor := p_condition->'after_cursor';
        END IF;
    END IF;
    
    -- Check append conditions first
    condition_result := check_append_condition(fail_if_events_match, after_cursor);
    
    -- If conditions failed, return the failure status
    IF (condition_result->>'success')::boolean = false THEN
        RETURN condition_result;
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

-- Function to acquire advisory locks based on tags with "lock:" prefix
-- This function takes the same contract as append_events_with_condition but adds locking
-- Returns JSONB with status instead of raising exceptions for better performance
-- Always uses 'events' table for maximum performance
CREATE OR REPLACE FUNCTION append_events_with_advisory_locks(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL,
    p_lock_timeout_ms INTEGER DEFAULT 5000 -- 5 second default timeout
) RETURNS JSONB AS $$
DECLARE
    fail_if_events_match JSONB;
    after_cursor JSONB;
    cleaned_tags TEXT[];
    lock_keys TEXT[];
    tag_array TEXT[];
    tag_key TEXT;
    lock_key TEXT;
    i INTEGER;
    lock_timeout_setting TEXT;
    condition_result JSONB;
BEGIN
    -- Set lock timeout for this transaction
    lock_timeout_setting := current_setting('lock_timeout', true);
    PERFORM set_config('lock_timeout', p_lock_timeout_ms::TEXT, false);
    
    -- Extract condition parameters
    IF p_condition IS NOT NULL THEN
        fail_if_events_match := p_condition->'fail_if_events_match';
        IF p_condition->'after_cursor' IS NOT NULL AND p_condition->'after_cursor' != 'null' THEN
            after_cursor := p_condition->'after_cursor';
        END IF;
    END IF;
    
    -- Process each event's tags to extract lock keys and clean tags
    FOR i IN 1..array_length(p_tags, 1) LOOP
        -- Parse the tag array string into actual array
        tag_array := p_tags[i]::TEXT[];
        
        -- Initialize arrays for this event
        cleaned_tags := '{}';
        lock_keys := '{}';
        
        -- Process each tag in the array
        FOREACH tag_key IN ARRAY tag_array LOOP
            -- Check if tag starts with "lock:"
            IF tag_key LIKE 'lock:%' THEN
                -- Extract the lock key (remove "lock:" prefix)
                lock_key := substring(tag_key from 6); -- 'lock:' is 5 chars, so start from position 6
                
                -- Add to lock keys array
                lock_keys := array_append(lock_keys, lock_key);
                
                -- Don't add to cleaned tags (remove the lock: prefix entirely)
            ELSE
                -- Add to cleaned tags (no lock: prefix)
                cleaned_tags := array_append(cleaned_tags, tag_key);
            END IF;
        END LOOP;
        
        -- Acquire advisory locks for all lock keys (sorted to prevent deadlocks)
        IF array_length(lock_keys, 1) > 0 THEN
            -- Sort lock keys to prevent deadlocks
            SELECT array_agg(key ORDER BY key) INTO lock_keys FROM unnest(lock_keys) AS key;
            
            -- Acquire locks for each key
            FOREACH lock_key IN ARRAY lock_keys LOOP
                -- Use pg_advisory_xact_lock for transaction-scoped locks
                -- Convert string to hash for advisory lock
                PERFORM pg_advisory_xact_lock(hashtext(lock_key));
            END LOOP;
        END IF;
        
        -- Update the tags array with cleaned tags
        p_tags[i] := array_to_string(cleaned_tags, ',');
    END LOOP;
    
    -- Check append conditions first
    condition_result := check_append_condition(fail_if_events_match, after_cursor);
    
    -- If conditions failed, return the failure status
    IF (condition_result->>'success')::boolean = false THEN
        -- Restore original lock timeout setting
        IF lock_timeout_setting IS NOT NULL THEN
            PERFORM set_config('lock_timeout', lock_timeout_setting, false);
        END IF;
        RETURN condition_result;
    END IF;
    
    -- If conditions pass, insert events using UNNEST for all cases
    PERFORM append_events_batch(p_types, p_tags, p_data);
    
    -- Restore original lock timeout setting
    IF lock_timeout_setting IS NOT NULL THEN
        PERFORM set_config('lock_timeout', lock_timeout_setting, false);
    END IF;
    
    -- Return success status
    RETURN jsonb_build_object(
        'success', true,
        'message', 'events appended successfully with advisory locks',
        'events_count', array_length(p_types, 1)
    );
END;
$$ LANGUAGE plpgsql;

-- Removed backward compatibility functions - now using simplified functions directly
-- All functions now use the 'events' table for maximum performance
-- Debug and monitoring functions moved to debug.sql
    