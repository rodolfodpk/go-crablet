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
    { duration: '30s', target: 5 },    // Warm-up: ramp up to 5 users
    { duration: '30s', target: 5 },    // Stay at 5 users (warm-up)
    { duration: '30s', target: 25 },   // Ramp up to 25 users
    { duration: '1m', target: 25 },    // Stay at 25 users
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 50 },    // Stay at 50 users
    { duration: '30s', target: 100 },  // Ramp up to 100 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    'grpc_req_duration': ['p(95)<1500'], // 95% of requests should be below 1500ms
    'grpc_req_duration': ['p(99)<3000'], // 99% of requests should be below 3000ms
    errors: ['rate<0.15'],             // Error rate should be below 15%
  },
};

// Generate unique IDs for each request to avoid concurrency bottlenecks
function generateUniqueId(prefix) {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Generate unique event with random IDs
function generateUniqueEvent(eventType, tagPrefixes, includeIteration) {
    const tags = tagPrefixes.map(prefix => `${prefix}:${generateUniqueId(prefix)}`);
    const eventData = {
        timestamp: new Date().toISOString(),
        message: `test event from k6`,
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
  if (__ITER === 0) {
    client.connect('localhost:9090', {
      plaintext: true,
    });
  }

  // Test 1: Append single event
  const singleEvent = generateUniqueEvent('CoursePlanned', ['course', 'user'], true);
  const appendSingleRequest = {
    events: [singleEvent]
  };

  const appendSingleResponse = client.invoke('eventstore.EventStoreService/Append', appendSingleRequest);
  
  check(appendSingleResponse, {
    'append single event status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!appendSingleResponse || appendSingleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append single event failed:', appendSingleResponse);
  }

  sleep(0.1);

  // Test 2: Append multiple events
  const multipleEvents = [
    generateUniqueEvent('StudentEnrolled', ['course', 'student'], true),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment'], true),
    generateUniqueEvent('GradeSubmitted', ['course', 'student', 'assignment'], true),
  ];

  const appendMultipleRequest = {
    events: multipleEvents
  };

  const appendMultipleResponse = client.invoke('eventstore.EventStoreService/Append', appendMultipleRequest);
  
  check(appendMultipleResponse, {
    'append multiple events status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!appendMultipleResponse || appendMultipleResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append multiple events failed:', appendMultipleResponse);
  }

  sleep(0.1);

  // Test 3: Read events by type and tags (targeted query)
  const readByTypeAndTagsRequest = {
    query: generateUniqueQuery(['StudentEnrolled'], ['course'])
  };

  const readByTypeAndTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeAndTagsRequest);
  
  check(readByTypeAndTagsResponse, {
    'read by type and tags status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTypeAndTagsResponse || readByTypeAndTagsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type and tags failed:', readByTypeAndTagsResponse);
  }

  sleep(0.1);

  // Test 4: Append with condition (should fail if events exist)
  const appendWithConditionRequest = {
    events: [generateUniqueEvent('DuplicateEvent', ['course'], true)],
    condition: {
      failIfEventsMatch: generateUniqueQuery(['DuplicateEvent'], ['course'])
    }
  };

  const appendWithConditionResponse = client.invoke('eventstore.EventStoreService/Append', appendWithConditionRequest);
  
  check(appendWithConditionResponse, {
    'append with condition status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!appendWithConditionResponse || appendWithConditionResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append with condition failed:', appendWithConditionResponse);
  }

  sleep(0.1);

  // Test 5: Complex query with multiple items
  const complexQueryRequest = {
    query: {
      items: [
        {
          types: ['CoursePlanned'],
          tags: [`course:${generateUniqueId('course')}`]
        },
        {
          types: ['StudentEnrolled'],
          tags: [`student:${generateUniqueId('student')}`]
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

  sleep(0.1);
}

// Setup function to initialize test data
export function setup() {
  console.log('Setting up gRPC test data...');
  
  // Wait a bit for the server to be ready
  sleep(3);
  
  // Connect to gRPC server
  client.connect('localhost:9090', {
    plaintext: true,
  });

  // Add some initial events with unique IDs
  const initialEvents = [
    generateUniqueEvent('CoursePlanned', ['course', 'user'], false),
    generateUniqueEvent('StudentEnrolled', ['course', 'student'], false),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment'], false),
  ];

  const setupRequest = {
    events: initialEvents
  };

  const res = client.invoke('eventstore.EventStoreService/Append', setupRequest);

  if (!res || res.status !== grpc.StatusOK) {
    console.log('Setup failed:', res);
  } else {
    console.log('gRPC test setup completed successfully');
  }

  client.close();
  return { baseUrl: 'localhost:9090' };
}

export function teardown(data) {
  client.close();
}
