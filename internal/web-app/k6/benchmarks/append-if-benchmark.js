import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const appendSuccessRate = new Rate('append_success');
const appendIfCounter = new Counter('append_if_operations');
const concurrencyErrorRate = new Rate('concurrency_errors');

// Test configuration - optimized for appendIf scenarios
export const options = {
  stages: [
    { duration: '20s', target: 10 },   // Warm-up: ramp up to 10 users
    { duration: '30s', target: 10 },   // Stay at 10 users (warm-up)
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 50 },    // Stay at 50 users
    { duration: '30s', target: 100 },  // Ramp up to 100 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests should be below 1000ms
    http_req_duration: ['p(99)<2000'], // 99% of requests should be below 2000ms
    errors: ['rate<0.10'],             // Error rate should be below 10%
    http_reqs: ['rate>100'],           // Should handle at least 100 req/s
    append_success: ['rate>0.95'],     // 95% of appends should succeed
    concurrency_errors: ['rate<0.05'], // Concurrency errors should be below 5%
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

// AppendIf specific scenarios (ReadSerializable isolation)
const APPEND_IF_SCENARIOS = {
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
    
    // Batch with condition (should succeed)
    BATCH_WITH_CONDITION: {
        name: 'Batch with Condition (Success)',
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
    
    // High concurrency scenario
    HIGH_CONCURRENCY: {
        name: 'High Concurrency Test',
        payload: () => ({
            events: generateUniqueEvent('ConcurrentEvent', ['concurrent', 'test']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['ConcurrentEvent'], ['concurrent'])
            }
        })
    },
    
    // Large batch with condition
    LARGE_BATCH_WITH_CONDITION: {
        name: 'Large Batch with Condition',
        payload: () => ({
            events: Array.from({ length: 20 }, (_, i) => 
                generateUniqueEvent('LogEntry', ['service', 'level', 'trace'])
            ),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['LogEntry'], ['service'])
            }
        })
    }
};

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '15s',
    };

    // Test different appendIf scenarios
    const scenarios = Object.values(APPEND_IF_SCENARIOS);
    const randomScenario = scenarios[Math.floor(Math.random() * scenarios.length)];
    
    const payload = randomScenario.payload();
    const response = http.post(
        `${BASE_URL}/append`,
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
        appendSuccessRate.add(1);
        appendIfCounter.add(1);
        
        // Check if condition failed
        try {
            const body = JSON.parse(response.body);
            if (body.appendConditionFailed) {
                concurrencyErrorRate.add(1);
            }
        } catch {
            // Ignore parsing errors
        }
    } else {
        errorRate.add(1);
    }

    // Variable sleep based on scenario complexity
    const sleepTime = randomScenario.name.includes('Large') ? 0.1 : 0.05;
    sleep(sleepTime);
}

// Setup function to validate basic functionality and appendIf conditions before running the benchmark
export function setup() {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    console.log('🧪 Validating basic functionality and appendIf conditions for appendIf benchmark...');

    // Test 1: Health endpoint
    const healthRes = http.get(`${BASE_URL}/health`, params);
    if (healthRes.status !== 200) {
        throw new Error(`Health check failed: status ${healthRes.status}`);
    }

    // Test 2: Simple append
    const testEvent = {
        type: 'AppendIfBenchmarkTestEvent',
        data: JSON.stringify({ message: 'appendIf benchmark test' }),
        tags: ['test:appendif', 'benchmark:validation']
    };
    const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify({ events: testEvent }), params);
    
    if (appendRes.status !== 200) {
        throw new Error(`Append test failed: status ${appendRes.status} body: ${appendRes.body}`);
    }

    // Test 3: AppendIf with condition that should succeed
    const appendIfEvent = {
        type: 'AppendIfBenchmarkTestEvent2',
        data: JSON.stringify({ message: 'appendIf condition test' }),
        tags: ['test:appendif', 'condition:success']
    };
    const appendIfPayload = {
        events: appendIfEvent,
        condition: {
            failIfEventsMatch: {
                items: [{ types: ['NonExistentEvent'], tags: ['test:appendif'] }]
            }
        }
    };
    const appendIfRes = http.post(`${BASE_URL}/append`, JSON.stringify(appendIfPayload), params);
    
    if (appendIfRes.status !== 200) {
        throw new Error(`AppendIf test failed: status ${appendIfRes.status} body: ${appendIfRes.body}`);
    }

    // Test 4: AppendIf with condition that should fail
    const appendIfFailEvent = {
        type: 'AppendIfBenchmarkTestEvent3',
        data: JSON.stringify({ message: 'appendIf condition fail test' }),
        tags: ['test:appendif', 'condition:fail']
    };
    const appendIfFailPayload = {
        events: appendIfFailEvent,
        condition: {
            failIfEventsMatch: {
                items: [{ types: ['AppendIfBenchmarkTestEvent'], tags: ['test:appendif'] }]
            }
        }
    };
    const appendIfFailRes = http.post(`${BASE_URL}/append`, JSON.stringify(appendIfFailPayload), params);
    
    if (appendIfFailRes.status !== 200) {
        throw new Error(`AppendIf fail test failed: status ${appendIfFailRes.status} body: ${appendIfFailRes.body}`);
    }

    // Test 5: Read events back
    const readPayload = {
        query: {
            items: [{ types: ['AppendIfBenchmarkTestEvent', 'AppendIfBenchmarkTestEvent2'], tags: ['test:appendif'] }]
        }
    };
    const readRes = http.post(`${BASE_URL}/read`, JSON.stringify(readPayload), params);
    
    if (readRes.status !== 200) {
        throw new Error(`Read test failed: status ${readRes.status} body: ${readRes.body}`);
    }

    const readBody = JSON.parse(readRes.body);
    if (!readBody || !('numberOfMatchingEvents' in readBody) || readBody.numberOfMatchingEvents < 2) {
        throw new Error(`Read test failed: did not return expected events. Body: ${readRes.body}`);
    }

    // Test 6: Clean up database before benchmark
    const cleanupRes = http.post(`${BASE_URL}/cleanup`, null, params);
    if (cleanupRes.status !== 200) {
        throw new Error(`Cleanup failed: status ${cleanupRes.status}`);
    }

    console.log('✅ Basic functionality and appendIf conditions validated - proceeding with appendIf benchmark');
}

// Teardown function
export function teardown(data) {
    console.log('AppendIf benchmark completed');
} 