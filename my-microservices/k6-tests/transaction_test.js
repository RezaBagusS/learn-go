import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Global variables
const BASE_URL_ACCOUNT = 'http://localhost:8082';
const BASE_URL_TRANSACTION = 'http://localhost:8083';

const PARTNER_ID = 'K6-PARTNER-001';
const CHANNEL_ID = 'WEB-APP';

// Config
export const options = {
    vus: 10,
    duration: '30s',
    thresholds: {
        http_req_duration: ['p(95)<200'], // 95% of requests should be < 200ms
        http_req_failed: ['rate<0.01'], // Error rate < 1%
    },
};

export function setup() {
    // Persiapan Data: Menyiapkan lingkungan atau data yang dibutuhkan sebelum tes dimulai. Contohnya: melakukan login admin untuk mendapatkan daftar akun, membuat data simulasi di database, atau memastikan server tujuan sudah aktif.
    // Efisiensi: Jika Anda butuh daftar 100 akun untuk dites, lebih baik mengambilnya sekali di setup daripada menyuruh setiap VU mengambil data yang sama berulang kali di fungsi default.
    // Data Sharing: Apa pun yang Anda return di fungsi setup akan dikirimkan sebagai argumen ke fungsi default (fungsi utama pengujian).
}

function getHeaders(token, accountId = '') {
    return {
        'Content-Type': 'application/json',
        'X-Timestamp': new Date().toISOString(),
        'X-Partner-ID': PARTNER_ID,
        'X-External-ID': uuidv4(),
        'X-Channel-ID': CHANNEL_ID,
        'X-ACCOUNT-ID': accountId,
        'Authorization': `Bearer ${token}`
    };
}

export default function () {
    // Dapatkan Token 
    const dummyAccountID = '98555e71-92df-4235-9856-f41852cc5bc3';

    const tokenRes = http.post(`${BASE_URL_ACCOUNT}/v1.0/access-token`, JSON.stringify({}), {
        headers: { 'X-ACCOUNT-ID': dummyAccountID }
    });

    if (!check(tokenRes, { 'token response is 200': (r) => r.status === 200 })) {
        console.error(`Failed to get token: ${tokenRes.body}`);
        return;
    }

    const token = tokenRes.json().data.token;

    // Get list akun
    const accountsRes = http.get(`${BASE_URL_ACCOUNT}/v1.0/accounts`, {
        headers: getHeaders(token, dummyAccountID)
    });

    check(accountsRes, { 'get accounts success': (r) => r.status === 200 });

    const accountList = accountsRes.json().data.accounts;
    if (accountList && accountList.length >= 2) {
        const source = accountList[0];
        const beneficiary = accountList[1];

        // Execute Transfer (Intrabank)
        const transferPayload = JSON.stringify({
            partnerReferenceNo: `TRX-${uuidv4().substring(0, 8)}`,
            amount: {
                value: "1000.00",
                currency: "IDR"
            },
            beneficiaryAccountNo: beneficiary.account_no,
            sourceAccountNo: source.account_no,
            transactionDate: new Date().toISOString(),
            additionalInfo: {}
        });

        const transferRes = http.post(`${BASE_URL_TRANSACTION}/v1.0/transfer-intrabank`, transferPayload, {
            headers: getHeaders(token, source.id)
        });

        check(transferRes, {
            'transfer success': (r) => r.status === 200,
            'transfer error code is 2000000': (r) => r.json().responseCode === '2000000'
        });
    } else {
        // Saldo ga cukup, force Topup 
        const topupPayload = JSON.stringify({
            partnerReferenceNo: `TOP-${uuidv4().substring(0, 8)}`,
            amount: {
                value: "50000.00",
                currency: "IDR"
            },
            sourceAccountNo: "1234567890",
            additionalInfo: {}
        });

        const topupRes = http.post(`${BASE_URL_TRANSACTION}/v1.0/topup`, topupPayload, {
            headers: getHeaders(token, dummyAccountID)
        });

        check(topupRes, { 'topup success': (r) => r.status === 200 });
    }

    sleep(1);
}

// k6 run k6-tests/transaction_test.js
