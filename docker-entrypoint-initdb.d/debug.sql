-- Debug and monitoring functions for go-crablet
-- These functions are currently unused by the Go code but available for future use
-- or manual debugging of advisory lock scenarios

-- Function to monitor advisory lock usage
-- NOTE: This function is currently unused by the Go code but available for future use
-- or manual debugging of advisory lock scenarios
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
-- NOTE: This function is currently unused by the Go code but available for future use
-- or manual debugging of advisory lock scenarios
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