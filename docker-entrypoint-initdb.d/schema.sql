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
                     transaction_id xid8 NOT NULL,
                     position BIGSERIAL NOT NULL PRIMARY KEY,
                     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                     CONSTRAINT chk_event_type_length CHECK (LENGTH(type) <= 64));

-- Core indexes for essential operations (based on actual usage analysis)
-- BRIN index for efficient range queries on transaction_id and position (causal ordering)
-- This handles all our cursor-based queries efficiently
-- CREATE INDEX idx_events_transaction_position_brin ON events USING BRIN (transaction_id, position);
CREATE INDEX idx_events_transaction_position_btree ON events (transaction_id, position);
CREATE INDEX idx_events_tags ON events USING GIN (tags);

-- Performance optimization indexes for filtering
-- CREATE INDEX idx_events_type_tags ON events USING GIN (type, tags);

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

-- Function to acquire advisory locks based on tags with "lock:" prefix
-- This function takes the same contract as append_events_with_condition but adds locking
CREATE OR REPLACE FUNCTION append_events_with_advisory_locks(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL,
    p_lock_timeout_ms INTEGER DEFAULT 5000 -- 5 second default timeout
) RETURNS VOID AS $$
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
    PERFORM check_append_condition(fail_if_events_match, after_cursor);
    
    -- If conditions pass, insert events using UNNEST for all cases
    PERFORM append_events_batch(p_types, p_tags, p_data);
    
    -- Restore original lock timeout setting
    IF lock_timeout_setting IS NOT NULL THEN
        PERFORM set_config('lock_timeout', lock_timeout_setting, false);
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Restore original lock timeout setting on error
        IF lock_timeout_setting IS NOT NULL THEN
            PERFORM set_config('lock_timeout', lock_timeout_setting, false);
        END IF;
        RAISE;
END;
$$ LANGUAGE plpgsql;

-- Function to monitor advisory lock usage
CREATE OR REPLACE FUNCTION get_advisory_lock_stats() 
RETURNS TABLE (
    lock_id BIGINT,
    database_id OID,
    object_id BIGINT,
    session_id INTEGER,
    application_name TEXT,
    client_addr INET,
    backend_start TIMESTAMPTZ,
    query_start TIMESTAMPTZ,
    state TEXT,
    wait_event_type TEXT,
    wait_event TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        al.lockid,
        al.database,
        al.objid,
        p.pid,
        p.application_name,
        p.client_addr,
        p.backend_start,
        p.query_start,
        p.state,
        p.wait_event_type,
        p.wait_event
    FROM pg_locks al
    JOIN pg_stat_activity p ON al.pid = p.pid
    WHERE al.locktype = 'advisory'
    AND p.state != 'idle'
    ORDER BY al.lockid, p.backend_start;
END;
$$ LANGUAGE plpgsql;

-- Function to get advisory lock count
CREATE OR REPLACE FUNCTION get_advisory_lock_count() 
RETURNS INTEGER AS $$
DECLARE
    lock_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO lock_count
    FROM pg_locks 
    WHERE locktype = 'advisory';
    
    RETURN lock_count;
END;
$$ LANGUAGE plpgsql;
    