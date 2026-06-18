import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/address-github-feedback.mjs'),
).href;
const scriptPath = path.resolve(testDir, '../../scripts/address-github-feedback.mjs');

const {
  buildGhOperationSpecs,
  planWritebackActions,
} = await import(moduleUrl);

function createSnapshotFixture() {
  return {
    repoFullName: 'acme/demo',
    generatedAt: '2026-06-17T00:05:00Z',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
    },
    issues: [
      {
        number: 12,
        url: 'https://github.com/acme/demo/issues/12',
        title: 'Track review cleanup',
        state: 'OPEN',
      },
    ],
    discussions: [
      {
        id: 'discussion-node-7',
        number: 7,
        url: 'https://github.com/acme/demo/discussions/7',
        title: 'Should this live in base?',
        isAnswered: false,
      },
    ],
    linked: {
      issues: [{ number: 12, url: 'https://github.com/acme/demo/issues/12' }],
      discussions: [{ number: 7, url: 'https://github.com/acme/demo/discussions/7' }],
    },
    summary: {
      totalItems: 4,
      sourceCounts: {
        'pull-request-review': 1,
        'issue-comment': 1,
        'discussion-comment': 2,
      },
      classificationCounts: {
        'actionable-change': 1,
        'explanation-only': 1,
        'out-of-scope': 1,
        closed: 1,
      },
      recommendedActionCounts: {
        change: 1,
        reply: 2,
        skip: 1,
      },
    },
    items: [
      {
        id: 'pr-comment-node-1',
        commentDatabaseId: 101,
        sourceType: 'pull-request-review',
        sourceUrl: 'https://github.com/acme/demo/pull/42#discussion_r101',
        body: 'Please update the PR gate docs.',
        status: 'open',
        createdAt: '2026-06-17T00:00:00Z',
        authorLogin: 'reviewer-1',
        parentUrl: 'https://github.com/acme/demo/pull/42',
        pullRequestNumber: 42,
        issueNumber: null,
        discussionNumber: null,
        discussionId: null,
        threadId: 'thread-node-1',
        parentCommentId: null,
        isAnswer: false,
        classification: 'actionable-change',
        recommendedAction: 'change',
        confidence: 'high',
        reason: 'matched actionable change phrasing',
      },
      {
        id: 'issue-comment-node-1',
        sourceType: 'issue-comment',
        sourceUrl: 'https://github.com/acme/demo/issues/12#issuecomment-1',
        body: 'Why is this kept in ops instead of base?',
        status: 'open',
        createdAt: '2026-06-17T00:02:00Z',
        authorLogin: 'ops-user',
        parentUrl: 'https://github.com/acme/demo/issues/12',
        pullRequestNumber: null,
        issueNumber: 12,
        discussionNumber: null,
        discussionId: null,
        threadId: null,
        parentCommentId: null,
        isAnswer: false,
        classification: 'explanation-only',
        recommendedAction: 'reply',
        confidence: 'high',
        reason: 'question without requested code change',
      },
      {
        id: 'discussion-comment-node-1',
        sourceType: 'discussion-comment',
        sourceUrl: 'https://github.com/acme/demo/discussions/7#discussioncomment-1',
        body: 'This belongs in a follow-up PR.',
        status: 'open',
        createdAt: '2026-06-17T00:03:00Z',
        authorLogin: 'maintainer',
        parentUrl: 'https://github.com/acme/demo/discussions/7',
        pullRequestNumber: null,
        issueNumber: null,
        discussionNumber: 7,
        discussionId: 'discussion-node-7',
        threadId: null,
        parentCommentId: null,
        isAnswer: false,
        classification: 'out-of-scope',
        recommendedAction: 'reply',
        confidence: 'high',
        reason: 'matched out-of-scope phrasing',
      },
      {
        id: 'discussion-reply-node-1',
        sourceType: 'discussion-comment',
        sourceUrl: 'https://github.com/acme/demo/discussions/7#discussioncomment-2',
        body: 'Answered by the base fix.',
        status: 'answered',
        createdAt: '2026-06-17T00:04:00Z',
        authorLogin: 'maintainer',
        parentUrl: 'https://github.com/acme/demo/discussions/7',
        pullRequestNumber: null,
        issueNumber: null,
        discussionNumber: 7,
        discussionId: 'discussion-node-7',
        threadId: null,
        parentCommentId: 'discussion-comment-node-1',
        isAnswer: true,
        classification: 'closed',
        recommendedAction: 'skip',
        confidence: 'high',
        reason: 'status:answered',
      },
    ],
    gate: {
      ready: false,
      blockingItems: [],
      blockingCounts: {},
    },
  };
}

test('planWritebackActions builds reply, resolve, answer, and close actions from an explicit plan', () => {
  const actions = planWritebackActions(createSnapshotFixture(), {
    replies: {
      'pr-comment-node-1': 'Updated the PR gate docs and reran the governance checks.',
      'issue-comment-node-1': 'This stays in base because the workflow gate is shared platform governance.',
      'discussion-comment-node-1': 'Tracked as a follow-up PR because this branch only carries the governance gate.',
    },
    markRepliedDiscussionItemIdsAsAnswer: ['discussion-comment-node-1'],
    resolveReviewThreadIds: ['thread-node-1'],
    markDiscussionAnswerIds: ['discussion-reply-node-1'],
    closeIssueNumbers: [12],
  });

  assert.deepEqual(
    actions.map((action) => action.type),
    [
      'reply-review-comment',
      'reply-issue-comment',
      'reply-discussion-comment',
      'resolve-review-thread',
      'mark-discussion-comment-as-answer',
      'close-issue',
    ],
  );
  assert.equal(actions[0].commentDatabaseId, 101);
  assert.equal(actions[1].issueNumber, 12);
  assert.equal(actions[2].replyToId, 'discussion-comment-node-1');
  assert.equal(actions[2].markReplyAsAnswer, true);
  assert.equal(actions[3].threadId, 'thread-node-1');
  assert.equal(actions[4].commentId, 'discussion-reply-node-1');
  assert.equal(actions[5].issueNumber, 12);
});

test('buildGhOperationSpecs renders the expected gh api calls for each writeback action', () => {
  const actions = planWritebackActions(createSnapshotFixture(), {
    replies: {
      'pr-comment-node-1': 'Updated the PR gate docs and reran the governance checks.',
      'issue-comment-node-1': 'This stays in base because the workflow gate is shared platform governance.',
      'discussion-comment-node-1': 'Tracked as a follow-up PR because this branch only carries the governance gate.',
    },
    markRepliedDiscussionItemIdsAsAnswer: ['discussion-comment-node-1'],
    resolveReviewThreadIds: ['thread-node-1'],
    markDiscussionAnswerIds: ['discussion-reply-node-1'],
    closeIssueNumbers: [12],
  });
  const operations = buildGhOperationSpecs('acme/demo', actions);

  assert.deepEqual(
    operations.map((operation) => operation.type),
    actions.map((action) => action.type),
  );
  assert.deepEqual(operations[0].args.slice(0, 4), ['api', '-X', 'POST', 'repos/acme/demo/pulls/42/comments/101/replies']);
  assert.match(operations[1].args.join(' '), /repos\/acme\/demo\/issues\/12\/comments/);
  assert.match(operations[2].args.join(' '), /addDiscussionComment/);
  assert.equal(operations[2].markReplyAsAnswer, true);
  assert.match(operations[2].description, /mark reply as answer/i);
  assert.match(operations[2].followUpArgs.join(' '), /markDiscussionCommentAsAnswer/);
  assert.match(operations[3].args.join(' '), /resolveReviewThread/);
  assert.match(operations[4].args.join(' '), /markDiscussionCommentAsAnswer/);
  assert.deepEqual(operations[5].args.slice(0, 4), ['api', '-X', 'PATCH', 'repos/acme/demo/issues/12']);
});

test('address-github-feedback dry run emits planned actions from snapshot and plan files', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-address-feedback-'));
  try {
    const snapshotPath = path.join(tempDir, 'snapshot.json');
    const planPath = path.join(tempDir, 'plan.json');
    fs.writeFileSync(snapshotPath, JSON.stringify(createSnapshotFixture(), null, 2));
    fs.writeFileSync(
      planPath,
      JSON.stringify(
        {
          replies: {
            'pr-comment-node-1': 'Updated the PR gate docs and reran the governance checks.',
          },
          resolveReviewThreadIds: ['thread-node-1'],
        },
        null,
        2,
      ),
    );

    const stdout = execFileSync(
      'node',
      [scriptPath, '--snapshot', snapshotPath, '--plan', planPath, '--dry-run', '--json'],
      { encoding: 'utf8' },
    );
    const payload = JSON.parse(stdout);

    assert.equal(payload.repoFullName, 'acme/demo');
    assert.equal(payload.actions.length, 2);
    assert.deepEqual(
      payload.actions.map((action) => action.type),
      ['reply-review-comment', 'resolve-review-thread'],
    );
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});
