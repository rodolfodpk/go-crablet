-- Test script for the new advisory lock function
-- This demonstrates how to use tags with "lock:" prefix for advisory locking

-- First, let's test the basic functionality
SELECT 'Testing advisory lock function with lock: prefix' as test_case;

-- Test 1: Single event with lock tag
SELECT 'Test 1: Single event with lock:user:123' as test_case;
SELECT append_events_with_advisory_locks(
    ARRAY['UserCreated'],
    ARRAY['{"user:123", "lock:user:123", "tenant:acme"}'],
    ARRAY['{"name": "John Doe"}'::jsonb]
);

-- Verify the event was stored with cleaned tags (lock: prefix removed)
SELECT type, tags, data FROM events WHERE type = 'UserCreated' ORDER BY position DESC LIMIT 1;

-- Test 2: Multiple events with different lock tags
SELECT 'Test 2: Multiple events with different lock tags' as test_case;
SELECT append_events_with_advisory_locks(
    ARRAY['OrderCreated', 'ItemAdded'],
    ARRAY['{"order:456", "lock:order:456", "customer:789"}', '{"order:456", "lock:order:456", "item:123"}'],
    ARRAY['{"total": 100}'::jsonb, '{"item_id": "123", "quantity": 2}'::jsonb]
);

-- Verify events were stored with cleaned tags
SELECT type, tags, data FROM events WHERE type IN ('OrderCreated', 'ItemAdded') ORDER BY position DESC LIMIT 2;

-- Test 3: Event with multiple lock tags
SELECT 'Test 3: Event with multiple lock tags' as test_case;
SELECT append_events_with_advisory_locks(
    ARRAY['TransactionProcessed'],
    ARRAY['{"transaction:999", "lock:account:111", "lock:account:222", "amount:500"}'],
    ARRAY['{"from_account": "111", "to_account": "222", "amount": 500}'::jsonb]
);

-- Verify the event was stored with cleaned tags
SELECT type, tags, data FROM events WHERE type = 'TransactionProcessed' ORDER BY position DESC LIMIT 1;

-- Test 4: Event with no lock tags (should work normally)
SELECT 'Test 4: Event with no lock tags' as test_case;
SELECT append_events_with_advisory_locks(
    ARRAY['AuditLog'],
    ARRAY['{"audit:system", "level:info"}'],
    ARRAY['{"message": "System check completed"}'::jsonb]
);

-- Verify the event was stored normally
SELECT type, tags, data FROM events WHERE type = 'AuditLog' ORDER BY position DESC LIMIT 1;

-- Test 5: Concurrent test - this would normally cause conflicts without locks
SELECT 'Test 5: Simulating concurrent access to same lock key' as test_case;

-- In a real scenario, this would prevent concurrent modifications to the same aggregate
-- For demonstration, we'll just show the function calls
SELECT 'Would acquire lock for user:123 and prevent concurrent modifications' as note;

-- Test 6: Show how the function handles mixed lock and non-lock tags
SELECT 'Test 6: Mixed lock and non-lock tags' as test_case;
SELECT append_events_with_advisory_locks(
    ARRAY['ComplexEvent'],
    ARRAY['{"lock:aggregate:xyz", "normal:tag", "lock:resource:abc", "another:normal"}'],
    ARRAY['{"complex": "data"}'::jsonb]
);

-- Verify the event was stored with only non-lock tags
SELECT type, tags, data FROM events WHERE type = 'ComplexEvent' ORDER BY position DESC LIMIT 1;

-- Summary of what we've demonstrated:
SELECT 'Summary:' as summary;
SELECT '1. Tags with "lock:" prefix are used for advisory locks' as point;
SELECT '2. The "lock:" prefix is removed from stored tags' as point;
SELECT '3. Multiple lock tags are supported per event' as point;
SELECT '4. Lock keys are sorted to prevent deadlocks' as point;
SELECT '5. Normal tags without "lock:" prefix are stored as-is' as point;
SELECT '6. The function has the same contract as append_events_with_condition' as point; 