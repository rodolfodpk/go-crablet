import grpc from "k6/net/grpc";
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

// Test configuration - optimized for higher resource allocation
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 10 },    // Stay at 10 users
    { duration: '30s', target: 25 },   // Ramp up to 25 users
    { duration: '2m', target: 25 },    // Stay at 25 users
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '3m', target: 50 },    // Stay at 50 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    'grpc_req_duration': ['p(95)<1500'], // 95% of requests should be below 1500ms
    'grpc_req_duration': ['p(99)<3000'], // 99% of requests should be below 3000ms
    errors: ['rate<0.15'],             // Error rate should be below 15%
    'grpc_reqs': ['rate>50'],          // Should handle at least 50 req/s
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
            message: `test event from k6`,
            iteration: __ITER
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

  sleep(0.1);

  // Test 2: Append single event
  const singleEvent = generateUniqueEvent('CoursePlanned', ['course', 'user']);
  const appendSingleRequest = {
    events: [singleEvent]
  };

  const appendSingleResponse = client.invoke('eventstore.EventStoreService/Append', appendSingleRequest);
  
  check(appendSingleResponse, {
    'append single event status is ok': (r) => r && r.status === grpc.StatusOK,
    'append single event duration < 200ms': (r) => r && r.duration < 200,
  });

  if (!appendSingleResponse || appendSingleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append single event failed:', appendSingleResponse);
  }

  sleep(0.1);

  // Test 3: Append multiple events
  const multipleEvents = [
    generateUniqueEvent('StudentEnrolled', ['course', 'student']),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment']),
    generateUniqueEvent('GradeSubmitted', ['course', 'student', 'assignment']),
  ];

  const appendMultipleRequest = {
    events: multipleEvents
  };

  const appendMultipleResponse = client.invoke('eventstore.EventStoreService/Append', appendMultipleRequest);
  
  check(appendMultipleResponse, {
    'append multiple events status is ok': (r) => r && r.status === grpc.StatusOK,
    'append multiple events duration < 300ms': (r) => r && r.duration < 300,
  });

  if (!appendMultipleResponse || appendMultipleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append multiple events failed:', appendMultipleResponse);
  }

  sleep(0.1);

  // Test 4: Read events by type
  const readByTypeRequest = {
    query: generateUniqueQuery(['CoursePlanned', 'StudentEnrolled'], [])
  };

  const readByTypeResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeRequest);
  
  check(readByTypeResponse, {
    'read by type status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type duration < 200ms': (r) => r && r.duration < 200,
  });

  if (!readByTypeResponse || readByTypeResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type failed:', readByTypeResponse);
  }

  sleep(0.1);

  // Test 5: Read events by tags
  const readByTagsRequest = {
    query: generateUniqueQuery([], ['course'])
  };

  const readByTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTagsRequest);
  
  check(readByTagsResponse, {
    'read by tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by tags duration < 200ms': (r) => r && r.duration < 200,
  });

  if (!readByTagsResponse || readByTagsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by tags failed:', readByTagsResponse);
  }

  sleep(0.1);

  // Test 6: Read events by type and tags
  const readByTypeAndTagsRequest = {
    query: generateUniqueQuery(['StudentEnrolled'], ['course'])
  };

  const readByTypeAndTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeAndTagsRequest);
  
  check(readByTypeAndTagsResponse, {
    'read by type and tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type and tags duration < 200ms': (r) => r && r.duration < 200,
  });

  if (!readByTypeAndTagsResponse || readByTypeAndTagsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type and tags failed:', readByTypeAndTagsResponse);
  }

  sleep(0.1);

  // Test 7: Complex query (OR)
  const complexQueryRequest = {
    query: {
      items: [
        {
          types: ['CoursePlanned'],
          tags: ['course']
        },
        {
          types: ['StudentEnrolled'],
          tags: ['student']
        }
      ]
    }
  };

  const complexQueryResponse = client.invoke('eventstore.EventStoreService/Read', complexQueryRequest);
  
  check(complexQueryResponse, {
    'complex query status is ok': (r) => r && r.status === grpc.StatusOK,
    'complex query duration < 250ms': (r) => r && r.duration < 250,
  });

  if (!complexQueryResponse || complexQueryResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Complex query failed:', complexQueryResponse);
  }

  sleep(0.1);

  // Test 8: Append with condition
  const appendConditionRequest = {
    events: [generateUniqueEvent('ConditionalEvent', ['conditional', 'test'])],
    condition: {
      fail_if_events_match: {
        items: [{
          types: ['ConditionalEvent'],
          tags: ['conditional']
        }]
      }
    }
  };

  const appendConditionResponse = client.invoke('eventstore.EventStoreService/Append', appendConditionRequest);
  
  check(appendConditionResponse, {
    'append with condition status is ok': (r) => r && r.status === grpc.StatusOK,
    'append with condition duration < 200ms': (r) => r && r.duration < 200,
  });

  if (!appendConditionResponse || appendConditionResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append with condition failed:', appendConditionResponse);
  }

  sleep(0.1);
}

export function teardown(data) {
  client.close();
}
