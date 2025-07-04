import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const appendSuccessRate = new Rate('append_success');
const appendIfCounter = new Counter('append_if_operations');
const concurrencyErrorRate = new Rate('concurrency_errors');

// Test configuration - quick version for appendIf
export const options = {
  stages: [
    { duration: '5s', target: 5 },     // Warm-up: ramp up to 5 users
    { duration: '10s', target: 10 },   // Ramp up to 10 users
    { duration: '10s', target: 10 },   // Stay at 10 users
    { duration: '5s', target: 0 },     // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests should be below 1000ms
    errors: ['rate<0.10'],             // Error rate should be below 10%
    append_success: ['rate>0.95'],     // 95% of appends should succeed
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

// Quick appendIf scenarios (ReadSerializable isolation)
const APPEND_IF_QUICK_SCENARIOS = {
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
    }
};

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    // Test different appendIf scenarios
    const scenarios = Object.values(APPEND_IF_QUICK_SCENARIOS);
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

    // Short sleep for quick test
    sleep(0.1);
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

    console.log('Setup completed - database cleaned and ready for appendIf quick benchmark');
}

// Teardown function
export function teardown(data) {
    console.log('AppendIf quick benchmark completed');
} 