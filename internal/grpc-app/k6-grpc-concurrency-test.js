import grpc from "k6/net/grpc";
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conflictRate = new Rate('conflicts');

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

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
    'grpc_req_duration': ['p(95)<2000'], // 95% of requests should be below 2000ms
    errors: ['rate<0.30'],             // Error rate should be below 30% (expecting some conflicts)
    conflicts: ['rate>0.05'],          // Should see some conflicts (optimistic locking working)
  },
};

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
            message: `contention test event from VU ${typeof __VU !== 'undefined' ? __VU : 'setup'}`,
            iteration: typeof __ITER !== 'undefined' ? __ITER : 'setup',
            vu: typeof __VU !== 'undefined' ? __VU : 'setup'
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
  if (__ITER === 0) {
    client.connect('localhost:9090', {
      plaintext: true,
    });
  }

  // Test 1: Health check
  const healthResponse = client.invoke('eventstore.EventStoreService/Health', {});
  
  check(healthResponse, {
    'health status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!healthResponse || healthResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Health check failed:', healthResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 2: Append single event with contention
  const singleEvent = generateContentionEvent('ContentionEvent', ['course', 'user']);
  const appendSingleRequest = {
    events: [singleEvent]
  };

  const appendSingleResponse = client.invoke('eventstore.EventStoreService/Append', appendSingleRequest);
  
  check(appendSingleResponse, {
    'append single event status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.status === grpc.StatusAborted),
  });

  if (appendSingleResponse && appendSingleResponse.status === grpc.StatusAborted) {
    conflictRate.add(1);
    console.log(`Conflict detected on VU ${__VU}, iteration ${__ITER}`);
  } else if (!appendSingleResponse || appendSingleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append single event failed:', appendSingleResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 3: Append multiple events with contention
  const multipleEvents = [
    generateContentionEvent('StudentEnrolled', ['course', 'student']),
    generateContentionEvent('AssignmentCreated', ['course', 'assignment']),
    generateContentionEvent('GradeSubmitted', ['course', 'student', 'assignment']),
  ];

  const appendMultipleRequest = {
    events: multipleEvents
  };

  const appendMultipleResponse = client.invoke('eventstore.EventStoreService/Append', appendMultipleRequest);
  
  check(appendMultipleResponse, {
    'append multiple events status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.status === grpc.StatusAborted),
  });

  if (appendMultipleResponse && appendMultipleResponse.status === grpc.StatusAborted) {
    conflictRate.add(1);
    console.log(`Multiple events conflict detected on VU ${__VU}, iteration ${__ITER}`);
  } else if (!appendMultipleResponse || appendMultipleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append multiple events failed:', appendMultipleResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 4: Read events by type (should succeed)
  const readByTypeRequest = {
    query: generateContentionQuery(['ContentionEvent', 'StudentEnrolled'], [])
  };

  const readByTypeResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeRequest);
  
  check(readByTypeResponse, {
    'read by type status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTypeResponse || readByTypeResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type failed:', readByTypeResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 5: Read events by tags (should succeed)
  const readByTagsRequest = {
    query: generateContentionQuery([], ['course'])
  };

  const readByTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTagsRequest);
  
  check(readByTagsResponse, {
    'read by tags status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTagsResponse || readByTagsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by tags failed:', readByTagsResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 6: Read events by type and tags (should succeed)
  const readByTypeAndTagsRequest = {
    query: generateContentionQuery(['StudentEnrolled'], ['course'])
  };

  const readByTypeAndTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeAndTagsRequest);
  
  check(readByTypeAndTagsResponse, {
    'read by type and tags status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTypeAndTagsResponse || readByTypeAndTagsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type and tags failed:', readByTypeAndTagsResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 7: Complex query (should succeed)
  const complexQueryRequest = {
    query: {
      items: [
        {
          types: ['ContentionEvent'],
          tags: [`course:${CONTENTION_TAGS.course}`]
        },
        {
          types: ['StudentEnrolled'],
          tags: [`student:${CONTENTION_TAGS.student}`]
        }
      ]
    }
  };

  const complexQueryResponse = client.invoke('eventstore.EventStoreService/Read', complexQueryRequest);
  
  check(complexQueryResponse, {
    'complex query status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!complexQueryResponse || complexQueryResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Complex query failed:', complexQueryResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency

  // Test 8: Append with condition (should create more contention)
  const appendConditionRequest = {
    events: [generateContentionEvent('ConditionalContentionEvent', ['conditional', 'test'])],
    condition: {
      fail_if_events_match: {
        items: [{
          types: ['ConditionalContentionEvent'],
          tags: [`conditional:${CONTENTION_TAGS.course}`]
        }]
      }
    }
  };

  const appendConditionResponse = client.invoke('eventstore.EventStoreService/Append', appendConditionRequest);
  
  check(appendConditionResponse, {
    'append with condition status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.status === grpc.StatusAborted),
  });

  if (appendConditionResponse && appendConditionResponse.status === grpc.StatusAborted) {
    conflictRate.add(1);
    console.log(`Conditional append conflict detected on VU ${__VU}, iteration ${__ITER}`);
  } else if (!appendConditionResponse || appendConditionResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append with condition failed:', appendConditionResponse);
  }

  sleep(0.1);  // Standardized sleep for consistency
}

// Setup function to initialize test data
export function setup() {
  console.log('Setting up gRPC concurrency test data...');
  
  // Connect to gRPC server
  client.connect('localhost:9090', {
    plaintext: true,
  });

  // Add some initial events with contention tags
  const initialEvents = [
    generateContentionEvent('ContentionEvent', ['course', 'user']),
    generateContentionEvent('StudentEnrolled', ['course', 'student']),
    generateContentionEvent('AssignmentCreated', ['course', 'assignment']),
  ];

  const setupRequest = {
    events: initialEvents
  };

  const res = client.invoke('eventstore.EventStoreService/Append', setupRequest);

  if (!res || res.status !== grpc.StatusOK) {
    console.log('Setup failed:', res);
  } else {
    console.log('gRPC concurrency test setup completed successfully');
  }

  client.close();
  return { baseUrl: 'localhost:9090' };
}

export function teardown(data) {
  client.close();
} 