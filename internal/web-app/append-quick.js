import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Quick test configuration
export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<500'],
    errors: ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function generateEvent(eventType, tags) {
    return {
        type: eventType,
        data: JSON.stringify({ 
            timestamp: new Date().toISOString(),
            id: generateUniqueId('event')
        }),
        tags: tags.map(tag => `${tag}:${generateUniqueId(tag)}`)
    };
}

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    };

    // Test 1: Single event append
    const singleEvent = generateEvent('UserCreated', ['user', 'tenant']);
    const singleRes = http.post(
        `${BASE_URL}/append`,
        JSON.stringify({ events: singleEvent }),
        params
    );

    const singleSuccess = check(singleRes, {
        'single event append status is 200': (r) => r.status === 200,
    });

    if (!singleSuccess) {
        errorRate.add(1);
    }

    sleep(0.1);

    // Test 2: Batch event append
    const batchEvents = [
        generateEvent('AccountOpened', ['account', 'user']),
        generateEvent('AccountFunded', ['account', 'user']),
        generateEvent('TransactionInitiated', ['transaction', 'account'])
    ];
    
    const batchRes = http.post(
        `${BASE_URL}/append`,
        JSON.stringify({ events: batchEvents }),
        params
    );

    const batchSuccess = check(batchRes, {
        'batch event append status is 200': (r) => r.status === 200,
    });

    if (!batchSuccess) {
        errorRate.add(1);
    }

    sleep(0.1);

    // Test 3: Conditional append (should succeed)
    const conditionalEvent = generateEvent('UniqueEvent', ['unique', 'test']);
    const conditionalRes = http.post(
        `${BASE_URL}/append`,
        JSON.stringify({
            events: conditionalEvent,
            condition: {
                failIfEventsMatch: {
                    items: [{
                        types: ['UniqueEvent'],
                        tags: [`unique:${generateUniqueId('unique')}`]
                    }]
                }
            }
        }),
        params
    );

    const conditionalSuccess = check(conditionalRes, {
        'conditional append status is 200': (r) => r.status === 200,
    });

    if (!conditionalSuccess) {
        errorRate.add(1);
    }

    sleep(0.1);
}

export function setup() {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '5s',
    };

    console.log('🧪 Validating basic functionality for append quick test...');

    // Test 1: Health endpoint
    const healthRes = http.get(`${BASE_URL}/health`, params);
    if (healthRes.status !== 200) {
        throw new Error(`Health check failed: status ${healthRes.status}`);
    }

    // Test 2: Simple append
    const testEvent = {
        type: 'AppendQuickTestEvent',
        data: JSON.stringify({ message: 'append quick test validation' }),
        tags: ['test:appendquick', 'validation:test']
    };
    const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify({ events: testEvent }), params);
    
    if (appendRes.status !== 200) {
        throw new Error(`Append test failed: status ${appendRes.status} body: ${appendRes.body}`);
    }

    // Test 3: Read the event back
    const readPayload = {
        query: {
            items: [{ types: ['AppendQuickTestEvent'], tags: ['test:appendquick'] }]
        }
    };
    const readRes = http.post(`${BASE_URL}/read`, JSON.stringify(readPayload), params);
    
    if (readRes.status !== 200) {
        throw new Error(`Read test failed: status ${readRes.status} body: ${readRes.body}`);
    }

    const readBody = JSON.parse(readRes.body);
    if (!readBody || !('numberOfMatchingEvents' in readBody) || readBody.numberOfMatchingEvents < 1) {
        throw new Error(`Read test failed: did not return the event. Body: ${readRes.body}`);
    }

    // Test 4: Clean database
    const cleanupRes = http.post(`${BASE_URL}/cleanup`, null, params);
    if (cleanupRes.status !== 200) {
        throw new Error(`Cleanup failed: status ${cleanupRes.status}`);
    }

    console.log('✅ Basic functionality validated - proceeding with append quick test');
} 