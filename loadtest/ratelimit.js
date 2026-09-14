import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const acceptedEvents = new Counter('events_accepted_200');
const rateLimitedEvents = new Counter('events_rate_limited_429');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:5000';

export const options = {
  // Single VU flooding requests to hit the 100 req/min limit quickly
  vus: 1,
  iterations: 130,
  thresholds: {
    events_accepted_200: ['count>=100'],
    events_rate_limited_429: ['count>=1'],
  },
};

export function setup() {
  const email = `ratelimit_target_${Date.now()}@example.com`;
  const password = 'Password123!';

  const res = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({ email, password }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const body = JSON.parse(res.body);
  return { token: body.access_token };
}

export default function (data) {
  const payload = JSON.stringify({
    event_type: 'signup_thankyou',
    recipient: 'test@example.com',
    payload: {
      name: 'RateLimit Test',
      year: 2026,
    },
  });

  const res = http.post(`${BASE_URL}/api/v1/event`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${data.token}`,
    },
  });

  if (res.status === 200) {
    acceptedEvents.add(1);
  } else if (res.status === 429) {
    rateLimitedEvents.add(1);
  }

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });

  // Rapidly fire without sleep to breach rate limit
  sleep(0.01);
}
