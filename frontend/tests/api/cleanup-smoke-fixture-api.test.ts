import assert from 'node:assert/strict';
import test from 'node:test';

import {
  fetchCleanupJson,
  getCleanupGetRetryDelayMs,
  getCleanupGetRetryOptions,
} from '../../scripts/lib/cleanup-http.mjs';

function createJsonResponse(status: number, payload: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return payload;
    },
  };
}

test('fetchCleanupJson retries rate-limited GET responses before succeeding', async () => {
  const calls: number[] = [];
  const waits: number[] = [];
  const fetchImpl = async () => {
    calls.push(calls.length + 1);
    if (calls.length === 1) {
      return createJsonResponse(429, { code: 429 });
    }
    return createJsonResponse(200, { code: 200, data: { items: [1, 2, 3] } });
  };

  const data = await fetchCleanupJson(fetchImpl, 'http://127.0.0.1:8080/api/v1/system/dict/type/list', {
    path: '/system/dict/type/list',
    sleep: async (ms) => {
      waits.push(ms);
    },
  });

  assert.deepEqual(data, { items: [1, 2, 3] });
  assert.equal(calls.length, 2);
  assert.deepEqual(waits, [getCleanupGetRetryDelayMs(1)]);
});

test('fetchCleanupJson surfaces the final rate-limit failure after retry budget is exhausted', async () => {
  const waits: number[] = [];
  const fetchImpl = async () => createJsonResponse(429, { code: 429 });

  await assert.rejects(
    () =>
      fetchCleanupJson(fetchImpl, 'http://127.0.0.1:8080/api/v1/system/dict/type/list', {
        path: '/system/dict/type/list',
        maxAttempts: 2,
        sleep: async (ms) => {
          waits.push(ms);
        },
      }),
    /GET \/system\/dict\/type\/list failed: HTTP 429/,
  );

  assert.deepEqual(waits, [getCleanupGetRetryDelayMs(1)]);
});

test('post-phase cleanup expands retry budget to absorb transient rate limits', async () => {
  const waits: number[] = [];
  let attempts = 0;
  const fetchImpl = async () => {
    attempts += 1;
    if (attempts < 5) {
      return createJsonResponse(429, { code: 429 });
    }
    return createJsonResponse(200, { code: 200, data: { items: ['ok'] } });
  };

  const data = await fetchCleanupJson(
    fetchImpl,
    'http://127.0.0.1:8080/api/v1/system/i18n/list?key=i18n.smoke.&page=1&pageSize=500',
    {
      path: '/system/i18n/list',
      ...getCleanupGetRetryOptions('post'),
      sleep: async (ms) => {
        waits.push(ms);
      },
    },
  );

  assert.deepEqual(data, { items: ['ok'] });
  assert.equal(attempts, 5);
  assert.deepEqual(waits, [
    getCleanupGetRetryDelayMs(1, 'post'),
    getCleanupGetRetryDelayMs(2, 'post'),
    getCleanupGetRetryDelayMs(3, 'post'),
    getCleanupGetRetryDelayMs(4, 'post'),
  ]);
});
