import assert from 'node:assert/strict';
import test from 'node:test';

import { compareStrings, sortStrings } from '../../scripts/harness/sort-utils.mjs';

test('sortStrings returns a locale-aware sorted copy without mutating input', () => {
  const values = ['zeta', 'alpha', 'beta'];
  const original = [...values];

  const sorted = sortStrings(values);

  assert.deepEqual(sorted, ['alpha', 'beta', 'zeta']);
  assert.deepEqual(values, original);
});

test('compareStrings matches localeCompare semantics for string inputs', () => {
  assert.equal(compareStrings('a', 'b'), 'a'.localeCompare('b'));
  assert.equal(compareStrings('b', 'a'), 'b'.localeCompare('a'));
  assert.equal(compareStrings('same', 'same'), 0);
});
