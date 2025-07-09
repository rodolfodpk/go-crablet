import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const appendSuccessRate = new Rate('append_success');
const readCommittedCounter = new Counter('read_committed_appends');
const repeatableReadCounter = new Counter('repeatable_read_appends');
const serializableCounter = new Counter('serializable_appends');

// Test configuration - optimized for isolation level comparison
export const options = {
  stages: [
    { duration: '20s', target: 5 },    // Warm-up: ramp up to 5 users
    { duration: '30s', target: 5 },    // Stay at 5 users (warm-up)
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 10 },    // Stay at 10 users
    { duration: '30s', target: 20 },   // Ramp up to 20 users
    { duration: '1m', target: 20 },    // Stay at 20 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2000ms
    http_req_duration: ['p(99)<5000'], // 99% of requests should be below 5000ms
    errors: ['rate<0.10'],             // Error rate should be below 10%
    http_reqs: ['rate>50'],            // Should handle at least 50 req/s
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

// Isolation level test scenarios
const ISOLATION_SCENARIOS = {
    // Simple append (Read Committed)
    READ_COMMITTED_SIMPLE: {
        name: 'Read Committed - Simple Append',
        payload: () => ({
            events: generateUniqueEvent('UserCreated', ['user', 'tenant'])
        }),
        isolation: 'read_committed'
    },
    
    // Conditional append (Read Committed)
    READ_COMMITTED_CONDITIONAL: {
        name: 'Read Committed - Conditional Append',
        payload: () => ({
            events: generateUniqueEvent('UniqueEvent', ['unique', 'test']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['UniqueEvent'], ['unique'])
            }
        }),
        isolation: 'read_committed'
    },
    
    // Simple append (Repeatable Read)
    REPEATABLE_READ_SIMPLE: {
        name: 'Repeatable Read - Simple Append',
        payload: () => ({
            events: generateUniqueEvent('AccountOpened', ['account', 'user'])
        }),
        isolation: 'repeatable_read'
    },
    
    // Conditional append (Repeatable Read)
    REPEATABLE_READ_CONDITIONAL: {
        name: 'Repeatable Read - Conditional Append',
        payload: () => ({
            events: generateUniqueEvent('TransactionInitiated', ['transaction', 'account']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['TransactionInitiated'], ['transaction'])
            }
        }),
        isolation: 'repeatable_read'
    },
    
    // Simple append (Serializable)
    SERIALIZABLE_SIMPLE: {
        name: 'Serializable - Simple Append',
        payload: () => ({
            events: generateUniqueEvent('CriticalEvent', ['critical', 'system'])
        }),
        isolation: 'serializable'
    },
    
    // Conditional append (Serializable)
    SERIALIZABLE_CONDITIONAL: {
        name: 'Serializable - Conditional Append',
        payload: () => ({
            events: generateUniqueEvent('SecureEvent', ['secure', 'auth']),
            condition: {
                failIfEventsMatch: generateUniqueQuery(['SecureEvent'], ['secure'])
            }
        }),
        isolation: 'serializable'
    },
    
    // Batch append (Read Committed)
    READ_COMMITTED_BATCH: {
        name: 'Read Committed - Batch Append',
        payload: () => ({
            events: [
                generateUniqueEvent('OrderCreated', ['order', 'customer']),
                generateUniqueEvent('ItemAdded', ['order', 'item']),
                generateUniqueEvent('ItemAdded', ['order', 'item']),
                generateUniqueEvent('OrderValidated', ['order', 'system'])
            ]
        }),
        isolation: 'read_committed'
    },
    
    // Batch append (Repeatable Read)
    REPEATABLE_READ_BATCH: {
        name: 'Repeatable Read - Batch Append',
        payload: () => ({
            events: [
                generateUniqueEvent('TransferInitiated', ['transfer', 'from']),
                generateUniqueEvent('TransferValidated', ['transfer', 'system']),
                generateUniqueEvent('TransferCompleted', ['transfer', 'to'])
            ]
        }),
        isolation: 'repeatable_read'
    },
    
    // Batch append (Serializable)
    SERIALIZABLE_BATCH: {
        name: 'Serializable - Batch Append',
        payload: () => ({
            events: [
                generateUniqueEvent('AuditLog', ['audit', 'system']),
                generateUniqueEvent('SecurityEvent', ['security', 'system']),
                generateUniqueEvent('ComplianceCheck', ['compliance', 'system'])
            ]
        }),
        isolation: 'serializable'
    }
};

export default function () {
    // Test different isolation level scenarios
    const scenarios = Object.values(ISOLATION_SCENARIOS);
    const randomScenario = scenarios[Math.floor(Math.random() * scenarios.length)];
    
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-Isolation-Level': randomScenario.isolation,
        },
        timeout: '15s',
    };
    
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

    // Track metrics by isolation level
    if (success) {
        appendSuccessRate.add(1);
        
        switch (randomScenario.isolation) {
            case 'read_committed':
                readCommittedCounter.add(1);
                break;
            case 'repeatable_read':
                repeatableReadCounter.add(1);
                break;
            case 'serializable':
                serializableCounter.add(1);
                break;
        }
    } else {
        errorRate.add(1);
    }

    // Add some think time between requests
    sleep(0.1);
}

// Setup function to validate basic functionality
export function setup() {
    console.log('🧪 Validating basic functionality for isolation level benchmark...');
    
    // Test basic health check
    const healthResponse = http.get(`${BASE_URL}/health`);
    if (healthResponse.status !== 200) {
        throw new Error(`Health check failed: ${healthResponse.status}`);
    }
    
    // Test basic append functionality
    const testEvent = {
        events: {
            type: 'TestEvent',
            data: JSON.stringify({ test: true }),
            tags: ['test:validation']
        }
    };
    
    const appendResponse = http.post(
        `${BASE_URL}/append`,
        JSON.stringify(testEvent),
        { headers: { 'Content-Type': 'application/json' } }
    );
    
    if (appendResponse.status !== 200) {
        throw new Error(`Basic append test failed: ${appendResponse.status}`);
    }
    
    console.log('✅ Basic functionality validated - proceeding with isolation level benchmark');
    
    return { startTime: new Date().toISOString() };
}

// Teardown function
export function teardown(data) {
    console.log(`🏁 Isolation level benchmark completed. Started at: ${data.startTime}`);
    console.log('📊 Check the metrics for isolation level performance comparison:');
    console.log('   - Read Committed vs Repeatable Read vs Serializable');
    console.log('   - Simple vs Conditional vs Batch operations');
    console.log('   - Throughput and latency differences');
} 