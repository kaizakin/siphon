import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('auth_errors');
const authDuration = new Trend('auth_duration_ms');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:5000';

export const options = {
  stages: [
    { duration: '10s', target: 5 },
    { duration: '20s', target: 15 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // bcrypt password hashing takes more CPU
    auth_errors: ['rate<0.05'],
  },
};

export default function () {
  const email = `auth_test_${__VU}_${__ITER}_${Date.now()}@example.com`;
  const password = 'SecretPassword123!';

  // 1. Register
  const registerPayload = JSON.stringify({ email, password });
  const regRes = http.post(`${BASE_URL}/api/v1/auth/register`, registerPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const regOk = check(regRes, {
    'register status is 201': (r) => r.status === 201,
    'register has access_token': (r) => {
      try {
        return Boolean(JSON.parse(r.body).access_token);
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!regOk);
  authDuration.add(regRes.timings.duration);

  sleep(0.3);

  // 2. Login
  const loginPayload = JSON.stringify({ email, password });
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  let refreshToken = null;
  const loginOk = check(loginRes, {
    'login status is 202 or 200': (r) => r.status === 202 || r.status === 200,
    'login has refresh_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        refreshToken = body.refresh_token;
        return Boolean(refreshToken);
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!loginOk);
  authDuration.add(loginRes.timings.duration);

  sleep(0.3);

  // 3. Refresh Token Rotation
  if (refreshToken) {
    const refreshPayload = JSON.stringify({ refresh_token: refreshToken });
    const refreshRes = http.post(`${BASE_URL}/api/v1/auth/refresh`, refreshPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    const refreshOk = check(refreshRes, {
      'refresh status is 200': (r) => r.status === 200,
      'refresh returns new tokens': (r) => {
        try {
          const body = JSON.parse(r.body);
          return Boolean(body.access_token && body.refresh_token);
        } catch {
          return false;
        }
      },
    });

    errorRate.add(!refreshOk);
    authDuration.add(refreshRes.timings.duration);
  }

  sleep(0.5);
}
