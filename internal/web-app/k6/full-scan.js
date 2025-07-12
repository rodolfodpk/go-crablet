import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Base URL for the web app
const BASE_URL = 'http://localhost:8080';

// Test configuration for full-scan scenarios (empty tags)
export const options = {
  stages: [
    { duration: '30s', target: 5 },    // Ramp up to 5 users
    { duration: '1m', target: 5 },     // Stay at 5 users
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '2m', target: 10 },    // Stay at 10 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2000ms
    http_req_duration: ['p(99)<4000'], // 99% of requests should be below 4000ms
    errors: ['rate<0.20'],             // Error rate should be below 20%
  },
};

// Generate unique IDs for each request to avoid concurrency bottlenecks
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
            message: 'full-scan test event from k6',
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

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Test 1: Read events by type only (empty tags - full scan)
  const readByTypeOnlyPayload = {
    query: generateUniqueQuery(['CoursePlanned', 'StudentEnrolled'], []),
  };

  const readByTypeOnlyRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTypeOnlyPayload),
    params
  );

  check(readByTypeOnlyRes, {
    'read by type only status is 200': (r) => r.status === 200,
  });

  if (readByTypeOnlyRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 2: Read events by tag only (empty types - full scan)
  const readByTagOnlyPayload = {
    query: generateUniqueQuery([], ['course']),
  };

  const readByTagOnlyRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTagOnlyPayload),
    params
  );

  check(readByTagOnlyRes, {
    'read by tag only status is 200': (r) => r.status === 200,
  });

  if (readByTagOnlyRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 3: Read all events (empty types and tags - full table scan)
  const readAllEventsPayload = {
    query: generateUniqueQuery([], []),
  };

  const readAllEventsRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readAllEventsPayload),
    params
  );

  check(readAllEventsRes, {
    'read all events status is 200': (r) => r.status === 200,
  });

  if (readAllEventsRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 4: Complex query with multiple items, some with empty filters
  const complexFullScanPayload = {
    query: {
      items: [
        {
          types: ['CoursePlanned'],
          tags: [] // Empty tags - will scan all CoursePlanned events
        },
        {
          types: [], // Empty types - will scan all events with student tag
          tags: [`student:${generateUniqueId('student')}`]
        },
        {
          types: ['StudentEnrolled'],
          tags: [] // Empty tags - will scan all StudentEnrolled events
        }
      ]
    }
  };

  const complexFullScanRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(complexFullScanPayload),
    params
  );

  check(complexFullScanRes, {
    'complex full scan status is 200': (r) => r.status === 200,
  });

  if (complexFullScanRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 5: Append events for full-scan testing
  const fullScanEvents = [
    generateUniqueEvent('FullScanEvent1', ['fullscan', 'test']),
    generateUniqueEvent('FullScanEvent2', ['fullscan', 'test']),
    generateUniqueEvent('FullScanEvent3', ['fullscan', 'test']),
  ];

  const appendFullScanPayload = {
    events: fullScanEvents,
  };

  const appendFullScanRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendFullScanPayload),
    params
  );

  check(appendFullScanRes, {
    'append full scan events status is 200': (r) => r.status === 200,
  });

  if (appendFullScanRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);
}

// Setup function to validate basic functionality before running the full-scan test
export function setup() {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  console.log('🧪 Validating basic functionality for full-scan test...');

  // Test 1: Health endpoint
  const healthRes = http.get(`${BASE_URL}/health`, params);
  if (healthRes.status !== 200) {
    throw new Error(`Health check failed: status ${healthRes.status}`);
  }

  // Test 2: Simple append
  const testEvent = {
    type: 'FullScanTestEvent',
    data: JSON.stringify({ message: 'full-scan test validation' }),
    tags: ['test:fullscan', 'validation:test']
  };
  const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify({ events: testEvent }), params);
  
  if (appendRes.status !== 200) {
    throw new Error(`Append test failed: status ${appendRes.status} body: ${appendRes.body}`);
  }

  // Test 3: Read the event back
  const readPayload = {
    query: {
      items: [{ types: ['FullScanTestEvent'], tags: ['test:fullscan'] }]
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

  // Test 4: Full scan read (empty query)
  const fullScanPayload = {
    query: {
      items: [{ types: [], tags: [] }]
    }
  };
  const fullScanRes = http.post(`${BASE_URL}/read`, JSON.stringify(fullScanPayload), params);
  
  if (fullScanRes.status !== 200) {
    throw new Error(`Full scan test failed: status ${fullScanRes.status} body: ${fullScanRes.body}`);
  }

  console.log('✅ Basic functionality validated - proceeding with full-scan test');
} 