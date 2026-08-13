import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildSharedReadCacheKey,
  shouldCacheSharedReadResponse,
} from '../smoke/helpers/shared-read-cache.ts';

test('buildSharedReadCacheKey normalizes query ordering for identical read requests', () => {
  assert.equal(
    buildSharedReadCacheKey('http://127.0.0.1:5173/api/v1/system/menu/tree?scope=manage&activeOnly=true'),
    buildSharedReadCacheKey('http://127.0.0.1:5173/api/v1/system/menu/tree?activeOnly=true&scope=manage'),
  );
  assert.equal(
    buildSharedReadCacheKey('http://127.0.0.1:5173/api/v1/system/menu/tree?scope=manage&activeOnly=true'),
    '/api/v1/system/menu/tree?activeOnly=true&scope=manage',
  );
});

test('shouldCacheSharedReadResponse only stores successful GET reads', () => {
  assert.equal(shouldCacheSharedReadResponse('GET', 200), true);
  assert.equal(shouldCacheSharedReadResponse('GET', 204), true);
  assert.equal(shouldCacheSharedReadResponse('POST', 200), false);
  assert.equal(shouldCacheSharedReadResponse('GET', 429), false);
  assert.equal(shouldCacheSharedReadResponse('GET', 500), false);
});

test('shared read cache key keeps distinct endpoints distinct even with identical query params', () => {
  assert.notEqual(
    buildSharedReadCacheKey('http://127.0.0.1:5173/api/v1/system/setting/public?scope=nav'),
    buildSharedReadCacheKey('http://127.0.0.1:5173/api/v1/system/menu/tree?scope=nav'),
  );
});
