import assert from 'node:assert/strict';
import test from 'node:test';

import { nestedApiRunnerFixtureValue } from './fixtures/index-import';

test('API runner executes nested ESM suites and resolves directory index imports', () => {
  assert.equal(import.meta.url.startsWith('file:'), true);
  assert.equal(nestedApiRunnerFixtureValue, 'resolved-through-index');
});
