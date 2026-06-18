import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildDeletionDecision,
  collectBranchCleanupCandidates,
} from '../../scripts/cleanup-github-branches.mjs';

test('collectBranchCleanupCandidates keeps only latest closed same-repo default-branch pull request heads', () => {
  const pullRequests = [
    {
      number: 10,
      state: 'closed',
      closed_at: '2026-06-18T02:56:12Z',
      head: { ref: 'chore/cleanup-one', sha: 'sha-new', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'main' },
    },
    {
      number: 9,
      state: 'closed',
      closed_at: '2026-06-18T02:00:00Z',
      head: { ref: 'chore/cleanup-one', sha: 'sha-old', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'main' },
    },
    {
      number: 8,
      state: 'closed',
      closed_at: '2026-06-18T01:00:00Z',
      head: { ref: 'main', sha: 'sha-main', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'main' },
    },
    {
      number: 7,
      state: 'closed',
      closed_at: '2026-06-18T00:00:00Z',
      head: { ref: 'chore/fork-branch', sha: 'sha-fork', repo: { full_name: 'someone/fork' } },
      base: { ref: 'main' },
    },
    {
      number: 6,
      state: 'closed',
      closed_at: '2026-06-17T23:00:00Z',
      head: { ref: 'release/0.8', sha: 'sha-release', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'release/0.8' },
    },
    {
      number: 5,
      state: 'open',
      closed_at: null,
      head: { ref: 'chore/still-open', sha: 'sha-open', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'main' },
    },
    {
      number: 4,
      state: 'closed',
      closed_at: '2026-06-17T22:00:00Z',
      head: { ref: 'fix/cleanup-two', sha: 'sha-two', repo: { full_name: 'duanxldragon/pantheon-ops' } },
      base: { ref: 'main' },
    },
  ];

  assert.deepEqual(collectBranchCleanupCandidates(pullRequests, {
    repo: 'duanxldragon/pantheon-ops',
    defaultBranch: 'main',
  }), [
    {
      branchName: 'chore/cleanup-one',
      closedAt: '2026-06-18T02:56:12Z',
      headSha: 'sha-new',
      number: 10,
    },
    {
      branchName: 'fix/cleanup-two',
      closedAt: '2026-06-17T22:00:00Z',
      headSha: 'sha-two',
      number: 4,
    },
  ]);
});

test('buildDeletionDecision deletes only exact stale closed-pr branch residues', () => {
  const candidate = {
    branchName: 'chore/cleanup-one',
    closedAt: '2026-06-18T02:56:12Z',
    headSha: 'sha-new',
    number: 10,
  };

  assert.deepEqual(buildDeletionDecision({
    candidate,
    currentBranchSha: 'sha-new',
    hasOpenPullRequest: false,
  }), {
    action: 'delete',
    reason: 'closed-pr-head-branch-residue',
  });

  assert.deepEqual(buildDeletionDecision({
    candidate,
    currentBranchSha: null,
    hasOpenPullRequest: false,
  }), {
    action: 'skip-missing',
    reason: 'branch-missing',
  });

  assert.deepEqual(buildDeletionDecision({
    candidate,
    currentBranchSha: 'sha-advanced',
    hasOpenPullRequest: false,
  }), {
    action: 'skip-recreated',
    reason: 'branch-recreated-or-advanced',
  });

  assert.deepEqual(buildDeletionDecision({
    candidate,
    currentBranchSha: 'sha-new',
    hasOpenPullRequest: true,
  }), {
    action: 'skip-open-pr',
    reason: 'open-pr-still-uses-branch',
  });
});
