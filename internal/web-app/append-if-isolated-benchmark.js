// Full AppendIfIsolated Benchmark (6m)
// Tests Serializable isolation level scenarios with comprehensive load

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
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 50 },    // Ramp up to 50 users
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '2m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
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

// AppendIfIsolated specific scenarios (Serializable isolation)
const APPEND_IF_ISOLATED_SCENARIOS = {
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
    
    // Complex condition with multiple event types
    COMPLEX_CONDITION: {
        name: 'Complex Condition (Multiple Types)',
        payload: () => ({
            events: [
                generateUniqueEvent('TransactionInitiated', ['transaction', 'account']),
                generateUniqueEvent('BalanceChecked', ['account', 'system'])
            ],
            condition: {
                failIfEventsMatch: {
                    items: [
                        {
                            types: ['TransactionInitiated'],
                            tags: ['transaction']
                        },
                        {
                            types: ['BalanceChecked'],
                            tags: ['account']
                        }
                    ]
                }
            }
        })
    },
    
    // Serializable-specific concurrency scenario
    SERIALIZABLE_CONCURRENCY: {
        name: 'Serializable Concurrency Test',
        payload: () => ({
            events: generateUniqueEvent('SerializableEvent', ['serializable', 'test']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['SerializableEvent'], ['serializable'])
            }
        })
    },
    
    // Medium batch with condition (smaller due to Serializable overhead)
    MEDIUM_BATCH_WITH_CONDITION: {
        name: 'Medium Batch with Condition',
        payload: () => ({
            events: Array.from({ length: 10 }, (_, i) => 
                generateUniqueEvent('LogEntry', ['service', 'level', 'trace'])
            ),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['LogEntry'], ['service'])
            }
        })
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
        },
        timeout: '20s', // Higher timeout due to Serializable isolation
    };

    // Test different appendIfIsolated scenarios
    const scenarios = Object.values(APPEND_IF_ISOLATED_SCENARIOS);
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

    // Variable sleep based on scenario complexity (longer due to Serializable)
    const sleepTime = randomScenario.name.includes('Medium') || 
                     randomScenario.name.includes('Complex') ? 0.2 : 0.1;
    sleep(sleepTime);
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

    console.log('Setup completed - database cleaned and ready for appendIfIsolated benchmark');
}

// Teardown function
export function teardown(data) {
    console.log('AppendIfIsolated benchmark completed');
} 