import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const projectSuccessRate = new Rate('project_success');
const projectCounter = new Counter('project_operations');
const singleProjectorCounter = new Counter('single_projector');
const multipleProjectorCounter = new Counter('multiple_projector');

// Test configuration - optimized for projection scenarios
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
    project_success: ['rate>0.95'],    // 95% of projections should succeed
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

// Projection scenarios
const PROJECT_SCENARIOS = {
    // Single projector (counter)
    SINGLE_PROJECTOR: {
        name: 'Single Projector (Counter)',
        payload: () => ({
            projectors: [{
                id: `counter_${generateUniqueId('counter')}`,
                query: generateUniqueQuery(['UserCreated', 'AccountOpened'], ['user', 'tenant']),
                initialState: 0,
                transitionFunction: 'function(state, event) { return state + 1; }'
            }]
        })
    },
    
    // Multiple projectors (counter + sum)
    MULTIPLE_PROJECTORS: {
        name: 'Multiple Projectors (Counter + Sum)',
        payload: () => ({
            projectors: [
                {
                    id: `counter_${generateUniqueId('counter')}`,
                    query: generateUniqueQuery(['UserCreated'], ['user']),
                    initialState: 0,
                    transitionFunction: 'function(state, event) { return state + 1; }'
                },
                {
                    id: `sum_${generateUniqueId('sum')}`,
                    query: generateUniqueQuery(['TransactionInitiated'], ['account']),
                    initialState: 0,
                    transitionFunction: 'function(state, event) { return state + 1; }'
                }
            ]
        })
    },
    
    // Complex projector with multiple event types
    COMPLEX_PROJECTOR: {
        name: 'Complex Projector (Multiple Types)',
        payload: () => ({
            projectors: [{
                id: `complex_${generateUniqueId('complex')}`,
                query: {
                    items: [
                        {
                            types: ['UserCreated', 'AccountOpened'],
                            tags: ['user']
                        },
                        {
                            types: ['TransactionInitiated', 'BalanceChecked'],
                            tags: ['account']
                        }
                    ]
                },
                initialState: { users: 0, accounts: 0, transactions: 0 },
                transitionFunction: 'function(state, event) { state.users++; return state; }'
            }]
        })
    },
    
    // Projection with cursor (from specific position)
    PROJECTION_WITH_CURSOR: {
        name: 'Projection with Cursor',
        payload: () => ({
            projectors: [{
                id: `cursor_${generateUniqueId('cursor')}`,
                query: generateUniqueQuery(['LogEntry'], ['service']),
                initialState: 0,
                transitionFunction: 'function(state, event) { return state + 1; }'
            }],
            after: '1000' // Start from position 1000
        })
    },
    
    // High-frequency projection
    HIGH_FREQUENCY_PROJECTION: {
        name: 'High Frequency Projection',
        payload: () => ({
            projectors: [{
                id: `high_freq_${generateUniqueId('highfreq')}`,
                query: generateUniqueQuery(['SensorReading', 'LogEntry'], ['sensor', 'service']),
                initialState: 0,
                transitionFunction: 'function(state, event) { return state + 1; }'
            }]
        })
    },
    
    // Multiple complex projectors
    MULTIPLE_COMPLEX_PROJECTORS: {
        name: 'Multiple Complex Projectors',
        payload: () => ({
            projectors: [
                {
                    id: `user_stats_${generateUniqueId('user')}`,
                    query: generateUniqueQuery(['UserCreated', 'UserUpdated'], ['user']),
                    initialState: { created: 0, updated: 0 },
                    transitionFunction: 'function(state, event) { if(event.type === "UserCreated") state.created++; else state.updated++; return state; }'
                },
                {
                    id: `order_stats_${generateUniqueId('order')}`,
                    query: generateUniqueQuery(['OrderCreated', 'OrderCompleted'], ['order']),
                    initialState: { created: 0, completed: 0 },
                    transitionFunction: 'function(state, event) { if(event.type === "OrderCreated") state.created++; else state.completed++; return state; }'
                },
                {
                    id: `transaction_stats_${generateUniqueId('transaction')}`,
                    query: generateUniqueQuery(['TransactionInitiated', 'TransactionCompleted'], ['transaction']),
                    initialState: { initiated: 0, completed: 0 },
                    transitionFunction: 'function(state, event) { if(event.type === "TransactionInitiated") state.initiated++; else state.completed++; return state; }'
                }
            ]
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

    // Test different projection scenarios
    const scenarios = Object.values(PROJECT_SCENARIOS);
    const randomScenario = scenarios[Math.floor(Math.random() * scenarios.length)];
    
    const payload = randomScenario.payload();
    const response = http.post(
        `${BASE_URL}/project`,
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
                       body.hasOwnProperty('states');
            } catch {
                return false;
            }
        }
    });

    // Track metrics
    if (success) {
        projectSuccessRate.add(1);
        projectCounter.add(1);
        
        // Track specific scenario metrics
        if (randomScenario.name.includes('Single')) {
            singleProjectorCounter.add(1);
        }
        if (randomScenario.name.includes('Multiple')) {
            multipleProjectorCounter.add(1);
        }
    } else {
        errorRate.add(1);
    }

    // Variable sleep based on scenario complexity
    const sleepTime = randomScenario.name.includes('Complex') || 
                     randomScenario.name.includes('Multiple') ? 0.1 : 0.05;
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

    // Test basic projection functionality
    const testPayload = {
        projectors: [{
            id: 'test_projector',
            query: {
                items: [{
                    types: ['TestEvent'],
                    tags: ['test:setup']
                }]
            },
            initialState: 0,
            transitionFunction: 'function(state, event) { return state + 1; }'
        }]
    };

    const response = http.post(
        `${BASE_URL}/project`,
        JSON.stringify(testPayload),
        params
    );

    if (response.status !== 200) {
        throw new Error(`Setup failed: ${response.status} - ${response.body}`);
    }

    console.log('Projection endpoint setup completed successfully');
    return { baseUrl: BASE_URL };
}

// Teardown function to clean up after benchmark
export function teardown(data) {
    console.log('Projection benchmark completed');
} 