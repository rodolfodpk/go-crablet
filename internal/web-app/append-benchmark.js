import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const appendSuccessRate = new Rate('append_success');
const batchAppendCounter = new Counter('batch_appends');
const conditionalAppendCounter = new Counter('conditional_appends');

// Test configuration - optimized for append scenarios
export const options = {
  stages: [
    { duration: '20s', target: 10 },   // Warm-up: ramp up to 10 users
    { duration: '30s', target: 10 },   // Stay at 10 users (warm-up)
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 50 },    // Stay at 50 users
    { duration: '30s', target: 100 },  // Ramp up to 100 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 200 },  // Ramp up to 200 users
    { duration: '1m', target: 200 },   // Stay at 200 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests should be below 1000ms
    http_req_duration: ['p(99)<2000'], // 99% of requests should be below 2000ms
    errors: ['rate<0.10'],             // Error rate should be below 10%
    http_reqs: ['rate>100'],           // Should handle at least 100 req/s
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

// Different append scenarios
const APPEND_SCENARIOS = {
    // Single event appends
    SINGLE_EVENT: {
        name: 'Single Event Append',
        payload: () => ({
            events: generateUniqueEvent('UserCreated', ['user', 'tenant'])
        })
    },
    
    // Small batch appends (2-3 events)
    SMALL_BATCH: {
        name: 'Small Batch Append',
        payload: () => ({
            events: [
                generateUniqueEvent('AccountOpened', ['account', 'user']),
                generateUniqueEvent('AccountFunded', ['account', 'user'])
            ]
        })
    },
    
    // Medium batch appends (5-10 events)
    MEDIUM_BATCH: {
        name: 'Medium Batch Append',
        payload: () => ({
            events: [
                generateUniqueEvent('OrderCreated', ['order', 'customer']),
                generateUniqueEvent('ItemAdded', ['order', 'item']),
                generateUniqueEvent('ItemAdded', ['order', 'item']),
                generateUniqueEvent('ItemAdded', ['order', 'item']),
                generateUniqueEvent('OrderValidated', ['order', 'system'])
            ]
        })
    },
    
    // Large batch appends (20+ events)
    LARGE_BATCH: {
        name: 'Large Batch Append',
        payload: () => ({
            events: Array.from({ length: 25 }, (_, i) => 
                generateUniqueEvent('LogEntry', ['service', 'level', 'trace'])
            )
        })
    },
    
    // Conditional append (should succeed)
    CONDITIONAL_SUCCESS: {
        name: 'Conditional Append (Success)',
        payload: () => ({
            events: generateUniqueEvent('UniqueEvent', ['unique', 'test']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['UniqueEvent'], ['unique'])
            }
        })
    },
    
    // Conditional append (should fail)
    CONDITIONAL_FAIL: {
        name: 'Conditional Append (Fail)',
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
    
    // Mixed event types
    MIXED_EVENTS: {
        name: 'Mixed Event Types',
        payload: () => ({
            events: [
                generateUniqueEvent('UserCreated', ['user', 'tenant']),
                generateUniqueEvent('AccountOpened', ['account', 'user']),
                generateUniqueEvent('TransactionInitiated', ['transaction', 'account']),
                generateUniqueEvent('NotificationSent', ['notification', 'user']),
                generateUniqueEvent('AuditLog', ['audit', 'system'])
            ]
        })
    },
    
    // High-frequency events
    HIGH_FREQUENCY: {
        name: 'High Frequency Events',
        payload: () => ({
            events: Array.from({ length: 50 }, (_, i) => 
                generateUniqueEvent('SensorReading', ['sensor', 'location', 'type'])
            )
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

    // Test different append scenarios
    const scenarios = Object.values(APPEND_SCENARIOS);
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
        
        // Track specific scenario metrics
        if (randomScenario.name.includes('Batch')) {
            batchAppendCounter.add(1);
        }
        if (randomScenario.name.includes('Conditional')) {
            conditionalAppendCounter.add(1);
        }
    } else {
        errorRate.add(1);
    }

    // Variable sleep based on scenario complexity
    const sleepTime = randomScenario.name.includes('Large') || 
                     randomScenario.name.includes('High Frequency') ? 0.1 : 0.05;
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

    console.log('Setup completed - database cleaned and ready for append benchmark');
}

// Teardown function
export function teardown(data) {
    console.log('Append benchmark completed');
} 