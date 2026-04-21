import http from 'k6/http';
import { check, sleep } from 'k6';

// Config
const BASE_URL_ACCOUNT = 'http://localhost:8082';
const ACCOUNT_ID = '1083b783-1085-441c-8a74-9d40ec3429a6';
const VERSION = "v1.0"

export const options = {
    stages: [
        { duration: '1m', target: 50 },  // Traffic normal
        { duration: '2m', target: 100 }, // Stress test
        { duration: '2m', target: 200 }, // Breaking point?
        { duration: '1m', target: 0 },   // Scale down
    ],
    thresholds: {
        http_req_failed: ['rate<0.05'], // Max 5% errors
    },
};

export default function () {
    const headers = {
        'X-ACCOUNT-ID': ACCOUNT_ID,
        'Content-Type': 'application/json'
    };

    // Stress test target: Access Token endpoint (might be heavily hit)
    const res = http.post(`${BASE_URL_ACCOUNT}/api/${VERSION}/access-token`, JSON.stringify({}), { headers });

    check(res, {
        'status is 200': (r) => r.status === 200,
    });

    sleep(0.5);
}

// k6 run k6-tests/stress_test.js
