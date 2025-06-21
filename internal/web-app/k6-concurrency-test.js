import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conflictRate = new Rate('conflicts');

// Test configuration - designed to create contention
export const options = {
  stages: [
    { duration: '10s', target: 5 },    // Ramp up to 5 users
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 20 },    // Ramp up to 20 users (high contention)
    { duration: '2m', target: 20 },    // Stay at 20 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2000ms
    errors: ['rate<0.30'],             // Error rate should be below 30% (expecting some conflicts)
    conflicts: ['rate>0.05'],          // Should see some conflicts (optimistic locking working)
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use fixed tag values to create contention
const CONTENTION_TAGS = {
  course: 'contention-course-123',
  student: 'contention-student-456',
  assignment: 'contention-assignment-789',
  user: 'contention-user-101',
};

// Generate event with contention tags
function generateContentionEvent(eventType, tagKeys) {
    const tags = tagKeys.map(key => `${key}:${CONTENTION_TAGS[key]}`);
    return {
        type: eventType,
        data: JSON.stringify({ 
            timestamp: new Date().toISOString(),
            message: `contention test event from VU ${__VU}`,
            iteration: __ITER,
            vu: __VU
        }),
        tags: tags
    };
}

// Generate query with contention tags
function generateContentionQuery(eventTypes, tagKeys) {
    const tags = tagKeys.map(key => `${key}:${CONTENTION_TAGS[key]}`);
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
    timeout: '30s',
  };

  // Test 1: Append single event with contention
  const singleEvent = generateContentionEvent('ContentionEvent', ['course', 'user']);
  const appendSinglePayload = {
    events: singleEvent,
  };

  const appendSingleRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendSinglePayload),
    params
  );

  check(appendSingleRes, {
    'append single event status is 200 or 409': (r) => r.status === 200 || r.status === 409,
    'append single event duration < 500ms': (r) => r.timings.duration < 500,
  });

  if (appendSingleRes.status === 409) {
    conflictRate.add(1);
  } else if (appendSingleRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 2: Append multiple events with contention
  const multipleEvents = [
    generateContentionEvent('StudentEnrolled', ['course', 'student']),
    generateContentionEvent('AssignmentCreated', ['course', 'assignment']),
    generateContentionEvent('GradeSubmitted', ['course', 'student', 'assignment']),
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
    'append multiple events status is 200 or 409': (r) => r.status === 200 || r.status === 409,
    'append multiple events duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (appendMultipleRes.status === 409) {
    conflictRate.add(1);
  } else if (appendMultipleRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 3: Read events by type (should succeed)
  const readByTypePayload = {
    query: generateContentionQuery(['ContentionEvent', 'StudentEnrolled'], []),
  };

  const readByTypeRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTypePayload),
    params
  );

  check(readByTypeRes, {
    'read by type status is 200': (r) => r.status === 200,
    'read by type duration < 300ms': (r) => r.timings.duration < 300,
  });

  if (readByTypeRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 4: Read events by tags (should succeed)
  const readByTagsPayload = {
    query: generateContentionQuery([], ['course']),
  };

  const readByTagsRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTagsPayload),
    params
  );

  check(readByTagsRes, {
    'read by tags status is 200': (r) => r.status === 200,
    'read by tags duration < 300ms': (r) => r.timings.duration < 300,
  });

  if (readByTagsRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 5: Append with condition (should create more contention)
  const appendWithConditionPayload = {
    events: generateContentionEvent('ConditionalContentionEvent', ['course']),
    condition: {
      failIfEventsMatch: generateContentionQuery(['ConditionalContentionEvent'], ['course']),
    },
  };

  const appendWithConditionRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendWithConditionPayload),
    params
  );

  check(appendWithConditionRes, {
    'append with condition status is 200 or 409': (r) => r.status === 200 || r.status === 409,
    'append with condition duration < 500ms': (r) => r.timings.duration < 500,
  });

  if (appendWithConditionRes.status === 409) {
    conflictRate.add(1);
  } else if (appendWithConditionRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);  // Reduced from 0.1s for better performance

  // Test 6: Complex query (should succeed)
  const complexQueryPayload = {
    query: {
      items: [
        {
          types: ['ContentionEvent'],
          tags: [`course:${CONTENTION_TAGS.course}`],
        },
        {
          types: ['StudentEnrolled'],
          tags: [`student:${CONTENTION_TAGS.student}`],
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
    'complex query duration < 400ms': (r) => r.timings.duration < 400,
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

  // Add some initial events for testing with static tags (no __ITER)
  const initialEvents = [
    {
      type: 'CoursePlanned',
      data: JSON.stringify({ message: 'setup event' }),
      tags: ['course:setup', 'user:setup'],
    },
    {
      type: 'StudentEnrolled',
      data: JSON.stringify({ message: 'setup event' }),
      tags: ['course:setup', 'student:setup'],
    },
    {
      type: 'AssignmentCreated',
      data: JSON.stringify({ message: 'setup event' }),
      tags: ['course:setup', 'assignment:setup'],
    },
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