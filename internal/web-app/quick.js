import http from 'k6/http';
import { check } from 'k6';

// Very short test configuration - optimized for speed
export const options = {
  vus: 2,  // Increased from 1 for better throughput
  duration: '10s',
  // Optimize for higher throughput
  batch: 10,
  batchPerHost: 10,
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup function to validate basic functionality before running the quick test
export function setup() {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '5s',
  };

  console.log('🧪 Validating basic functionality for quick test...');

  // Test 1: Health endpoint
  const healthRes = http.get(`${BASE_URL}/health`, params);
  if (healthRes.status !== 200) {
    throw new Error(`Health check failed: status ${healthRes.status}`);
  }

  // Test 2: Simple append
  const testEvent = {
    type: 'QuickTestEvent',
    data: JSON.stringify({ message: 'quick test validation' }),
    tags: ['test:quick', 'validation:test']
  };
  const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify({ events: testEvent }), params);
  
  if (appendRes.status !== 200) {
    throw new Error(`Append test failed: status ${appendRes.status} body: ${appendRes.body}`);
  }

  // Test 3: Read the event back
  const readPayload = {
    query: {
      items: [{ types: ['QuickTestEvent'], tags: ['test:quick'] }]
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

  console.log('✅ Basic functionality validated - proceeding with quick test');
}

// Generate unique IDs for each request to avoid concurrency bottlenecks
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',  // Reduced timeout for faster failure detection
  };

  // Generate unique IDs for this iteration
  const testId = generateUniqueId('test');
  const quickId = generateUniqueId('quick');

  // Test 1: Append single event
  const singleEvent = {
    type: 'TestEvent',
    data: JSON.stringify({
      id: testId,
      message: 'Hello World',
    }),
    tags: [`test:${testId}`, `quick:${quickId}`],
  };

  const appendRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify({ events: singleEvent }),
    params
  );

  check(appendRes, {
    'append status is 200': (r) => r.status === 200,
    'append has duration': (r) => r.json('durationInMicroseconds') > 0,
  });

  // Test 2: Read events
  const readPayload = {
    query: {
      items: [
        {
          types: ['TestEvent'],
          tags: [`test:${testId}`],
        },
      ],
    },
  };

  const readRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readPayload),
    params
  );

  check(readRes, {
    'read status is 200': (r) => r.status === 200,
    'read has duration': (r) => r.json('durationInMicroseconds') > 0,
    'read has event count': (r) => r.json('numberOfMatchingEvents') >= 0,
  });
} 