// Quick AppendIfIsolated Benchmark (30s)
// Tests Serializable isolation level scenarios

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const appendIfIsolatedOperations = new Rate('append_if_isolated_operations');
const concurrencyErrors = new Rate('concurrency_errors');
const serializationErrors = new Rate('serialization_errors');
const appendSuccess = new Rate('append_success');

export const options = {
  stages: [
    { duration: '5s', target: 2 },   // Ramp up to 2 users
    { duration: '20s', target: 5 },  // Ramp up to 5 users
    { duration: '5s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2s
    http_req_failed: ['rate<0.15'],    // Error rate should be below 15%
  },
};

// Test data
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Generate unique IDs for each request
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Generate unique event with random IDs
function generateUniqueEvent(eventType, tagPrefixes) {
    const tags = tagPrefixes.map(prefix => `${prefix}:${generateUniqueId(prefix)}`);
    return {
        type: eventType,
        data: JSON.stringify({ 
            timestamp: new Date().toISOString(),
            id: generateUniqueId('event'),
            value: Math.random() * 1000
        }),
        tags: tags
    };
}

// Generate unique query with random IDs
function generateUniqueQuery(eventTypes, tagPrefixes) {
    const tags = tagPrefixes.map(prefix => `${prefix}:${generateUniqueId(prefix)}`);
    return {
        items: [{
            types: eventTypes,
            tags: tags
        }]
    };
}

// Quick appendIfSerializable scenarios (Serializable isolation)
const APPEND_IF_SERIALIZABLE_QUICK_SCENARIOS = {
    // Single event with condition (should succeed)
    SINGLE_WITH_CONDITION: {
        name: 'Single Event with Condition (Success)',
        payload: () => ({
            events: generateUniqueEvent('UserCreated', ['user', 'tenant']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['UserCreated'], ['user'])
            }
        })
    },
    
    // Single event without condition (should succeed)
    SINGLE_WITHOUT_CONDITION: {
        name: 'Single Event without Condition',
        payload: () => ({
            events: generateUniqueEvent('AccountOpened', ['account', 'user'])
        })
    },
    
    // Small batch with condition (should succeed)
    SMALL_BATCH_WITH_CONDITION: {
        name: 'Small Batch with Condition (Success)',
        payload: () => ({
            events: [
                generateUniqueEvent('OrderCreated', ['order', 'customer']),
                generateUniqueEvent('ItemAdded', ['order', 'item'])
            ],
            condition: {
                failIfEventsMatch: generateUniqueQuery(['OrderCreated'], ['order'])
            }
        })
    },
    
    // Condition that should fail (duplicate detection)
    CONDITION_FAIL: {
        name: 'Condition Fail (Duplicate Detection)',
        payload: () => {
            const eventType = 'DuplicateEvent';
            const tags = [`duplicate:${generateUniqueId('duplicate')}`];
            return {
                events: generateUniqueEvent(eventType, ['duplicate']),
                condition: {
                    failIfEventsMatch: {
                        items: [{
                            types: [eventType],
                            tags: tags
                        }]
                    }
                }
            };
        }
    },
    
    // After position condition test
    AFTER_POSITION_CONDITION: {
        name: 'After Position Condition Test',
        payload: () => ({
            events: generateUniqueEvent('AfterPositionEvent', ['after', 'position']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['AfterPositionEvent'], ['after']),
                after: '1000' // After position 1000
            }
        })
    }
};

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-Append-If-Isolation': 'serializable',
        },
        timeout: '15s', // Higher timeout due to Serializable isolation
    };

    // Test different appendIfSerializable scenarios
    const scenarios = Object.values(APPEND_IF_SERIALIZABLE_QUICK_SCENARIOS);
    const randomScenario = scenarios[Math.floor(Math.random() * scenarios.length)];
    
    const payload = randomScenario.payload();
    const response = http.post(
        `${BASE_URL}/append-if`,
        JSON.stringify(payload),
        params
    );

    // Check response
    const success = check(response, {
        [`${randomScenario.name} status is 200`]: (r) => r.status === 200,
        [`${randomScenario.name} has valid response`]: (r) => {
            if (r.status !== 200) return false;
            try {
                const body = JSON.parse(r.body);
                return body.hasOwnProperty('durationInMicroseconds') && 
                       body.hasOwnProperty('appendConditionFailed');
            } catch {
                return false;
            }
        }
    });

    // Track metrics
    if (success) {
        appendSuccess.add(1);
        appendIfIsolatedOperations.add(1);
        
        // Check if condition failed
        try {
            const body = JSON.parse(response.body);
            if (body.appendConditionFailed) {
                concurrencyErrors.add(1);
            }
        } catch {
            // Ignore parsing errors
        }
    } else {
        appendIfIsolatedOperations.add(1);
        
        // Check for specific error types
        if (response.status === 409) { // Conflict status
            serializationErrors.add(1);
        }
    }

    // Longer sleep for Serializable isolation
    sleep(0.2);
}

// Setup function to ensure database is ready
export function setup() {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    // Test health endpoint
    const healthRes = http.get(`${BASE_URL}/health`, params);
    check(healthRes, {
        'health check status is 200': (r) => r.status === 200,
    });

    if (healthRes.status !== 200) {
        throw new Error('Web app is not ready');
    }

    // Clean up database before benchmark
    const cleanupRes = http.post(`${BASE_URL}/cleanup`, null, params);
    check(cleanupRes, {
        'cleanup status is 200': (r) => r.status === 200,
    });

    console.log('Setup completed - database cleaned and ready for appendIfSerializable quick benchmark');
}

// Teardown function
export function teardown(data) {
    console.log('AppendIfSerializable quick benchmark completed');
} 