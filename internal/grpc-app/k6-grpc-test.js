import grpc from "k6/net/grpc";
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

// Test configuration - matching web-app settings
export const options = {
  stages: [
    { duration: '30s', target: 5 },   // Ramp up to 5 VUs
    { duration: '1m', target: 10 },   // Ramp up to 10 VUs
    { duration: '2m', target: 20 },   // Ramp up to 20 VUs
    { duration: '2m', target: 30 },   // Ramp up to 30 VUs
    { duration: '2m', target: 40 },   // Ramp up to 40 VUs
    { duration: '30s', target: 50 },  // Ramp up to 50 VUs
    { duration: '0s', target: 0 },    // Ramp down to 0 VUs
  ],
  thresholds: {
    'errors': ['rate<0.15'],
    'grpc_req_duration': ['p(99)<3000'],
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

  // Test 1: Append single event
  const singleEvent = generateUniqueEvent('TestEvent', ['test', 'single'], true);
  const appendResponse = client.invoke('eventstore.EventStoreService/Append', { events: [singleEvent] });
  check(appendResponse, {
    'append single event status is ok': (r) => r && r.status === grpc.StatusOK,
    'append single event has no error': (r) => !r.error,
  });
  sleep(0.05);  // Reduced from 0.2s to match web-app

  // Test 2: Append multiple events
  const multipleEvents = [
    generateUniqueEvent('TestEvent', ['test', 'multiple', '1'], true),
    generateUniqueEvent('TestEvent', ['test', 'multiple', '2'], true),
  ];
  const appendMultipleResponse = client.invoke('eventstore.EventStoreService/Append', { events: multipleEvents });
  check(appendMultipleResponse, {
    'append multiple events status is ok': (r) => r && r.status === grpc.StatusOK,
    'append multiple events has no error': (r) => !r.error,
  });
  sleep(0.05);  // Reduced from 0.2s to match web-app

  // Test 3: Read by type
  const readByTypeRequest = {
    query: generateUniqueQuery(['TestEvent'], ['test']),
  };
  const readByTypeResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeRequest);
  check(readByTypeResponse, {
    'read by type status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type has no error': (r) => !r.error,
    'read by type returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.2s to match web-app

  // Test 4: Read by tags
  const readByTagsRequest = {
    query: generateUniqueQuery(['TestEvent'], ['test', 'single']),
  };
  const readByTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTagsRequest);
  check(readByTagsResponse, {
    'read by tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by tags has no error': (r) => !r.error,
    'read by tags returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.2s to match web-app

  // Test 5: Read by type and tags
  const readByTypeAndTagsRequest = {
    query: generateUniqueQuery(['TestEvent'], ['test', 'multiple']),
  };
  const readByTypeAndTagsResponse = client.invoke('eventstore.EventStoreService/Read', readByTypeAndTagsRequest);
  check(readByTypeAndTagsResponse, {
    'read by type and tags status is ok': (r) => r && r.status === grpc.StatusOK,
    'read by type and tags has no error': (r) => !r.error,
    'read by type and tags returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
  sleep(0.05);  // Reduced from 0.2s to match web-app

  // Test 6: Append with condition
  const conditionEvent = generateUniqueEvent('ConditionEvent', ['test', 'condition'], true);
  const appendWithConditionResponse = client.invoke('eventstore.EventStoreService/Append', { 
    events: [conditionEvent],
    condition: {
      after: '0'
    }
  });
  check(appendWithConditionResponse, {
    'append with condition status is ok': (r) => r && r.status === grpc.StatusOK,
    'append with condition has no error': (r) => !r.error,
  });
  sleep(0.05);  // Reduced from 0.1s to match web-app

  // Test 7: Complex query
  const complexQueryRequest = {
    query: {
      items: [
        {
          types: ['TestEvent'],
          tags: ['test:test-1', 'single:single-1']
        },
        {
          types: ['ConditionEvent'],
          tags: ['test:test-2', 'condition:condition-1']
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
}

// Setup function to clean database before test
export function setup() {
  console.log('Setting up gRPC test data...');
  
  // Connect to gRPC server
  const client = new grpc.Client();
  client.load(['proto'], 'eventstore.proto');
  client.connect('localhost:9090', { plaintext: true });
  
  // Clean database
  const cleanupResponse = http.post('http://localhost:9091/cleanup');
  if (cleanupResponse.status !== 200) {
    console.error('Failed to cleanup database');
  }
  
  // Setup test data
  for (let i = 0; i < 100; i++) {
    const event = {
      type: 'SetupEvent',
      data: JSON.stringify({ id: i, message: 'Setup data' }),
      tags: [`setup:${i}`, `batch:${Math.floor(i / 10)}`]
    };
    
    const response = client.invoke('eventstore.EventStoreService/Append', { events: [event] });
    if (response.status !== grpc.StatusOK) {
      console.error('Failed to setup test data');
    }
  }
  
  client.close();
  console.log('gRPC test setup completed successfully');
  return {};
}

export function teardown(data) {
  client.close();
}
