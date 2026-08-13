import assert from 'node:assert/strict';
import test from 'node:test';

import { buildCleanupI18nQueries } from '../../scripts/lib/cleanup-fixture-query-plan.mjs';

test('buildCleanupI18nQueries compacts smoke prefixes into a minimal query plan', () => {
  const queries = buildCleanupI18nQueries({
    prefixes: [
      'i18n.enter.',
      'i18n.smoke.',
      'i18n.import.',
      'i18n.sync.',
      'dict.smoke_biz_status.',
      'dict.smoke_batch_delete_dict_.',
    ],
    exacts: ['system.smoke'],
  });

  assert.deepEqual(queries, ['i18n.', 'dict.smoke', 'system.smoke']);
});
