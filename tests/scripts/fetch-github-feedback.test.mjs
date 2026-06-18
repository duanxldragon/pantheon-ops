import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/fetch-github-feedback.mjs'),
).href;

const {
  buildFeedbackSnapshot,
  classifyFeedbackItem,
  evaluateFeedbackGate,
  extractGitHubReferences,
} = await import(moduleUrl);

test('extractGitHubReferences keeps same-repo issue and discussion urls only once', () => {
  const references = extractGitHubReferences(
    `
See https://github.com/acme/demo/issues/12
and https://github.com/acme/demo/discussions/7
and again https://github.com/acme/demo/issues/12
but ignore https://github.com/acme/other/issues/99
`,
    'acme/demo',
  );

  assert.deepEqual(references, {
    issues: [{ number: 12, url: 'https://github.com/acme/demo/issues/12' }],
    discussions: [{ number: 7, url: 'https://github.com/acme/demo/discussions/7' }],
  });
});

test('extractGitHubReferences also recognizes same-repo issue shorthand from PR close directives', () => {
  const references = extractGitHubReferences(
    `
Closes #12
fixes #12
Resolves #34
but ignore closes #not-a-number
`,
    'acme/demo',
  );

  assert.deepEqual(references, {
    issues: [
      { number: 12, url: 'https://github.com/acme/demo/issues/12' },
      { number: 34, url: 'https://github.com/acme/demo/issues/34' },
    ],
    discussions: [],
  });
});

test('classifyFeedbackItem marks resolved feedback as closed', () => {
  const result = classifyFeedbackItem({
    sourceType: 'pull-request-review',
    status: 'resolved',
    body: 'Fixed in the latest patch.',
  });

  assert.equal(result.classification, 'closed');
  assert.equal(result.recommendedAction, 'skip');
  assert.equal(result.confidence, 'high');
});

test('classifyFeedbackItem marks imperative review feedback as actionable', () => {
  const result = classifyFeedbackItem({
    sourceType: 'pull-request-review',
    status: 'open',
    body: 'Please add a regression test for this auth branch.',
  });

  assert.equal(result.classification, 'actionable-change');
  assert.equal(result.recommendedAction, 'change');
});

test('classifyFeedbackItem marks question-only feedback as explanation-only', () => {
  const result = classifyFeedbackItem({
    sourceType: 'issue-comment',
    status: 'open',
    body: 'Why is this kept in ops instead of base?',
  });

  assert.equal(result.classification, 'explanation-only');
  assert.equal(result.recommendedAction, 'reply');
});

test('buildFeedbackSnapshot flattens review threads, issue comments, and discussion replies', () => {
  const snapshot = buildFeedbackSnapshot({
    repoFullName: 'acme/demo',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
      body: 'Related issue https://github.com/acme/demo/issues/12 and discussion https://github.com/acme/demo/discussions/7',
      reviewThreads: [
        {
          id: 'thread-1',
          isResolved: false,
          isOutdated: false,
          comments: [
            {
              id: 'pr-comment-1',
              body: 'Please update the PR gate docs.',
              url: 'https://github.com/acme/demo/pull/42#discussion_r1',
              createdAt: '2026-06-17T00:00:00Z',
              author: { login: 'reviewer-1' },
            },
          ],
        },
        {
          id: 'thread-2',
          isResolved: true,
          isOutdated: false,
          comments: [
            {
              id: 'pr-comment-2',
              body: 'Looks good now.',
              url: 'https://github.com/acme/demo/pull/42#discussion_r2',
              createdAt: '2026-06-17T00:01:00Z',
              author: { login: 'reviewer-2' },
            },
          ],
        },
      ],
    },
    issues: [
      {
        number: 12,
        url: 'https://github.com/acme/demo/issues/12',
        title: 'Track review cleanup',
        state: 'OPEN',
        comments: [
          {
            id: 'issue-comment-1',
            body: 'Can you explain why this check is base-first?',
            url: 'https://github.com/acme/demo/issues/12#issuecomment-1',
            createdAt: '2026-06-17T00:02:00Z',
            author: { login: 'ops-user' },
          },
        ],
      },
    ],
    discussions: [
      {
        number: 7,
        url: 'https://github.com/acme/demo/discussions/7',
        title: 'Should this live in base?',
        isAnswered: true,
        comments: [
          {
            id: 'discussion-comment-1',
            body: 'This belongs in a follow-up PR.',
            url: 'https://github.com/acme/demo/discussions/7#discussioncomment-1',
            createdAt: '2026-06-17T00:03:00Z',
            author: { login: 'maintainer' },
            isAnswer: false,
            replies: [
              {
                id: 'discussion-reply-1',
                body: 'Answered by the base fix.',
                url: 'https://github.com/acme/demo/discussions/7#discussioncomment-2',
                createdAt: '2026-06-17T00:04:00Z',
                author: { login: 'maintainer' },
                isAnswer: true,
              },
            ],
          },
        ],
      },
    ],
  });

  assert.equal(snapshot.items.length, 5);
  assert.deepEqual(snapshot.linked, {
    issues: [{ number: 12, url: 'https://github.com/acme/demo/issues/12' }],
    discussions: [{ number: 7, url: 'https://github.com/acme/demo/discussions/7' }],
  });
  assert.equal(snapshot.summary.totalItems, 5);
  assert.equal(snapshot.summary.sourceCounts['pull-request-review'], 2);
  assert.equal(snapshot.summary.sourceCounts['issue-comment'], 1);
  assert.equal(snapshot.summary.sourceCounts['discussion-comment'], 2);
  assert.equal(snapshot.items[0].commentDatabaseId, null);
  assert.equal(snapshot.summary.classificationCounts['actionable-change'], 1);
  assert.equal(snapshot.summary.classificationCounts.closed, 3);
  assert.equal(snapshot.summary.classificationCounts['explanation-only'], 1);
  assert.equal(snapshot.summary.classificationCounts['out-of-scope'] ?? 0, 0);
});

test('buildFeedbackSnapshot preserves PR review comment database ids for reply writeback', () => {
  const snapshot = buildFeedbackSnapshot({
    repoFullName: 'acme/demo',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
      body: '',
      reviewThreads: [
        {
          id: 'thread-1',
          isResolved: false,
          isOutdated: false,
          comments: [
            {
              id: 'pr-comment-1',
              commentDatabaseId: 101,
              databaseId: 101,
              body: 'Please update the PR gate docs.',
              url: 'https://github.com/acme/demo/pull/42#discussion_r101',
              createdAt: '2026-06-17T00:00:00Z',
              author: { login: 'reviewer-1' },
            },
          ],
        },
      ],
    },
    issues: [],
    discussions: [],
  });

  assert.equal(snapshot.items[0].commentDatabaseId, 101);
});

test('evaluateFeedbackGate blocks auto-closure when non-closed feedback remains', () => {
  const snapshot = buildFeedbackSnapshot({
    repoFullName: 'acme/demo',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
      body: 'Related issue https://github.com/acme/demo/issues/12',
      reviewThreads: [
        {
          id: 'thread-1',
          isResolved: false,
          isOutdated: false,
          comments: [
            {
              id: 'pr-comment-1',
              body: 'Please update the PR gate docs.',
              url: 'https://github.com/acme/demo/pull/42#discussion_r1',
              createdAt: '2026-06-17T00:00:00Z',
              author: { login: 'reviewer-1' },
            },
          ],
        },
      ],
    },
    issues: [
      {
        number: 12,
        url: 'https://github.com/acme/demo/issues/12',
        title: 'Track review cleanup',
        state: 'OPEN',
        comments: [
          {
            id: 'issue-comment-1',
            body: 'Why is this kept in ops instead of base?',
            url: 'https://github.com/acme/demo/issues/12#issuecomment-1',
            createdAt: '2026-06-17T00:02:00Z',
            author: { login: 'ops-user' },
          },
        ],
      },
    ],
    discussions: [],
  });

  const gate = evaluateFeedbackGate(snapshot);

  assert.equal(gate.ready, false);
  assert.equal(gate.blockingItems.length, 2);
  assert.equal(gate.blockingCounts['actionable-change'], 1);
  assert.equal(gate.blockingCounts['explanation-only'], 1);
});

test('evaluateFeedbackGate allows auto-closure when every item is closed', () => {
  const snapshot = buildFeedbackSnapshot({
    repoFullName: 'acme/demo',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
      body: '',
      reviewThreads: [
        {
          id: 'thread-1',
          isResolved: true,
          isOutdated: false,
          comments: [
            {
              id: 'pr-comment-1',
              body: 'Looks good now.',
              url: 'https://github.com/acme/demo/pull/42#discussion_r1',
              createdAt: '2026-06-17T00:00:00Z',
              author: { login: 'reviewer-1' },
            },
          ],
        },
      ],
    },
    issues: [
      {
        number: 12,
        url: 'https://github.com/acme/demo/issues/12',
        title: 'Track review cleanup',
        state: 'CLOSED',
        comments: [
          {
            id: 'issue-comment-1',
            body: 'Fixed in the latest patch.',
            url: 'https://github.com/acme/demo/issues/12#issuecomment-1',
            createdAt: '2026-06-17T00:02:00Z',
            author: { login: 'ops-user' },
          },
        ],
      },
    ],
    discussions: [],
  });

  const gate = evaluateFeedbackGate(snapshot);

  assert.equal(gate.ready, true);
  assert.equal(gate.blockingItems.length, 0);
  assert.deepEqual(gate.blockingCounts, {});
});
