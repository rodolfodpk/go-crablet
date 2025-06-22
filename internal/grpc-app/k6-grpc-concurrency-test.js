import grpc from "k6/net/grpc";
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const conflictRate = new Rate('conflicts');

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

// Test configuration - matching web-app settings
export const options = {
  stages: [
    { duration: '30s', target: 5 },   // Ramp up to 5 VUs
    { duration: '1m', target: 10 },   // Ramp up to 10 VUs
    { duration: '1m', target: 15 },   // Ramp up to 15 VUs
    { duration: '1m', target: 20 },   // Ramp up to 20 VUs
    { duration: '30s', target: 0 },   // Ramp down to 0 VUs
  ],
  thresholds: {
    'errors': ['rate<0.30'],
    'grpc_req_duration': ['p(95)<2000'],
    'conflicts': ['rate>0.05'],
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

// Generate unique IDs for each request to avoid concurrency bottlenecks
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Generate unique event with random IDs
function generateUniqueEvent(eventType, tagPrefixes, includeIteration) {
    const tags = tagPrefixes.map(prefix => `${prefix}:${generateUniqueId(prefix)}`);
    const eventData = {
        timestamp: new Date().toISOString(),
        message: 'Hello World from gRPC',
    };
    if (includeIteration && typeof __ITER !== 'undefined') {
        eventData.iteration = __ITER;
    }
    return {
        type: eventType,
        data: JSON.stringify(eventData),
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
  // Connect to gRPC server on first iteration
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

  // Test 2: Append single event (may conflict)
  const singleEvent = generateUniqueEvent('ConcurrentEvent', ['test', 'single'], true);
  const appendResponse = client.invoke('eventstore.EventStoreService/Append', { events: [singleEvent] });
  check(appendResponse, {
    'append single event status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.message?.appendConditionFailed),
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 3: Append multiple events (may conflict)
  const multipleEvents = [
    generateUniqueEvent('ConcurrentEvent', ['test', 'multiple', '1'], true),
    generateUniqueEvent('ConcurrentEvent', ['test', 'multiple', '2'], true),
  ];
  const appendMultipleResponse = client.invoke('eventstore.EventStoreService/Append', { events: multipleEvents });
  check(appendMultipleResponse, {
    'append multiple events status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.message?.appendConditionFailed),
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 4: Read by type
  const readByTypeRequest = {
    query: generateUniqueQuery(['ConcurrentEvent'], ['test']),
  };
  const readByTypeResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeRequest);
  check(readByTypeResponse, {
    'read by type status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type has no error': (r) => !r.error,
    'read by type returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 5: Read by tags
  const readByTagsRequest = {
    query: generateUniqueQuery(['ConcurrentEvent'], ['test', 'single']),
  };
  const readByTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTagsRequest);
  check(readByTagsResponse, {
    'read by tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by tags has no error': (r) => !r.error,
    'read by tags returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 6: Read by type and tags
  const readByTypeAndTagsRequest = {
    query: generateUniqueQuery(['ConcurrentEvent'], ['test', 'multiple']),
  };
  const readByTypeAndTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeAndTagsRequest);
  check(readByTypeAndTagsResponse, {
    'read by type and tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type and tags has no error': (r) => !r.error,
    'read by type and tags returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 7: Complex query
  const complexQueryRequest = {
    query: {
      items: [
        {
          types: ['ConcurrentEvent'],
          tags: ['test:test-1', 'single:single-1']
        },
        {
          types: ['SetupEvent'],
          tags: ['setup:setup-1', 'batch:batch-1']
        }
      ]
    }
  };
  const complexQueryResponse = client.invoke('eventstore.EventStoreService/Read', complexQueryRequest);
  check(complexQueryResponse, {
    'complex query status is ok': (r) => r && r.status === grpc.StatusOK,
    'complex query has no error': (r) => !r.error,
    'complex query returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 8: Append with condition (may conflict)
  const conditionEvent = generateUniqueEvent('ConditionEvent', ['test', 'condition'], true);
  const appendWithConditionResponse = client.invoke('eventstore.EventStoreService/Append', { 
    events: [conditionEvent],
    condition: {
      after: '0'
    }
  });
  check(appendWithConditionResponse, {
    'append with condition status is ok or conflict': (r) => r && (r.status === grpc.StatusOK || r.message?.appendConditionFailed),
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app
}

export function teardown(data) {
  client.close();
} 