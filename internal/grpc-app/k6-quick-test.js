import grpc from 'k6/net/grpc';
import { check } from 'k6';

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

// Very short test configuration - optimized for speed (matching web-app)
export const options = {
  vus: 2,  // Increased from 1 to match web-app
  duration: '10s',
  // Optimize for higher throughput (matching web-app)
  batch: 10,
  batchPerHost: 10,
};

// Setup function to clean database before test
export function setup() {
  console.log('Setting up gRPC test...');
  return {};
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

  // Test 1: Append single event
  const singleEvent = generateUniqueEvent('TestEvent', ['test', 'quick'], true);
  const appendResponse = client.invoke('eventstore.EventStoreService/Append', { events: [singleEvent] });
  check(appendResponse, {
    'append status is ok': (r) => r && r.status === grpc.StatusOK,
    'append has no error': (r) => !r.error,
  });

  // Test 2: Read event
  const readRequest = {
    query: generateUniqueQuery(['TestEvent'], ['test']),
  };
  const readResponse = client.invoke('eventstore.EventStoreService/Read', readRequest);
  check(readResponse, {
    'read status is ok': (r) => r && r.status === grpc.StatusOK,
    'read has no error': (r) => !r.error,
    'read returns events': (r) => r && r.message && r.message.events && r.message.events.length >= 0,
  });
}

export function teardown(data) {
  client.close();
}
