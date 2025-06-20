import grpc from 'k6/net/grpc';
import { check } from 'k6';

// Load the proto file
const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

// Very short test configuration
export const options = {
  vus: 1,
  duration: '10s',
};

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

  // Test 2: Append single event
  const singleEvent = {
    type: 'TestEvent',
    data: JSON.stringify({
      id: 'test-1',
      message: 'Hello World from gRPC',
      timestamp: new Date().toISOString(),
    }),
    tags: ['test:1', 'quick:test', 'grpc:test'],
  };
  const appendResponse = client.invoke('eventstore.EventStoreService/Append', { events: [singleEvent] });
  check(appendResponse, {
    'append status is ok': (r) => r && r.status === grpc.StatusOK,
    'append has no error': (r) => !r.error,
  });

  // Test 3: Read event
  const readRequest = {
    query: {
      items: [{ types: ['TestEvent'], tags: ['test:1'] }],
    },
  };
  const readResponse = client.invoke('eventstore.EventStoreService/Read', readRequest);
  check(readResponse, {
    'read status is ok': (r) => r && r.status === grpc.StatusOK,
    'read has no error': (r) => !r.error,
    'read returns events': (r) => r && r.message && r.message.events && r.message.events.length > 0,
  });
}

export function teardown(data) {
  client.close();
}
