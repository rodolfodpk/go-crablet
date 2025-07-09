import http from 'k6/http';
import { check } from 'k6';

export const options = {
    vus: 1,
    iterations: 1000,
    maxDuration: '10m',
    gracefulStop: '30s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup function to validate basic functionality before running the quickiest test
export function setup() {
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '5s',
    };

    console.log('🧪 Validating basic functionality for quickiest test...');

    // Test 1: Health endpoint
    const healthRes = http.get(`${BASE_URL}/health`, params);
    if (healthRes.status !== 200) {
        throw new Error(`Health check failed: status ${healthRes.status}`);
    }

    // Test 2: Simple append
    const testEvent = {
        type: 'QuickiestTestEvent',
        data: JSON.stringify({ message: 'quickiest test validation' }),
        tags: ['test:quickiest', 'validation:test']
    };
    const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify({ events: testEvent }), params);
    
    if (appendRes.status !== 200) {
        throw new Error(`Append test failed: status ${appendRes.status} body: ${appendRes.body}`);
    }

    // Test 3: Read the event back
    const readPayload = {
        query: {
            items: [{ types: ['QuickiestTestEvent'], tags: ['test:quickiest'] }]
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

    console.log('✅ Basic functionality validated - proceeding with quickiest test');
}

export default function () {
    const response = http.get(`${BASE_URL}/health`);
    
    check(response, {
        'status is 200': (r) => r.status === 200,
        'body has ok': (r) => r.body.includes('ok'),
    });
} 