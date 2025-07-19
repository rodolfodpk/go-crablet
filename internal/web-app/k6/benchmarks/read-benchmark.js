import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const readSuccessRate = new Rate('read_success');
const readCounter = new Counter('read_operations');
const simpleQueryCounter = new Counter('simple_query');
const complexQueryCounter = new Counter('complex_query');

// Test configuration - optimized for read scenarios
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
    http_req_duration: ['p(95)<500'],  // 95% of requests should be below 500ms
    http_req_duration: ['p(99)<1000'], // 99% of requests should be below 1000ms
    errors: ['rate<0.10'],             // Error rate should be below 10%
    http_reqs: ['rate>200'],           // Should handle at least 200 req/s
    read_success: ['rate>0.95'],       // 95% of reads should succeed
  },
};

// Test data
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Generate unique IDs for each request
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
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

// Read scenarios
const READ_SCENARIOS = {
    // Simple single event type query
    SIMPLE_QUERY: {
        name: 'Simple Query (Single Type)',
        payload: () => ({
            query: generateUniqueQuery(['UserCreated'], ['user'])
        })
    },
    
    // Multiple event types query
    MULTIPLE_TYPES: {
        name: 'Multiple Event Types Query',
        payload: () => ({
            query: generateUniqueQuery(['UserCreated', 'AccountOpened', 'TransactionInitiated'], ['user', 'account'])
        })
    },
    
    // Complex query with multiple conditions
    COMPLEX_QUERY: {
        name: 'Complex Query (Multiple Conditions)',
        payload: () => ({
            query: {
                items: [
                    {
                        types: ['UserCreated', 'UserUpdated'],
                        tags: ['user']
                    },
                    {
                        types: ['AccountOpened', 'AccountClosed'],
                        tags: ['account']
                    }
                ]
            }
        })
    },
    
    // High-frequency event query
    HIGH_FREQUENCY_QUERY: {
        name: 'High Frequency Event Query',
        payload: () => ({
            query: generateUniqueQuery(['LogEntry', 'SensorReading', 'AuditLog'], ['service', 'sensor', 'audit'])
        })
    },
    
    // Specific tag query
    SPECIFIC_TAG_QUERY: {
        name: 'Specific Tag Query',
        payload: () => ({
            query: {
                items: [{
                    types: ['OrderCreated', 'OrderCompleted'],
                    tags: [`order:${generateUniqueId('order')}`]
                }]
            }
        })
    },
    
    // Mixed query with various event types
    MIXED_QUERY: {
        name: 'Mixed Query (Various Types)',
        payload: () => ({
            query: {
                items: [
                    {
                        types: ['UserCreated'],
                        tags: ['user']
                    },
                    {
                        types: ['TransactionInitiated', 'TransactionCompleted'],
                        tags: ['transaction']
                    },
                    {
                        types: ['NotificationSent'],
                        tags: ['notification']
                    }
                ]
            }
        })
    },
    
    // Empty query (should return all events)
    EMPTY_QUERY: {
        name: 'Empty Query (All Events)',
        payload: () => ({
            query: {
                items: []
            }
        })
    },
    
    // Single event type with multiple tags
    SINGLE_TYPE_MULTIPLE_TAGS: {
        name: 'Single Type Multiple Tags',
        payload: () => ({
            query: {
                items: [{
                    types: ['LogEntry'],
                    tags: ['service', 'level', 'environment']
                }]
            }
        })
    }
};

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    // Test different read scenarios
    const scenarios = Object.values(READ_SCENARIOS);
    const randomScenario = scenarios[Math.floor(Math.random() * scenarios.length)];
    
    const payload = randomScenario.payload();
    const response = http.post(
        `${BASE_URL}/read`,
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
                       body.hasOwnProperty('numberOfMatchingEvents');
            } catch {
                return false;
            }
        }
    });

    // Track metrics
    if (success) {
        readSuccessRate.add(1);
        readCounter.add(1);
        
        // Track specific scenario metrics
        if (randomScenario.name.includes('Simple')) {
            simpleQueryCounter.add(1);
        }
        if (randomScenario.name.includes('Complex') || randomScenario.name.includes('Multiple')) {
            complexQueryCounter.add(1);
        }
    } else {
        errorRate.add(1);
    }

    // Variable sleep based on scenario complexity
    const sleepTime = randomScenario.name.includes('Complex') || 
                     randomScenario.name.includes('Multiple') ? 0.05 : 0.02;
    sleep(sleepTime);
}

// Setup function to validate basic functionality before running the benchmark
export function setup() {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    // Test basic read functionality
    const testPayload = {
        query: {
            items: [{
                types: ['TestEvent'],
                tags: ['test:setup']
            }]
        }
    };

    const response = http.post(
        `${BASE_URL}/read`,
        JSON.stringify(testPayload),
        params
    );

    if (response.status !== 200) {
        throw new Error(`Setup failed: ${response.status} - ${response.body}`);
    }

    console.log('Read endpoint setup completed successfully');
    return { baseUrl: BASE_URL };
}

// Teardown function to clean up after benchmark
export function teardown(data) {
    console.log('Read benchmark completed');
} 