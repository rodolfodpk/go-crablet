import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conflictRate = new Rate('conflicts');
const appendSuccessRate = new Rate('append_success');
const appendIfSuccessRate = new Rate('append_if_success');
const appendIfIsolatedSuccessRate = new Rate('append_if_isolated_success');

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
    append_success: ['rate>0.70'],     // Append should succeed most of the time
    append_if_success: ['rate>0.50'],  // AppendIf should succeed about half the time due to conditions
    append_if_isolated_success: ['rate>0.30'], // AppendIfIsolated should succeed less due to Serializable
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
  const baseParams = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '30s',
  };

  // Test 1: Simple Append (Read Committed) - should succeed most of the time
  const singleEvent = generateContentionEvent('SimpleAppendEvent', ['course', 'user']);
  const appendPayload = {
    events: singleEvent,
  };

  const appendRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendPayload),
    baseParams
  );

  check(appendRes, {
    'simple append status is 200': (r) => r.status === 200,
    'simple append duration < 500ms': (r) => r.timings.duration < 500,
  });

  if (appendRes.status === 200) {
    appendSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 2: AppendIf (Repeatable Read) - with condition, should have some conflicts
  const appendIfEvent = generateContentionEvent('AppendIfEvent', ['course', 'student']);
  const appendIfPayload = {
    events: appendIfEvent,
    condition: {
      failIfEventsMatch: generateContentionQuery(['AppendIfEvent'], ['course']),
    },
  };

  const appendIfRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendIfPayload),
    baseParams
  );

  check(appendIfRes, {
    'appendIf status is 200 or condition failed': (r) => r.status === 200,
    'appendIf duration < 500ms': (r) => r.timings.duration < 500,
  });

  if (appendIfRes.status === 200) {
    const response = JSON.parse(appendIfRes.body);
    if (response.appendConditionFailed) {
      conflictRate.add(1);
    } else {
      appendIfSuccessRate.add(1);
    }
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 3: AppendIfIsolated (Serializable) - with condition and header, should have more conflicts
  const appendIfIsolatedEvent = generateContentionEvent('AppendIfIsolatedEvent', ['course', 'assignment']);
  const appendIfIsolatedPayload = {
    events: appendIfIsolatedEvent,
    condition: {
      failIfEventsMatch: generateContentionQuery(['AppendIfIsolatedEvent'], ['course']),
    },
  };

  const isolatedParams = {
    ...baseParams,
    headers: {
      ...baseParams.headers,
      'X-Append-If-Isolation': 'SERIALIZABLE',
    },
  };

  const appendIfIsolatedRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendIfIsolatedPayload),
    isolatedParams
  );

  check(appendIfIsolatedRes, {
    'appendIfIsolated status is 200 or condition failed': (r) => r.status === 200,
    'appendIfIsolated duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (appendIfIsolatedRes.status === 200) {
    const response = JSON.parse(appendIfIsolatedRes.body);
    if (response.appendConditionFailed) {
      conflictRate.add(1);
    } else {
      appendIfIsolatedSuccessRate.add(1);
    }
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 4: Batch Append (Read Committed) - multiple events
  const batchEvents = [
    generateContentionEvent('BatchEvent1', ['course', 'student']),
    generateContentionEvent('BatchEvent2', ['course', 'assignment']),
    generateContentionEvent('BatchEvent3', ['course', 'user']),
  ];

  const batchPayload = {
    events: batchEvents,
  };

  const batchRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(batchPayload),
    baseParams
  );

  check(batchRes, {
    'batch append status is 200': (r) => r.status === 200,
    'batch append duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (batchRes.status === 200) {
    appendSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 5: Read events by type (should succeed)
  const readByTypePayload = {
    query: generateContentionQuery(['SimpleAppendEvent', 'AppendIfEvent', 'AppendIfIsolatedEvent'], []),
  };

  const readByTypeRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTypePayload),
    baseParams
  );

  check(readByTypeRes, {
    'read by type status is 200': (r) => r.status === 200,
    'read by type duration < 300ms': (r) => r.timings.duration < 300,
  });

  if (readByTypeRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 6: Read events by tags (should succeed)
  const readByTagsPayload = {
    query: generateContentionQuery([], ['course']),
  };

  const readByTagsRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readByTagsPayload),
    baseParams
  );

  check(readByTagsRes, {
    'read by tags status is 200': (r) => r.status === 200,
    'read by tags duration < 300ms': (r) => r.timings.duration < 300,
  });

  if (readByTagsRes.status !== 200) {
    errorRate.add(1);
  }

  sleep(0.05);
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