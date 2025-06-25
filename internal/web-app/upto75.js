import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    default: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },   // Ramp up to 10 VUs
        { duration: '30s', target: 25 },   // Ramp up to 25 VUs
        { duration: '30s', target: 40 },   // Ramp up to 40 VUs
        { duration: '30s', target: 55 },   // Ramp up to 55 VUs
        { duration: '30s', target: 70 },   // Ramp up to 70 VUs
        { duration: '30s', target: 75 },   // Ramp up to 75 VUs
        { duration: '2m', target: 75 },    // Stay at 75 VUs for 2 minutes
      ],
      gracefulRampDown: '30s',
      gracefulStop: '30s',
    },
  },
  maxDuration: '4m30s',
};

export default function () {
  // Randomly choose between append and read operations
  const operation = Math.random() < 0.5 ? 'append' : 'read';
  
  if (operation === 'append') {
    // Append operation
    const appendPayload = JSON.stringify({
      events: [
        {
          type: "TestEvent",
          tags: ["test:upto75", "load:high"],
          data: JSON.stringify({ value: "test", timestamp: Date.now() })
        }
      ]
    });

    const appendResponse = http.post('http://localhost:8080/append', appendPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    check(appendResponse, {
      'append status is 200': (r) => r.status === 200,
      'append has duration': (r) => r.json('durationInMicroseconds') > 0,
    });
  } else {
    // Read operation
    const readPayload = JSON.stringify({
      query: {
        items: [
          {
            types: ["TestEvent"],
            tags: ["test:upto75"]
          }
        ]
      }
    });

    const readResponse = http.post('http://localhost:8080/read', readPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    check(readResponse, {
      'read status is 200': (r) => r.status === 200,
      'read has duration': (r) => r.json('durationInMicroseconds') > 0,
      'read has event count': (r) => r.json('numberOfMatchingEvents') >= 0,
    });
  }

  // Small sleep to prevent overwhelming the system
  sleep(0.1);
} 