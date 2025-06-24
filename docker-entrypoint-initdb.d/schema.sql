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

-- Function to check append conditions
CREATE OR REPLACE FUNCTION check_append_condition(
    p_fail_if_events_match JSONB DEFAULT NULL,
    p_after_position BIGINT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    condition_sql TEXT;
    condition_count INTEGER;
    query_item JSONB;
    event_types TEXT[];
    tags TEXT[];
    item_conditions TEXT[];
    combined_condition TEXT;
    args TEXT[];
    arg_count INTEGER;
BEGIN
    -- Check FailIfEventsMatch condition (with optional after position scope)
    IF p_fail_if_events_match IS NOT NULL THEN
        -- Build dynamic SQL from the query structure
        item_conditions := ARRAY[]::TEXT[];
        args := ARRAY[]::TEXT[];
        arg_count := 0;
        
        FOR query_item IN SELECT * FROM jsonb_array_elements(p_fail_if_events_match->'items')
        LOOP
            -- Extract event types
            event_types := ARRAY[]::TEXT[];
            IF query_item->'event_types' IS NOT NULL THEN
                SELECT array_agg(value::TEXT) INTO event_types 
                FROM jsonb_array_elements_text(query_item->'event_types');
            END IF;
            
            -- Extract tags (fix: use obj->>'key' and obj->>'value' with explicit cast)
            tags := ARRAY[]::TEXT[];
            IF query_item->'tags' IS NOT NULL THEN
                SELECT array_agg((obj->>'key') || ':' || (obj->>'value')) INTO tags
                FROM jsonb_array_elements((query_item->'tags')::jsonb) AS obj;
            END IF;
            
            -- Build condition for this item
            combined_condition := '';
            
            IF array_length(event_types, 1) > 0 THEN
                arg_count := arg_count + 1;
                combined_condition := 'type = ANY($' || arg_count || '::TEXT[])';
                args := array_append(args, array_to_string(event_types, ','));
            END IF;
            
            IF array_length(tags, 1) > 0 THEN
                IF combined_condition != '' THEN
                    combined_condition := combined_condition || ' AND ';
                END IF;
                arg_count := arg_count + 1;
                combined_condition := combined_condition || 'tags @> $' || arg_count || '::TEXT[]';
                args := array_append(args, array_to_string(tags, ','));
            END IF;
            
            IF combined_condition != '' THEN
                item_conditions := array_append(item_conditions, combined_condition);
            END IF;
        END LOOP;
        
        -- Build final SQL with query-scoped after position
        IF array_length(item_conditions, 1) > 0 THEN
            condition_sql := 'SELECT COUNT(*) FROM events WHERE (' || array_to_string(item_conditions, ' OR ') || ')';
            
            -- Add after position filter within query scope (DCB compliant)
            IF p_after_position IS NOT NULL THEN
                arg_count := arg_count + 1;
                condition_sql := condition_sql || ' AND position > $' || arg_count;
                args := array_append(args, p_after_position::TEXT);
            END IF;
            
            -- Execute the query with dynamic arguments
            EXECUTE condition_sql INTO condition_count USING args;
            
            IF condition_count > 0 THEN
                RAISE EXCEPTION 'append condition violated: % matching events found', condition_count;
            END IF;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to batch insert events
CREATE OR REPLACE FUNCTION append_events_batch(
    p_types TEXT[],
    p_tags TEXT[], -- array of Postgres array literals as strings
    p_data JSONB[]
) RETURNS BIGINT AS $$
DECLARE
    last_position BIGINT;
    values_clause TEXT;
    i INT;
BEGIN
    -- Build VALUES clause for true batch insert
    values_clause := '';
    FOR i IN 1..array_length(p_types, 1) LOOP
        IF i > 1 THEN
            values_clause := values_clause || ', ';
        END IF;
        values_clause := values_clause || 
            '(' || quote_literal(p_types[i]) || ', ' || 
            quote_literal(p_tags[i]) || '::text[], ' || 
            quote_literal(p_data[i]) || ')';
    END LOOP;
    
    -- Execute true batch insert
    EXECUTE 'INSERT INTO events (type, tags, data) VALUES ' || values_clause || ' RETURNING position' INTO last_position;
    
    RETURN last_position;
END;
$$ LANGUAGE plpgsql;

-- Combined function to check conditions and append events atomically
CREATE OR REPLACE FUNCTION append_events_with_condition(
    p_types TEXT[],
    p_tags TEXT[][],
    p_data JSONB[],
    p_condition JSONB DEFAULT NULL
) RETURNS BIGINT AS $$
DECLARE
    last_position BIGINT;
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
    
    -- If conditions pass, insert events
    SELECT append_events_batch(p_types, p_tags, p_data) INTO last_position;
    
    RETURN last_position;
END;
$$ LANGUAGE plpgsql;
    