import http from 'k6/http';
import { check } from 'k6';

export const options = {
    vus: 1,
    iterations: 1000,
    maxDuration: '10m',
    gracefulStop: '30s',
};

export default function () {
    const response = http.get('http://localhost:8080/health');
    
    check(response, {
        'status is 200': (r) => r.status === 200,
        'body has ok': (r) => r.body.includes('ok'),
    });
} 