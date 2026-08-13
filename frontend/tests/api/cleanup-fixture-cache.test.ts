import assert from 'node:assert/strict';
import test from 'node:test';

import { createCleanupFixtureCache } from '../../scripts/lib/cleanup-fixture-cache.mjs';

test('createCleanupFixtureCache reuses the same dict type snapshot within one cleanup run', async () => {
  const cache = createCleanupFixtureCache();
  let loads = 0;

  const first = await cache.getDictTypes(async () => {
    loads += 1;
    return [{ id: 1, dictCode: 'system_sync_1' }];
  });
  const second = await cache.getDictTypes(async () => {
    loads += 1;
    return [{ id: 2, dictCode: 'system_sync_2' }];
  });

  assert.equal(loads, 1);
  assert.deepEqual(first, [{ id: 1, dictCode: 'system_sync_1' }]);
  assert.deepEqual(second, first);
});

test('createCleanupFixtureCache can invalidate the dict type snapshot after mutations', async () => {
  const cache = createCleanupFixtureCache();
  let loads = 0;

  await cache.getDictTypes(async () => {
    loads += 1;
    return [{ id: 1, dictCode: 'system_sync_1' }];
  });

  cache.clearDictTypes();

  const next = await cache.getDictTypes(async () => {
    loads += 1;
    return [{ id: 2, dictCode: 'system_sync_2' }];
  });

  assert.equal(loads, 2);
  assert.deepEqual(next, [{ id: 2, dictCode: 'system_sync_2' }]);
});

test('createCleanupFixtureCache reuses one i18n list snapshot per query key', async () => {
  const cache = createCleanupFixtureCache();
  let loads = 0;

  const first = await cache.getI18nList('i18n.smoke.', async () => {
    loads += 1;
    return { items: [{ id: 1, key: 'i18n.smoke.a' }] };
  });
  const second = await cache.getI18nList('i18n.smoke.', async () => {
    loads += 1;
    return { items: [{ id: 2, key: 'i18n.smoke.b' }] };
  });
  const third = await cache.getI18nList('i18n.sync.', async () => {
    loads += 1;
    return { items: [{ id: 3, key: 'i18n.sync.a' }] };
  });

  assert.equal(loads, 2);
  assert.deepEqual(first, second);
  assert.deepEqual(third, { items: [{ id: 3, key: 'i18n.sync.a' }] });
});
