import http from 'k6/http';
import { check } from 'k6';

// Very short test configuration
export const options = {
  vus: 1,
  duration: '10s',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  // Test 1: Append single event
  const singleEvent = {
    type: 'TestEvent',
    data: JSON.stringify({
      id: 'test-1',
      message: 'Hello World',
    }),
    tags: ['test:1', 'quick:test'],
  };

  const appendRes = http.post(
    `${BASE_URL}/append`,
    JSON.stringify({ events: singleEvent }),
    params
  );

  check(appendRes, {
    'append status is 200': (r) => r.status === 200,
    'append has duration': (r) => r.json('durationInMicroseconds') > 0,
  });

  console.log('Append response:', appendRes.json());

  // Test 2: Read events
  const readPayload = {
    query: {
      items: [
        {
          types: ['TestEvent'],
          tags: ['test:1'],
        },
      ],
    },
  };

  const readRes = http.post(
    `${BASE_URL}/read`,
    JSON.stringify(readPayload),
    params
  );

  check(readRes, {
    'read status is 200': (r) => r.status === 200,
    'read has duration': (r) => r.json('durationInMicroseconds') > 0,
    'read has event count': (r) => r.json('numberOfMatchingEvents') >= 0,
  });

  console.log('Read response:', readRes.json());
} 