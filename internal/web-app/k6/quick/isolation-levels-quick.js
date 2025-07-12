import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Custom metrics
const appendSuccess = new Rate('append_success');
const readSuccess = new Rate('read_success');

// Test all isolation levels
const ISOLATION_LEVELS = [
    { name: 'READ_COMMITTED', header: null }, // Default
    { name: 'REPEATABLE_READ', header: 'X-Isolation-Level: REPEATABLE_READ' },
    { name: 'SERIALIZABLE', header: 'X-Isolation-Level: SERIALIZABLE' }
];

export const options = {
    stages: [
        { duration: '10s', target: 5 },  // Ramp up
        { duration: '20s', target: 10 }, // Sustained load
    ],
    thresholds: {
        http_req_failed: ['rate<0.05'],     // Less than 5% errors
        http_req_duration: ['p(95)<500'],   // 95% of requests under 500ms
        append_success: ['rate>0.95'],      // 95% append success rate
        read_success: ['rate>0.95'],        // 95% read success rate
    },
};

export function setup() {
    // Clean database
    const cleanupRes = http.post(`${BASE_URL}/cleanup`);
    check(cleanupRes, {
        'cleanup successful': (r) => r.status === 200,
    });

    // Test basic append for each isolation level
    for (const level of ISOLATION_LEVELS) {
        const testEvent = {
            type: 'TestEvent',
            data: JSON.stringify({ message: `Test ${level.name} isolation` }),
            metadata: { timestamp: Date.now() },
            tags: [`test:${level.name.toLowerCase()}:setup`]
        };
        
        const testPayload = {
            events: testEvent
        };

        const params = {
            headers: {
                'Content-Type': 'application/json',
            },
        };

        // Add isolation level header if specified
        if (level.header) {
            const [key, value] = level.header.split(': ');
            params.headers[key] = value;
        }

        const response = http.post(
            `${BASE_URL}/append`,
            JSON.stringify(testPayload),
            params
        );

        check(response, {
            [`setup ${level.name} append successful`]: (r) => r.status === 200,
            [`setup ${level.name} response has duration`]: (r) => r.json('durationInMicroseconds') > 0,
        });

        // Test read back
        const readPayload = {
            query: {
                items: [{ types: ['TestEvent'], tags: [`test:${level.name.toLowerCase()}:setup`] }]
            }
        };
        const readRes = http.post(`${BASE_URL}/read`, JSON.stringify(readPayload), params);
        check(readRes, {
            [`setup ${level.name} read successful`]: (r) => r.status === 200,
            [`setup ${level.name} read returns events`]: (r) => r.json('numberOfMatchingEvents') > 0,
        });
    }

    return { isolationLevels: ISOLATION_LEVELS };
}

export default function(data) {
    // Randomly select an isolation level for this iteration
    const level = data.isolationLevels[Math.floor(Math.random() * data.isolationLevels.length)];
    const streamId = `${level.name.toLowerCase()}-${__VU}-${__ITER}`;
    
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    // Add isolation level header if specified
    if (level.header) {
        const [key, value] = level.header.split(': ');
        params.headers[key] = value;
    }

    // Test 1: Simple append
    const simpleEvent = {
        type: 'SimpleEvent',
        data: JSON.stringify({ value: Math.random(), isolation: level.name }),
        metadata: { timestamp: Date.now() },
        tags: [`${level.name.toLowerCase()}:${__VU}`]
    };
    
    const simplePayload = {
        events: simpleEvent
    };

    const response = http.post(
        `${BASE_URL}/append`,
        JSON.stringify(simplePayload),
        params
    );

    const appendCheck = check(response, {
        'append successful': (r) => r.status === 200,
        'response has duration': (r) => r.json('durationInMicroseconds') > 0,
        'response has correct structure': (r) => r.json('appendConditionFailed') !== undefined,
    });

    appendSuccess.add(appendCheck);

    // Test 2: Conditional append
    const conditionalEvent = {
        type: 'ConditionalEvent',
        data: JSON.stringify({ value: Math.random(), isolation: level.name }),
        metadata: { timestamp: Date.now() },
        tags: [`${level.name.toLowerCase()}:${__VU}`]
    };
    
    const conditionalPayload = {
        events: conditionalEvent,
        condition: {
            expectedVersion: 0
        }
    };

    const conditionalResponse = http.post(
        `${BASE_URL}/append`,
        JSON.stringify(conditionalPayload),
        params
    );

    check(conditionalResponse, {
        'conditional append successful': (r) => r.status === 200,
        'conditional response has duration': (r) => r.json('durationInMicroseconds') > 0,
    });

    // Test 3: Read back events
    const readPayload = {
        query: {
            items: [{ types: ['SimpleEvent', 'ConditionalEvent'], tags: [`${level.name.toLowerCase()}:${__VU}`] }]
        }
    };
    const readRes = http.post(`${BASE_URL}/read`, JSON.stringify(readPayload), params);
    const readCheck = check(readRes, {
        'read successful': (r) => r.status === 200,
        'read returns events': (r) => r.json('numberOfMatchingEvents') > 0,
    });

    readSuccess.add(readCheck);

    sleep(0.1);
} 