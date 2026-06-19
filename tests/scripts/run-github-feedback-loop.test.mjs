import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(testDir, '../..');
const canonicalRepoName = fs.existsSync(path.join(repoRoot, 'docs', 'PROJECT_INHERITANCE.md'))
  ? 'pantheon-ops'
  : 'pantheon-base';
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/run-github-feedback-loop.mjs'),
).href;
const scriptPath = path.resolve(testDir, '../../scripts/run-github-feedback-loop.mjs');

const {
  deriveFeedbackContext,
  deriveAutomatedWritebackPlan,
} = await import(moduleUrl);

function createSnapshotFixture() {
  return {
    repoFullName: 'acme/demo',
    pullRequest: {
      number: 42,
      url: 'https://github.com/acme/demo/pull/42',
      title: 'Tighten GitHub governance',
      body: '',
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
    items: [
      {
        id: 'pr-comment-node-1',
        commentDatabaseId: 101,
        sourceType: 'pull-request-review',
        sourceUrl: 'https://github.com/acme/demo/pull/42#discussion_r101',
        body: 'Please update the PR gate docs.',
        status: 'open',
        authorLogin: 'reviewer-1',
        pullRequestNumber: 42,
        issueNumber: null,
        discussionNumber: null,
        discussionId: null,
        threadId: 'thread-node-1',
        parentCommentId: null,
        isAnswer: false,
        classification: 'actionable-change',
        recommendedAction: 'change',
      },
      {
        id: 'issue-comment-node-1',
        sourceType: 'issue-comment',
        sourceUrl: 'https://github.com/acme/demo/issues/12#issuecomment-1',
        body: 'Why is this kept in base instead of ops?',
        status: 'open',
        authorLogin: 'ops-user',
        pullRequestNumber: null,
        issueNumber: 12,
        discussionNumber: null,
        discussionId: null,
        threadId: null,
        parentCommentId: null,
        isAnswer: false,
        classification: 'explanation-only',
        recommendedAction: 'reply',
      },
      {
        id: 'discussion-comment-node-1',
        sourceType: 'discussion-comment',
        sourceUrl: 'https://github.com/acme/demo/discussions/7#discussioncomment-1',
        body: 'This belongs in a follow-up PR.',
        status: 'open',
        authorLogin: 'maintainer',
        pullRequestNumber: null,
        issueNumber: null,
        discussionNumber: 7,
        discussionId: 'discussion-node-7',
        threadId: null,
        parentCommentId: null,
        isAnswer: false,
        classification: 'out-of-scope',
        recommendedAction: 'reply',
      },
      {
        id: 'discussion-reply-node-1',
        sourceType: 'discussion-comment',
        sourceUrl: 'https://github.com/acme/demo/discussions/7#discussioncomment-2',
        body: 'Answered by the base fix.',
        status: 'answered',
        authorLogin: 'maintainer',
        pullRequestNumber: null,
        issueNumber: null,
        discussionNumber: 7,
        discussionId: 'discussion-node-7',
        threadId: null,
        parentCommentId: 'discussion-comment-node-1',
        isAnswer: true,
        classification: 'closed',
        recommendedAction: 'skip',
      },
    ],
    gate: {
      ready: false,
      blockingItems: [],
      blockingCounts: {},
    },
  };
}

function createContextFixture() {
  return {
    changeSummary: [
      'Updated the PR governance docs and GitHub feedback automation.',
      'Re-ran the repository governance workflow tests.',
    ],
    verificationSummary: [
      'node --test tests/scripts/address-github-feedback.test.mjs tests/scripts/fetch-github-feedback.test.mjs',
      'node --test tests/scripts/pr-automation-workflow.test.mjs',
    ],
    ownershipSummary:
      'This stays in pantheon-base because the PR automation gate is shared platform governance rather than a business overlay.',
    outOfScopeSummary:
      'Follow-up behavior changes are intentionally deferred to keep this PR scoped to GitHub governance automation.',
    closeIssueNumbers: [12],
    resolveOutOfScopeReviewThreads: false,
  };
}

function createAutoContextTaskId() {
  return '2026-06-17-github-feedback-automation';
}

function createAutoContextOwnershipNote() {
  if (canonicalRepoName === 'pantheon-ops') {
    return 'This patch stays in `pantheon-ops` because it is repository governance and local PR closure logic, not shared platform runtime behavior.';
  }
  return 'This stays in `pantheon-base` because the PR automation gate is shared platform governance rather than a business overlay.';
}

function createAutoContextManifest(taskId) {
  return {
    taskId,
    title: 'GitHub feedback automation governance',
    goal: 'Tighten the GitHub governance automation so feedback can be closed without manual triage.',
    primaryLayer: 'business/deploy',
    scope: {
      in: [
        'Update the GitHub feedback automation scripts and writeback flow.',
        'Keep repository governance checks aligned with the automation loop.',
      ],
      out: [
        'Follow-up product runtime changes stay outside this PR.',
        'Existing repo baseline debt remains tracked separately.',
      ],
    },
    implementationNotes: [
      createAutoContextOwnershipNote(),
      'The automation reuses repo-local evidence instead of hand-written closeout notes.',
    ],
    linkage: {
      evidenceDir: `.harness/evidence/${taskId}/`,
      reviewFile: `.harness/evidence/${taskId}/review.md`,
      changeRef: 'none',
      planRefs: [],
    },
    verificationPlan: {
      commands: [
        'node --test tests/scripts/address-github-feedback.test.mjs tests/scripts/fetch-github-feedback.test.mjs',
        'node --test tests/scripts/pr-automation-workflow.test.mjs',
      ],
      runtimeEvidence: [],
    },
    runtimeSensitive: false,
    evidenceRequired: ['commands.json'],
    humanGates: ['none'],
    completionChecklist: [
      'Layer and boundary declared',
      'Contract anchors read',
      'Verification run or exception recorded',
      'Evidence saved or summarized',
      'Review completed',
    ],
  };
}

function createAutoContextCommands(taskId) {
  return {
    taskId,
    repo: canonicalRepoName,
    commands: [
      {
        command:
          'node --test tests/scripts/address-github-feedback.test.mjs tests/scripts/fetch-github-feedback.test.mjs',
        cwd: canonicalRepoName,
        status: 'passed',
        durationMs: 0,
        notes: 'GitHub feedback script tests passed.',
      },
      {
        command: 'node --test tests/scripts/pr-automation-workflow.test.mjs',
        cwd: canonicalRepoName,
        status: 'passed',
        durationMs: 0,
        notes: 'PR automation workflow regression tests passed.',
      },
      {
        command: 'npm run check:inheritance',
        cwd: canonicalRepoName,
        status: 'failed',
        durationMs: 0,
        notes: 'Unrelated baseline drift remains deferred.',
      },
    ],
    linkage: {
      taskManifest: `.harness/tasks/${taskId}/manifest.json`,
      evidenceDir: `.harness/evidence/${taskId}/`,
      reviewFile: `.harness/evidence/${taskId}/review.md`,
      changeRef: 'none',
      planRefs: [],
    },
    completedAt: '2026-06-17T06:40:00Z',
  };
}

function createAutoContextPullRequestBody(taskId) {
  return `
## Summary

- Automation-only governance follow-up.

- Task ID: ${taskId}
- Task Manifest: .harness/tasks/${taskId}/manifest.json
- Evidence: .harness/evidence/${taskId}/commands.json
- Review Artifact: .harness/evidence/${taskId}/review.md

Closes #12
`;
}

function createAutoContextSnapshotFixture(taskId) {
  const snapshot = createSnapshotFixture();
  snapshot.repoFullName = 'acme/pantheon-ops';
  snapshot.pullRequest.body = createAutoContextPullRequestBody(taskId);
  return snapshot;
}

function writeAutoContextArtifacts(rootDir) {
  const taskId = createAutoContextTaskId();
  const evidenceDir = path.join(rootDir, '.harness', 'evidence', taskId);
  const manifestPath = path.join(rootDir, '.harness', 'tasks', taskId, 'manifest.json');
  fs.mkdirSync(evidenceDir, { recursive: true });
  fs.mkdirSync(path.dirname(manifestPath), { recursive: true });
  fs.writeFileSync(
    manifestPath,
    `${JSON.stringify(createAutoContextManifest(taskId), null, 2)}\n`,
  );
  fs.writeFileSync(
    path.join(evidenceDir, 'commands.json'),
    JSON.stringify(createAutoContextCommands(taskId), null, 2),
  );
  fs.writeFileSync(
    path.join(evidenceDir, 'review.md'),
    '# Review\n\nNo findings.\n',
  );
  return {
    taskId,
    evidenceDir,
  };
}

test('deriveAutomatedWritebackPlan creates replies and closures from snapshot plus context', () => {
  const result = deriveAutomatedWritebackPlan(createSnapshotFixture(), createContextFixture());

  assert.deepEqual(result.blockedItems, []);
  assert.equal(Object.keys(result.plan.replies).length, 3);
  assert.match(result.plan.replies['pr-comment-node-1'], /Updated the PR governance docs/);
  assert.match(result.plan.replies['issue-comment-node-1'], /shared platform governance/);
  assert.match(result.plan.replies['discussion-comment-node-1'], /intentionally deferred/);
  assert.deepEqual(result.plan.resolveReviewThreadIds, ['thread-node-1']);
  assert.deepEqual(result.plan.closeIssueNumbers, [12]);
  assert.deepEqual(result.plan.markRepliedDiscussionItemIdsAsAnswer, ['discussion-comment-node-1']);
  assert.deepEqual(result.plan.markDiscussionAnswerIds, []);
});

test('deriveAutomatedWritebackPlan blocks explanation-only items when explanation context is missing', () => {
  const context = createContextFixture();
  delete context.ownershipSummary;

  const result = deriveAutomatedWritebackPlan(createSnapshotFixture(), context);

  assert.equal(result.blockedItems.length, 1);
  assert.equal(result.blockedItems[0].id, 'issue-comment-node-1');
  assert.match(result.blockedItems[0].reason, /ownershipSummary/);
});

test('deriveFeedbackContext pulls change, verification, ownership, and close intent from local evidence', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-feedback-context-'));
  try {
    const { taskId } = writeAutoContextArtifacts(tempDir);
    const snapshot = createAutoContextSnapshotFixture(taskId);

    const context = deriveFeedbackContext(snapshot, { repoRoot: tempDir });

    assert.match(
      context.changeSummary[0],
      /Tighten the GitHub governance automation so feedback can be closed without manual triage/i,
    );
    assert.match(context.changeSummary[1], /Update the GitHub feedback automation scripts/i);
    assert.deepEqual(context.verificationSummary, [
      'node --test tests/scripts/address-github-feedback.test.mjs tests/scripts/fetch-github-feedback.test.mjs',
      'node --test tests/scripts/pr-automation-workflow.test.mjs',
    ]);
    assert.match(context.ownershipSummary, new RegExp(canonicalRepoName, 'i'));
    assert.match(context.outOfScopeSummary, /Follow-up product runtime changes stay outside this PR/i);
    assert.deepEqual(context.closeIssueNumbers, [12]);
    assert.equal(context.resolveOutOfScopeReviewThreads, false);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('run-github-feedback-loop dry run emits plan and actions from snapshot and context files', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-feedback-loop-'));
  try {
    const snapshotPath = path.join(tempDir, 'snapshot.json');
    const contextPath = path.join(tempDir, 'context.json');
    fs.writeFileSync(snapshotPath, JSON.stringify(createSnapshotFixture(), null, 2));
    fs.writeFileSync(contextPath, JSON.stringify(createContextFixture(), null, 2));

    const stdout = execFileSync(
      'node',
      [
        scriptPath,
        '--snapshot',
        snapshotPath,
        '--context',
        contextPath,
        '--dry-run',
        '--json',
      ],
      { encoding: 'utf8' },
    );
    const payload = JSON.parse(stdout);

    assert.equal(payload.blockedItems.length, 0);
    assert.equal(Object.keys(payload.plan.replies).length, 3);
    assert.deepEqual(payload.plan.markRepliedDiscussionItemIdsAsAnswer, ['discussion-comment-node-1']);
    assert.equal(payload.actions.length, 5);
    assert.deepEqual(
      payload.actions.map((action) => action.type),
      [
        'reply-review-comment',
        'reply-issue-comment',
        'reply-discussion-comment',
        'resolve-review-thread',
        'close-issue',
      ],
    );
    assert.equal(payload.actions[2].markReplyAsAnswer, true);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test('run-github-feedback-loop dry run derives context automatically when no context file is provided', () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-feedback-loop-auto-'));
  try {
    const { taskId } = writeAutoContextArtifacts(tempDir);
    const snapshotPath = path.join(tempDir, 'snapshot.json');
    const snapshot = createAutoContextSnapshotFixture(taskId);
    fs.writeFileSync(snapshotPath, JSON.stringify(snapshot, null, 2));

    const stdout = execFileSync(
      'node',
      [scriptPath, '--snapshot', snapshotPath, '--dry-run', '--json'],
      {
        cwd: tempDir,
        encoding: 'utf8',
      },
    );
    const payload = JSON.parse(stdout);

    assert.equal(payload.blockedItems.length, 0);
    assert.equal(payload.contextSource, 'derived');
    assert.deepEqual(payload.context.closeIssueNumbers, [12]);
    assert.equal(Object.keys(payload.plan.replies).length, 3);
    assert.deepEqual(payload.plan.markRepliedDiscussionItemIdsAsAnswer, ['discussion-comment-node-1']);
    assert.deepEqual(
      payload.actions.map((action) => action.type),
      [
        'reply-review-comment',
        'reply-issue-comment',
        'reply-discussion-comment',
        'resolve-review-thread',
        'close-issue',
      ],
    );
    assert.equal(payload.actions[2].markReplyAsAnswer, true);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});
