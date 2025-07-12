#!/bin/bash

# Database Connection Pool Monitor
# This script helps monitor PostgreSQL connection pool health

set -e

# Configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-crablet}
DB_USER=${DB_USER:-crablet}
MONITOR_INTERVAL=${MONITOR_INTERVAL:-5}

echo "🔍 Database Connection Pool Monitor"
echo "=================================="
echo "Host: $DB_HOST:$DB_PORT"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo "Monitor interval: ${MONITOR_INTERVAL}s"
echo ""

# Function to get connection stats
get_connection_stats() {
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            datname,
            numbackends as active_connections,
            xact_commit,
            xact_rollback,
            blks_read,
            blks_hit
        FROM pg_stat_database 
        WHERE datname = '$DB_NAME';
    " 2>/dev/null || echo "ERROR: Could not connect to database"
}

# Function to get detailed connection info
get_detailed_connections() {
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            pid,
            usename,
            application_name,
            client_addr,
            state,
            query_start,
            state_change,
            query
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME'
        ORDER BY query_start DESC;
    " 2>/dev/null || echo "ERROR: Could not get connection details"
}

# Function to check for long-running queries
check_long_running_queries() {
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            pid,
            usename,
            application_name,
            state,
            EXTRACT(EPOCH FROM (now() - query_start)) as duration_seconds,
            query
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME' 
        AND state = 'active'
        AND query_start < now() - interval '30 seconds'
        ORDER BY query_start;
    " 2>/dev/null || echo "ERROR: Could not check long-running queries"
}

# Function to check connection pool exhaustion
check_connection_pool() {
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            'Connection Pool Status' as status,
            count(*) as total_connections,
            count(*) FILTER (WHERE state = 'active') as active_connections,
            count(*) FILTER (WHERE state = 'idle') as idle_connections,
            count(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction,
            count(*) FILTER (WHERE state = 'disabled') as disabled_connections
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME';
    " 2>/dev/null || echo "ERROR: Could not check connection pool"
}

# Main monitoring loop
echo "Starting monitoring... (Press Ctrl+C to stop)"
echo ""

while true; do
    echo "📊 $(date '+%Y-%m-%d %H:%M:%S')"
    echo "----------------------------------------"
    
    # Get basic stats
    echo "📈 Database Statistics:"
    get_connection_stats
    echo ""
    
    # Check connection pool
    echo "🔗 Connection Pool Status:"
    check_connection_pool
    echo ""
    
    # Check for long-running queries
    echo "⏱️  Long-running Queries (>30s):"
    long_running=$(check_long_running_queries)
    if [[ "$long_running" == *"ERROR"* ]]; then
        echo "$long_running"
    elif [[ -z "$long_running" ]]; then
        echo "✅ No long-running queries detected"
    else
        echo "$long_running"
    fi
    echo ""
    
    # Check for idle in transaction connections (potential leaks)
    echo "🚨 Idle in Transaction Connections (potential leaks):"
    idle_in_txn=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            pid,
            usename,
            application_name,
            state_change,
            query
        FROM pg_stat_activity 
        WHERE datname = '$DB_NAME' 
        AND state = 'idle in transaction'
        ORDER BY state_change;
    " 2>/dev/null || echo "ERROR: Could not check idle in transaction")
    
    if [[ "$idle_in_txn" == *"ERROR"* ]]; then
        echo "$idle_in_txn"
    elif [[ -z "$idle_in_txn" ]]; then
        echo "✅ No idle in transaction connections"
    else
        echo "$idle_in_txn"
    fi
    echo ""
    
    echo "========================================"
    echo ""
    
    sleep "$MONITOR_INTERVAL"
done 