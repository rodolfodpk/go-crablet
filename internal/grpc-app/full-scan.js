import grpc from "k6/net/grpc";
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

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
    'grpc_req_duration': ['p(95)<2000'], // 95% of requests should be below 2000ms
    'grpc_req_duration': ['p(99)<4000'], // 99% of requests should be below 4000ms
    errors: ['rate<0.20'],             // Error rate should be below 20%
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
        message: `full-scan test event from k6`,
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

  // Test 1: Read events by type only (empty tags - full scan)
  const readByTypeOnlyRequest = {
    query: generateUniqueQuery(['CoursePlanned', 'StudentEnrolled'], [])
  };

  const readByTypeOnlyResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeOnlyRequest);
  
  check(readByTypeOnlyResponse, {
    'read by type only status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTypeOnlyResponse || readByTypeOnlyResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by type only failed:', readByTypeOnlyResponse);
  }

  sleep(0.1);

  // Test 2: Read events by tag only (empty types - full scan)
  const readByTagOnlyRequest = {
    query: generateUniqueQuery([], ['course'])
  };

  const readByTagOnlyResponse = client.invoke('eventstore.EventStoreService/Read', readByTagOnlyRequest);
  
  check(readByTagOnlyResponse, {
    'read by tag only status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readByTagOnlyResponse || readByTagOnlyResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read by tag only failed:', readByTagOnlyResponse);
  }

  sleep(0.1);

  // Test 3: Read all events (empty types and tags - full table scan)
  const readAllEventsRequest = {
    query: generateUniqueQuery([], [])
  };

  const readAllEventsResponse = client.invoke('eventstore.EventStoreService/Read', readAllEventsRequest);
  
  check(readAllEventsResponse, {
    'read all events status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!readAllEventsResponse || readAllEventsResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Read all events failed:', readAllEventsResponse);
  }

  sleep(0.1);

  // Test 4: Complex query with multiple items, some with empty filters
  const complexFullScanRequest = {
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

  const complexFullScanResponse = client.invoke('eventstore.EventStoreService/Read', complexFullScanRequest);
  
  check(complexFullScanResponse, {
    'complex full scan status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!complexFullScanResponse || complexFullScanResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Complex full scan failed:', complexFullScanResponse);
  }

  sleep(0.1);

  // Test 5: Append events for full-scan testing
  const fullScanEvents = [
    generateUniqueEvent('FullScanEvent1', ['fullscan', 'test'], true),
    generateUniqueEvent('FullScanEvent2', ['fullscan', 'test'], true),
    generateUniqueEvent('FullScanEvent3', ['fullscan', 'test'], true),
  ];

  const appendFullScanRequest = {
    events: fullScanEvents
  };

  const appendFullScanResponse = client.invoke('eventstore.EventStoreService/Append', appendFullScanRequest);
  
  check(appendFullScanResponse, {
    'append full scan events status is ok': (r) => r && r.status === grpc.StatusOK,
  });

  if (!appendFullScanResponse || appendFullScanResponse.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log('Append full scan events failed:', appendFullScanResponse);
  }

  sleep(0.1);
}

// Setup function to initialize test data
export function setup() {
  console.log('Setting up gRPC full-scan test data...');
  
  // Wait a bit for the server to be ready
  sleep(3);
  
  // Connect to gRPC server
  client.connect('localhost:9090', {
    plaintext: true,
  });

  // Add some initial events for full-scan testing
  const initialEvents = [
    generateUniqueEvent('CoursePlanned', ['course', 'user'], false),
    generateUniqueEvent('StudentEnrolled', ['course', 'student'], false),
    generateUniqueEvent('AssignmentCreated', ['course', 'assignment'], false),
    generateUniqueEvent('FullScanEvent', ['fullscan', 'baseline'], false),
  ];

  const setupRequest = {
    events: initialEvents
  };

  const res = client.invoke('eventstore.EventStoreService/Append', setupRequest);

  if (!res || res.status !== grpc.StatusOK) {
    console.log('Full-scan setup failed:', res);
  } else {
    console.log('gRPC full-scan test setup completed successfully');
  }

  client.close();
  return { baseUrl: 'localhost:9090' };
}

export function teardown(data) {
  client.close();
} 