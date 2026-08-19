import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildRefreshTopicKey,
  canonicalizeRefreshTopics,
} from '../../src/core/refresh/topicKey.ts';
import { shouldPublishRefreshForVersionChange } from '../../src/core/refresh/versioning.ts';

test('shouldPublishRefreshForVersionChange suppresses the first observed snapshot', () => {
  assert.equal(shouldPublishRefreshForVersionChange(undefined, 0), false);
  assert.equal(shouldPublishRefreshForVersionChange(undefined, 1), false);
});

test('shouldPublishRefreshForVersionChange publishes after a tracked topic advances from zero', () => {
  assert.equal(shouldPublishRefreshForVersionChange(0, 1), true);
});

test('shouldPublishRefreshForVersionChange only publishes on monotonic increases', () => {
  assert.equal(shouldPublishRefreshForVersionChange(3, 3), false);
  assert.equal(shouldPublishRefreshForVersionChange(3, 2), false);
  assert.equal(shouldPublishRefreshForVersionChange(3, 4), true);
});

test('canonicalizeRefreshTopics deduplicates and stabilizes topic ordering', () => {
  assert.deepEqual(canonicalizeRefreshTopics(['system:role:changed', 'system:user:changed', 'system:role:changed']), [
    'system:role:changed',
    'system:user:changed',
  ]);
  assert.equal(
    buildRefreshTopicKey(['system:user:changed', 'system:role:changed', 'system:user:changed']),
    'system:role:changed,system:user:changed',
  );
});
