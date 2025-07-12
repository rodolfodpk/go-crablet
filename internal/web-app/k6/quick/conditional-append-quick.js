import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Custom metrics
const appendIfSuccess = new Rate('append_if_success');
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
        http_req_failed: ['rate<0.10'],     // Less than 10% errors
        http_req_duration: ['p(95)<1000'],  // 95% of requests under 1000ms
        append_if_success: ['rate>0.95'],   // 95% conditional append success rate
        read_success: ['rate>0.95'],        // 95% read success rate
    },
};

export function setup() {
    // Clean database
    const cleanupRes = http.post(`${BASE_URL}/cleanup`);
    check(cleanupRes, {
        'cleanup successful': (r) => r.status === 200,
    });

    // Test basic conditional append for each isolation level
    for (const level of ISOLATION_LEVELS) {
        const testEvent = {
            type: 'TestEvent',
            data: JSON.stringify({ message: `Test conditional append with ${level.name} isolation` }),
            metadata: { timestamp: Date.now() },
            tags: [`test:conditional-${level.name.toLowerCase()}:setup`]
        };
        
        const testPayload = {
            events: testEvent,
            condition: {
                expectedVersion: 0
            }
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
            [`setup ${level.name} conditional append successful`]: (r) => r.status === 200,
            [`setup ${level.name} response has duration`]: (r) => r.json('durationInMicroseconds') > 0,
            [`setup ${level.name} condition not failed`]: (r) => r.json('appendConditionFailed') === false,
        });

        // Test read back
        const readPayload = {
            query: {
                items: [{ types: ['TestEvent'], tags: [`test:conditional-${level.name.toLowerCase()}:setup`] }]
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
    const streamId = `conditional-${level.name.toLowerCase()}-${__VU}-${__ITER}`;
    
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

    // Test 1: Conditional append with expectedVersion 0 (should succeed)
    const conditionalEvent = {
        type: 'ConditionalEvent',
        data: JSON.stringify({ value: Math.random(), isolation: level.name, condition: 'expectedVersion_0' }),
        metadata: { timestamp: Date.now() },
        tags: [`conditional-${level.name.toLowerCase()}:${__VU}`]
    };
    
    const conditionalPayload = {
        events: conditionalEvent,
        condition: {
            expectedVersion: 0
        }
    };

    const response = http.post(
        `${BASE_URL}/append`,
        JSON.stringify(conditionalPayload),
        params
    );

    const appendIfCheck = check(response, {
        'conditional append successful': (r) => r.status === 200,
        'response has duration': (r) => r.json('durationInMicroseconds') > 0,
        'condition not failed': (r) => r.json('appendConditionFailed') === false,
    });

    appendIfSuccess.add(appendIfCheck);

    // Test 2: Conditional append with expectedVersion 1 (should fail)
    const conditionalEvent2 = {
        type: 'ConditionalEvent',
        data: JSON.stringify({ value: Math.random(), isolation: level.name, condition: 'expectedVersion_1' }),
        metadata: { timestamp: Date.now() },
        tags: [`conditional-${level.name.toLowerCase()}:${__VU}`]
    };
    
    const conditionalPayload2 = {
        events: conditionalEvent2,
        condition: {
            expectedVersion: 1
        }
    };

    const response2 = http.post(
        `${BASE_URL}/append`,
        JSON.stringify(conditionalPayload2),
        params
    );

    check(response2, {
        'conditional append with wrong version returns 200': (r) => r.status === 200,
        'condition failed as expected': (r) => r.json('appendConditionFailed') === true,
    });

    // Test 3: Read back events
    const readPayload = {
        query: {
            items: [{ types: ['ConditionalEvent'], tags: [`conditional-${level.name.toLowerCase()}:${__VU}`] }]
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