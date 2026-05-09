import http from 'k6/http';
import { check, sleep } from 'k6';
import crypto from 'k6/crypto';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import dotenv from 'https://jslib.k6.io/dotenv/2.0.0/index.js';

// Parse the .env file located at the project root
const env = dotenv.parse(open('../.env'));

export const options = {
    // Stress test: aggressively ramp up to 100 concurrent webhook deliveries
    stages: [
        { duration: '10s', target: 50 },
        { duration: '30s', target: 100 }, 
        { duration: '10s', target: 0 }, 
    ],
    thresholds: {
        http_req_duration: ['p(95)<1000'], // Webhooks write to DB + Kafka, allow up to 1s
        http_req_failed: ['rate<0.01'],    // Should rarely fail
    },
};

const BASE_URL = __ENV.BASE_URL || env.APP_PORT ? `http://localhost:${env.APP_PORT}/api/v1` : 'http://localhost:8080/api/v1';
// Read from parsed .env directly
const SERVER_KEY = __ENV.MIDTRANS_SERVER_KEY || env.MIDTRANS_SERVER_KEY || 'SB-Mid-server-YOUR_SERVER_KEY'; 

export default function () {
    // 1. Generate realistic fake Midtrans data
    const orderId = `ORDER-qris-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    const transactionId = `midtrans-${uuidv4()}`;
    const statusCode = '200';
    const grossAmount = '100000.00';

    // 2. Generate valid SHA512 signature so the app doesn't reject it (401)
    // Formula: order_id + status_code + gross_amount + server_key
    const payloadStr = orderId + statusCode + grossAmount + SERVER_KEY;
    const signature = crypto.sha512(payloadStr, 'hex');

    const payload = JSON.stringify({
        transaction_id: transactionId,
        order_id: orderId,
        gross_amount: grossAmount,
        status_code: statusCode,
        fraud_status: 'accept',
        transaction_status: 'settlement',
        payment_type: 'qris',
        signature_key: signature
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    // 3. Fire the webhook
    let res = http.post(`${BASE_URL}/payments/webhook/midtrans`, payload, params);
    
    // Note: If the order_id doesn't actually exist in the DB, our app will return an error (404/500).
    // To make this a pure stress test of the DB Lock & Kafka Producer, you should pre-seed the DB 
    // or accept that it might fail with "Transaction Not Found". 
    // For this example, we just check that the server responds quickly, even if it's a 404/500 from the app layer.
    
    check(res, {
        'Webhook responds quickly': (r) => r.timings.duration < 1000,
    });

    // Small sleep to not completely DDoS the local machine
    sleep(0.5);
}