import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
    // 1. Initial configuration for load testing
    stages: [
        { duration: '30s', target: 50 },  // Ramp-up to 50 virtual users over 30s
        { duration: '1m', target: 50 },   // Stay at 50 users for 1 minute
        { duration: '30s', target: 0 },   // Ramp-down to 0 users over 30s
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
        http_req_failed: ['rate<0.01'],   // Error rate must be less than 1%
    },
};

const BASE_URL = 'http://localhost:8080/api/v1';

export default function () {
    const customerId = '550e8400-e29b-41d4-a716-446655440001';

    // ─────────────────────────────────────────────────────────────
    // 1. Get Payment Methods
    // ─────────────────────────────────────────────────────────────
    let resMethods = http.get(`${BASE_URL}/payment-methods`);
    check(resMethods, {
        'GET /payment-methods status is 200': (r) => r.status === 200,
    });
    sleep(1);

    // ─────────────────────────────────────────────────────────────
    // 2. Charge Transaction (Create Payment)
    // ─────────────────────────────────────────────────────────────
    // Unique Idempotency Key per user iteration
    const idempotencyKey = uuidv4();
    
    const payload = JSON.stringify({
        customer_id: customerId,
        payment_method_id: 'qris',
        amount: 150000,
        currency: 'IDR'
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'X-Idempotency-Key': idempotencyKey,
        },
    };

    let resCharge = http.post(`${BASE_URL}/payments/charge`, payload, params);
    check(resCharge, {
        'POST /payments/charge status is 201': (r) => r.status === 201,
        'has order_id': (r) => r.json('data.order_id') !== undefined,
    });

    // ─────────────────────────────────────────────────────────────
    // 3. Get Transaction Status
    // ─────────────────────────────────────────────────────────────
    let orderId = resCharge.json('data.order_id');
    if (orderId) {
        sleep(1); // Wait a bit before checking status
        let resStatus = http.get(`${BASE_URL}/payments/${orderId}`);
        check(resStatus, {
            'GET /payments/:order_id status is 200': (r) => r.status === 200,
            'status is pending': (r) => r.json('data.status') === 'pending',
        });
    }

    // ─────────────────────────────────────────────────────────────
    // 4. List Transaction History
    // ─────────────────────────────────────────────────────────────
    sleep(1);
    let resHistory = http.get(`${BASE_URL}/payments?limit=10&offset=0&customer_id=${customerId}`);
    check(resHistory, {
        'GET /payments (history) status is 200': (r) => r.status === 200,
    });

    sleep(1);
}
