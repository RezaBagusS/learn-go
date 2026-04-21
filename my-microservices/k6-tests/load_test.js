import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Global variables
const BASE_URL_ACCOUNT = 'http://localhost:8082';
const BASE_URL_TRANSACTION = 'http://localhost:8083';
const VERSION = "v1.0"

const PARTNER_ID = 'PARTNER-TEST-001';
const CHANNEL_ID = 'K6-LOAD-TEST';
const ACCOUNT_ID = '1083b783-1085-441c-8a74-9d40ec3429a6';

// Options load test
export const options = {
    stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 20 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'],
        http_req_failed: ['rate<0.01'],
    },
};

// SNAP headers
function getSnapHeaders(token = null) {
    const timestamp = new Date().toISOString();
    const externalId = uuidv4();

    let headers = {
        'Content-Type': 'application/json',
        'X-Timestamp': timestamp,
        'X-Partner-ID': PARTNER_ID,
        'X-External-ID': externalId,
        'X-Channel-ID': CHANNEL_ID,
        'X-ACCOUNT-ID': ACCOUNT_ID,
    };

    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    return headers;
}

export default function () {
    // Get Access Token
    const tokenRes = http.post(
        `${BASE_URL_ACCOUNT}/api/${VERSION}/access-token`,
        JSON.stringify({}),
        { headers: { 'X-ACCOUNT-ID': ACCOUNT_ID } }
    );

    console.log(tokenRes.body);

    check(tokenRes, {
        'get token status is 200': (r) => r.status === 200,
        'token is present': (r) => r.json().token !== undefined,
    });

    const token = tokenRes.json().token;

    // Get All Accounts
    const accountsRes = http.get(
        `${BASE_URL_ACCOUNT}/api/${VERSION}/accounts`,
        { headers: getSnapHeaders(token) }
    );

    check(accountsRes, {
        'get accounts status is 200': (r) => r.status === 200,
    });

    // Topup 
    const topupPayload = JSON.stringify({
        partner_reference_no: `TOPUP-${uuidv4().substring(0, 8)}`,
        amount: {
            value: "100000.00",
            currency: "IDR"
        },
        source_account_no: accountsRes.json().accounts[0].account_number, // Ganti dengan nomor rekening valid
        additional_info: {
            deviceId: "DEVICE-WEB-TEST",
            channel: "MOBILE"
        }
    });

    const topupRes = http.post(
        `${BASE_URL_TRANSACTION}/api/${VERSION}/topup`,
        topupPayload,
        { headers: getSnapHeaders(token) }
    );

    check(topupRes, {
        'topup status is 200': (r) => r.status === 200,
    });

    // Get Transaction History
    const historyRes = http.get(
        `${BASE_URL_TRANSACTION}/api/${VERSION}/transactions`,
        { headers: getSnapHeaders(token) }
    );

    check(historyRes, {
        'get transactions status is 200': (r) => r.status === 200,
    });

    sleep(1);
}


// k6 run k6-tests/load_test.js