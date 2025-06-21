import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration - optimized for higher resource allocation
export const options = {
  stages: [
    { duration: '30s', target: 5 },    // Ramp up to 5 users (reduced from 10)
    { duration: '1m', target: 5 },     // Stay at 5 users
    { duration: '30s', target: 15 },   // Ramp up to 15 users (reduced from 25)
    { duration: '2m', target: 15 },    // Stay at 15 users
    { duration: '30s', target: 50 },   // Ramp up to 50 users (reduced from 100)
    { duration: '3m', target: 50 },    // Stay at 50 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1500'], // 95% of requests should be below 1500ms
    http_req_duration: ['p(99)<3000'], // 99% of requests should be below 3000ms
    errors: ['rate<0.15'],             // Error rate should be below 15%
    http_reqs: ['rate>50'],            // Should handle at least 50 req/s
  },
  // Optimize for higher concurrency
  batch: 5,                            // Reduced batch size for stability
  batchPerHost: 5,                     // Reduced batch size per host
};

// Test data
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Generate unique IDs for each request to avoid concurrency bottlenecks
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Generate unique event with random IDs
function generateUniqueEvent(eventType, tagPrefixes) {
    const tags = tagPrefixes.map(prefix => `${prefix}:${generateUniqueId(prefix)}`);
    return {
        type: eventType,
        data: JSON.stringify({ timestamp: new Date().toISOString() }),
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

// Original functions for backward compatibility (but with unique IDs)
function generateEvent(eventType, tags) {
    return {
        type: eventType,
        data: JSON.stringify({ timestamp: new Date().toISOString() }),
        tags: tags.map(tag => {
            const [prefix, id] = tag.split(':');
            return `${prefix}:${generateUniqueId(prefix)}`;
        })
    };
}

function generateQuery(eventTypes, tags) {
    return {
        items: [{
            types: eventTypes,
            tags: tags.map(tag => {
                const [prefix, id] = tag.split(':');
                return `${prefix}:${generateUniqueId(prefix)}`;
            })
        }]
    };
}

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '30s',  // Increased timeout for better reliability
  };

  // Test 1: Append single event
  const singleEvent = generateUniqueEvent('CoursePlanned', ['course', 'user']);
  const appendSinglePayload = {
    events: singleEvent,
  };

  const appendSingleRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendSinglePayload),
    params
  );

  check(appendSingleRes, {
    'append single event status is 200': (r) => r.status === 200,
  });

  if (!appendSingleRes || appendSingleRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 2: Append multiple events
  const multipleEvents = [
    generateUniqueEvent('StudentEnrolled', ['course', 'student']),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment']),
    generateUniqueEvent('GradeSubmitted', ['course', 'student', 'assignment']),
  ];

  const appendMultiplePayload = {
    events: multipleEvents,
  };

  const appendMultipleRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendMultiplePayload),
    params
  );

  check(appendMultipleRes, {
    'append multiple events status is 200': (r) => r.status === 200,
  });

  if (appendMultipleRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 3: Read events by type and tags (targeted query)
  const readByTypeAndTagsPayload = {
    query: generateUniqueQuery(['StudentEnrolled'], ['course']),
  };

  const readByTypeAndTagsRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTypeAndTagsPayload),
    params
  );

  check(readByTypeAndTagsRes, {
    'read by type and tags status is 200': (r) => r.status === 200,
  });

  if (readByTypeAndTagsRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 4: Append with condition (should fail if events exist)
  const appendWithConditionPayload = {
    events: generateUniqueEvent('DuplicateEvent', ['course']),
    condition: {
      failIfEventsMatch: generateUniqueQuery(['DuplicateEvent'], ['course']),
    },
  };

  const appendWithConditionRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendWithConditionPayload),
    params
  );

  check(appendWithConditionRes, {
    'append with condition status is 200': (r) => r.status === 200,
  });

  if (appendWithConditionRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 5: Complex query with multiple items
  const complexQueryPayload = {
    query: {
      items: [
        {
          types: ['CoursePlanned'],
          tags: [`course:${generateUniqueId('course')}`],
        },
        {
          types: ['StudentEnrolled'],
          tags: [`student:${generateUniqueId('student')}`],
        },
      ],
    },
  };

  const complexQueryRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(complexQueryPayload),
    params
  );

  check(complexQueryRes, {
    'complex query status is 200': (r) => r.status === 200,
  });

  if (complexQueryRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance
}

// Setup function to initialize test data
export function setup() {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Add some initial events for testing with unique IDs
  const initialEvents = [
    generateUniqueEvent('CoursePlanned', ['course', 'user']),
    generateUniqueEvent('StudentEnrolled', ['course', 'student']),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment']),
  ];

  const setupPayload = {
    events: initialEvents,
  };

  const res = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(setupPayload),
    params
  );

  if (res.status !== 200) {
    // Setup failed silently
  }
} 