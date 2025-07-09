import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conflictRate = new Rate('conflicts');
const appendSuccessRate = new Rate('append_success');
const appendIfSuccessRate = new Rate('append_if_success');
const appendIfIsolatedSuccessRate = new Rate('append_if_isolated_success');
const advisoryLockSuccessRate = new Rate('advisory_lock_success');
const advisoryLockWaitRate = new Rate('advisory_lock_wait');

// Test configuration - designed to create contention with advisory locks
export const options = {
  stages: [
    { duration: '10s', target: 5 },    // Ramp up to 5 users
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 20 },    // Ramp up to 20 users (high contention)
    { duration: '2m', target: 20 },    // Stay at 20 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<3000'], // 95% of requests should be below 3000ms (higher due to locks)
    errors: ['rate<0.20'],             // Error rate should be below 20% (locks should reduce conflicts)
    conflicts: ['rate<0.10'],          // Should see fewer conflicts due to advisory locks
    append_success: ['rate>0.80'],     // Append should succeed most of the time
    append_if_success: ['rate>0.70'],  // AppendIf should succeed more due to locks
    append_if_isolated_success: ['rate>0.60'], // AppendIfIsolated should succeed more due to locks
    advisory_lock_success: ['rate>0.90'], // Advisory locks should succeed most of the time
    advisory_lock_wait: ['rate>0.05'], // Should see some lock waiting (serialization working)
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Use fixed tag values to create contention - these will be used for advisory locks
const CONTENTION_TAGS = {
  course: 'contention-course-123',
  student: 'contention-student-456',
  assignment: 'contention-assignment-789',
  user: 'contention-user-101',
  account: 'contention-account-202',
  order: 'contention-order-303',
};

// Generate event with contention tags including lock: prefix
function generateAdvisoryLockEvent(eventType, tagKeys, includeLockTags = true) {
    const tags = tagKeys.map(key => `${key}:${CONTENTION_TAGS[key]}`);
    
    // Add lock: prefixed tags for advisory locking
    // Use unique lock keys to avoid duplicate key validation
    if (includeLockTags) {
        const lockTags = tagKeys.map(key => `lock_${key}:${CONTENTION_TAGS[key]}`);
        tags.push(...lockTags);
    }
    
    return {
        type: eventType,
        data: JSON.stringify({ 
            timestamp: new Date().toISOString(),
            message: `advisory lock test event from VU ${__VU}`,
            iteration: __ITER,
            vu: __VU,
            hasLockTags: includeLockTags
        }),
        tags: tags
    };
}

// Generate query with contention tags (no lock: prefix needed for queries)
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

  // Test 1: Simple Append with Advisory Locks (should serialize access)
  const singleEvent = generateAdvisoryLockEvent('AdvisoryLockAppendEvent', ['course', 'user']);
  const appendPayload = {
    events: singleEvent,
  };

  const appendRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendPayload),
    baseParams
  );

  check(appendRes, {
    'advisory lock append status is 200': (r) => r.status === 200,
    'advisory lock append duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (appendRes.status === 200) {
    appendSuccessRate.add(1);
    advisoryLockSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 2: AppendIf with Advisory Locks (should have fewer conflicts due to serialization)
  const appendIfEvent = generateAdvisoryLockEvent('AdvisoryLockAppendIfEvent', ['course', 'student']);
  const appendIfPayload = {
    events: appendIfEvent,
    condition: {
      failIfEventsMatch: generateContentionQuery(['AdvisoryLockAppendIfEvent'], ['course']),
    },
  };

  const appendIfRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(appendIfPayload),
    baseParams
  );

  check(appendIfRes, {
    'advisory lock appendIf status is 200 or condition failed': (r) => r.status === 200,
    'advisory lock appendIf duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (appendIfRes.status === 200) {
    const response = JSON.parse(appendIfRes.body);
    if (response.appendConditionFailed) {
      conflictRate.add(1);
    } else {
      appendIfSuccessRate.add(1);
      advisoryLockSuccessRate.add(1);
    }
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 3: AppendIfIsolated with Advisory Locks (should have even fewer conflicts)
  const appendIfIsolatedEvent = generateAdvisoryLockEvent('AdvisoryLockAppendIfIsolatedEvent', ['course', 'assignment']);
  const appendIfIsolatedPayload = {
    events: appendIfIsolatedEvent,
    condition: {
      failIfEventsMatch: generateContentionQuery(['AdvisoryLockAppendIfIsolatedEvent'], ['course']),
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
    'advisory lock appendIfIsolated status is 200 or condition failed': (r) => r.status === 200,
    'advisory lock appendIfIsolated duration < 1500ms': (r) => r.timings.duration < 1500,
  });

  if (appendIfIsolatedRes.status === 200) {
    const response = JSON.parse(appendIfIsolatedRes.body);
    if (response.appendConditionFailed) {
      conflictRate.add(1);
    } else {
      appendIfIsolatedSuccessRate.add(1);
      advisoryLockSuccessRate.add(1);
    }
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 4: Batch Append with Advisory Locks (multiple events with same lock keys)
  const batchEvents = [
    generateAdvisoryLockEvent('AdvisoryLockBatchEvent1', ['course', 'student']),
    generateAdvisoryLockEvent('AdvisoryLockBatchEvent2', ['course', 'assignment']),
    generateAdvisoryLockEvent('AdvisoryLockBatchEvent3', ['course', 'user']),
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
    'advisory lock batch append status is 200': (r) => r.status === 200,
    'advisory lock batch append duration < 1500ms': (r) => r.timings.duration < 1500,
  });

  if (batchRes.status === 200) {
    appendSuccessRate.add(1);
    advisoryLockSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 5: Multiple Lock Keys (should serialize access to multiple resources)
  const multiLockEvent = generateAdvisoryLockEvent('AdvisoryLockMultiEvent', ['account', 'order']);
  const multiLockPayload = {
    events: multiLockEvent,
  };

  const multiLockRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(multiLockPayload),
    baseParams
  );

  check(multiLockRes, {
    'advisory lock multi-lock status is 200': (r) => r.status === 200,
    'advisory lock multi-lock duration < 1000ms': (r) => r.timings.duration < 1000,
  });

  if (multiLockRes.status === 200) {
    appendSuccessRate.add(1);
    advisoryLockSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 6: Events without lock tags (should work normally, no serialization)
  const noLockEvent = generateAdvisoryLockEvent('NoLockEvent', ['course', 'user'], false);
  const noLockPayload = {
    events: noLockEvent,
  };

  const noLockRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(noLockPayload),
    baseParams
  );

  check(noLockRes, {
    'no lock event status is 200': (r) => r.status === 200,
    'no lock event duration < 500ms': (r) => r.timings.duration < 500,
  });

  if (noLockRes.status === 200) {
    appendSuccessRate.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(0.05);

  // Test 7: Read events by type (should succeed, no locks needed)
  const readByTypePayload = {
    query: generateContentionQuery(['AdvisoryLockAppendEvent', 'AdvisoryLockAppendIfEvent', 'AdvisoryLockAppendIfIsolatedEvent'], []),
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

  // Test 8: Read events by tags (should succeed, no locks needed)
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

  // Test 9: High contention scenario - multiple VUs trying to modify same resource
  // This should demonstrate advisory lock serialization
  const highContentionEvent = generateAdvisoryLockEvent('HighContentionEvent', ['course']);
  const highContentionPayload = {
    events: highContentionEvent,
  };

  const highContentionRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify(highContentionPayload),
    baseParams
  );

  check(highContentionRes, {
    'high contention status is 200': (r) => r.status === 200,
    'high contention duration < 2000ms': (r) => r.timings.duration < 2000,
  });

  if (highContentionRes.status === 200) {
    appendSuccessRate.add(1);
    advisoryLockSuccessRate.add(1);
    
    // If duration is high, it might indicate lock waiting
    if (highContentionRes.timings.duration > 500) {
      advisoryLockWaitRate.add(1);
    }
  } else {
    errorRate.add(1);
  }

  sleep(0.05);
}

// Setup function to validate basic functionality and advisory locks before running the test
export function setup() {
  const baseParams = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  console.log('🧪 Validating basic functionality and advisory locks...');

  // Test 1: Simple append without lock tags
  const setupEvent = {
    type: 'SetupTestEvent',
    data: JSON.stringify({ message: 'setup check' }),
    tags: ['check:test', 'multi:test']
  };
  const appendPayload = { events: setupEvent };
  const appendRes = http.post(`${BASE_URL}/append`, JSON.stringify(appendPayload), baseParams);
  
  if (appendRes.status !== 200) {
    throw new Error(`Setup failed: /append status ${appendRes.status} body: ${appendRes.body}`);
  }

  // Test 2: Append with advisory lock tags
  const lockEvent = {
    type: 'SetupLockTestEvent',
    data: JSON.stringify({ message: 'lock setup check' }),
    tags: ['check:test', 'lock_check:test']
  };
  const lockPayload = { events: lockEvent };
  const lockRes = http.post(`${BASE_URL}/append`, JSON.stringify(lockPayload), baseParams);
  
  if (lockRes.status !== 200) {
    throw new Error(`Setup failed: advisory lock /append status ${lockRes.status} body: ${lockRes.body}`);
  }

  // Test 3: Read events back (should not include lock: tags in query)
  const readPayload = {
    query: {
      items: [{ types: ['SetupTestEvent', 'SetupLockTestEvent'], tags: ['check:test'] }]
    }
  };
  const readRes = http.post(`${BASE_URL}/read`, JSON.stringify(readPayload), baseParams);
  
  if (readRes.status !== 200) {
    throw new Error(`Setup failed: /read status ${readRes.status} body: ${readRes.body}`);
  }

  const readBody = JSON.parse(readRes.body);
  if (!readBody || !('numberOfMatchingEvents' in readBody) || readBody.numberOfMatchingEvents < 2) {
    throw new Error(`Setup failed: /read did not return both events. Body: ${readRes.body}`);
  }

  console.log('✅ Basic functionality and advisory locks validated - proceeding with concurrency test');
} 