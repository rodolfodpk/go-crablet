import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const client = new grpc.Client();
client.load(['proto'], 'eventstore.proto');

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '2m', target: 10 },   // Stay at 10 users for 2 minutes
    { duration: '30s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    'grpc_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'errors': ['rate<0.1'],             // Error rate should be below 10%
  },
};

export default function () {
  if (__ITER === 0) {
    client.connect('localhost:9090', {
      plaintext: true,
    });
  }

  const data = {
    stream_id: `test-stream-${__VU}-${__ITER}`,
    events: [
      {
        event_type: 'test_event',
        data: JSON.stringify({
          message: `Test event ${__ITER} from VU ${__VU}`,
          timestamp: new Date().toISOString(),
        }),
        metadata: JSON.stringify({
          source: 'k6-benchmark',
          version: '1.0',
        }),
      },
    ],
  };

  const response = client.invoke('eventstore.EventStore/AppendEvents', data);

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
    'has stream_id': (r) => r && r.message && r.message.stream_id,
  });

  if (response.status !== grpc.StatusOK) {
    errorRate.add(1);
    console.log(`Error: ${response.status} - ${response.status_text}`);
  }

  sleep(1);
}

export function teardown(data) {
  client.close();
} 