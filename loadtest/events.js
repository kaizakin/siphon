import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const eventDuration = new Trend("event_duration_ms");

const BASE_URL = __ENV.BASE_URL || "http://localhost:5000";

export const options = {
  stages: [
    { duration: "15s", target: 10 }, // Ramp up to 10 VUs
    { duration: "30s", target: 20 }, // Steady load at 20 VUs
    { duration: "15s", target: 0 }, // Ramp down to 0 VUs
  ],
  thresholds: {
    http_req_duration: ["p(95)<300"], // 95% of requests should be below 300ms
    errors: ["rate<0.05"], // Error rate must be under 5%
  },
};

const EVENT_TEMPLATES = [
  {
    type: "order_success",
    payload: (i) => ({
      name: `User ${i}`,
      order_id: `ORD-${Date.now()}-${i}`,
      amount: 49.99,
      year: new Date().getFullYear(),
    }),
  },
  {
    type: "signup_thankyou",
    payload: (i) => ({
      name: `User ${i}`,
      year: new Date().getFullYear(),
    }),
  },
  {
    type: "payment_success",
    payload: (i) => ({
      name: `User ${i}`,
      transaction_id: `TXN-${Date.now()}-${i}`,
      amount: 120.5,
      year: new Date().getFullYear(),
    }),
  },
  {
    type: "order_failed",
    payload: (i) => ({
      name: `User ${i}`,
      order_id: `ORD-${Date.now()}-${i}`,
    }),
  },
];

function generateUUID() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function setup() {
  // Pre-flight check against base URL
  const ping = http.get(`${BASE_URL}/api/v1/auth/login`);
  return { baseUrl: BASE_URL };
}

// Each VU registers a unique user on start to avoid getting rate-limited by the per-user limit
let token = null;

function authenticate(vuId) {
  const email = `loadtest_vu_${vuId}_${Date.now()}@example.com`;
  const password = "Password123!";

  const registerRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({ email, password }),
    { headers: { "Content-Type": "application/json" } },
  );

  if (registerRes.status === 201) {
    const body = JSON.parse(registerRes.body);
    return body.access_token;
  }

  // Fallback to login if already exists
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email, password }),
    { headers: { "Content-Type": "application/json" } },
  );

  if (loginRes.status === 202 || loginRes.status === 200) {
    const body = JSON.parse(loginRes.body);
    return body.access_token;
  }

  return null;
}

export default function () {
  if (!token) {
    token = authenticate(__VU);
  }

  if (!token) {
    errorRate.add(1);
    sleep(1);
    return;
  }

  const selectedTemplate =
    EVENT_TEMPLATES[Math.floor(Math.random() * EVENT_TEMPLATES.length)];

  const eventPayload = {
    event_type: selectedTemplate.type,
    recipient: `customer_${__VU}_${__ITER}@example.com`,
    payload: selectedTemplate.payload(__ITER),
  };

  const idempotencyKey = generateUUID();

  const params = {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      "Idempotency-Key": idempotencyKey,
    },
  };

  const res = http.post(
    `${BASE_URL}/api/v1/event`,
    JSON.stringify(eventPayload),
    params,
  );

  const isSuccess = check(res, {
    "event status is 200": (r) => r.status === 200,
    "has event_id": (r) => {
      try {
        const body = JSON.parse(r.body);
        return Boolean(body.event_id);
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!isSuccess);
  eventDuration.add(res.timings.duration);

  // Sleep between requests to maintain paced throughput per VU
  sleep(0.5);
}
