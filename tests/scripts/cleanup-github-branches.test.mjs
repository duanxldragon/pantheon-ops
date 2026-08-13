import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildBranchDeletePath,
  buildBranchLookupPath,
  buildDeletionDecision,
  collectBranchCleanupCandidates,
  deleteBranch,
} from '../../scripts/cleanup-github-branches.mjs';

test('collectBranchCleanupCandidates keeps only latest closed same-repo default-branch pull request heads', () => {
  const pullRequests = [
    {
      number: 10,
      state: 'closed',
      closed_at: '2026-06-18T03:01:10Z',
      head: { ref: 'chore/cleanup-one', sha: 'sha-new', repo: { full_name: 'duanxldragon/pantheon-base' } },
      base: { ref: 'main' },
    },
    {
      number: 9,
      state: 'closed',
      closed_at: '2026-06-18T02:00:00Z',
      head: { ref: 'chore/cleanup-one', sha: 'sha-old', repo: { full_name: 'duanxldragon/pantheon-base' } },
      base: { ref: 'main' },
    },
    {
      number: 8,
      state: 'closed',
      closed_at: '2026-06-18T01:00:00Z',
      head: { ref: 'main', sha: 'sha-main', repo: { full_name: 'duanxldragon/pantheon-base' } },
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
      head: { ref: 'release/0.8', sha: 'sha-release', repo: { full_name: 'duanxldragon/pantheon-base' } },
      base: { ref: 'release/0.8' },
    },
    {
      number: 5,
      state: 'open',
      closed_at: null,
      head: { ref: 'chore/still-open', sha: 'sha-open', repo: { full_name: 'duanxldragon/pantheon-base' } },
      base: { ref: 'main' },
    },
    {
      number: 4,
      state: 'closed',
      closed_at: '2026-06-17T22:00:00Z',
      head: { ref: 'fix/cleanup-two', sha: 'sha-two', repo: { full_name: 'duanxldragon/pantheon-base' } },
      base: { ref: 'main' },
    },
  ];

  assert.deepEqual(collectBranchCleanupCandidates(pullRequests, {
    repo: 'duanxldragon/pantheon-base',
    defaultBranch: 'main',
  }), [
    {
      branchName: 'chore/cleanup-one',
      closedAt: '2026-06-18T03:01:10Z',
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
    closedAt: '2026-06-18T03:01:10Z',
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

test('branch path helpers preserve slash-separated branch names for GitHub branch endpoints', () => {
  assert.equal(
    buildBranchLookupPath({
      owner: 'duanxldragon',
      repo: 'pantheon-base',
      branchName: 'verify/branch-hygiene-residue-base-20260618',
    }),
    '/repos/duanxldragon/pantheon-base/branches/verify/branch-hygiene-residue-base-20260618',
  );

  assert.equal(
    buildBranchDeletePath({
      owner: 'duanxldragon',
      repo: 'pantheon-base',
      branchName: 'verify/branch-hygiene-residue-base-20260618',
    }),
    '/repos/duanxldragon/pantheon-base/git/refs/heads/verify/branch-hygiene-residue-base-20260618',
  );
});

test('deleteBranch returns skip result for protected (422) and missing (404) branches', async () => {
  const originalFetch = globalThis.fetch;

  // 422 - protected branch
  globalThis.fetch = async () => ({ status: 422, ok: false, text: async () => '{"message":"Repository rule violations"}' });
  const protectedResult = await deleteBranch('owner', 'repo', 'release/v1', { token: 'test', apiBase: 'https://api.test' });
  assert.deepEqual(protectedResult, { deleted: false, reason: 'protected-or-rejected' });

  // 403 - insufficient permissions
  globalThis.fetch = async () => ({ status: 403, ok: false, text: async () => '{"message":"forbidden"}' });
  const forbiddenResult = await deleteBranch('owner', 'repo', 'protected-branch', { token: 'test', apiBase: 'https://api.test' });
  assert.deepEqual(forbiddenResult, { deleted: false, reason: 'protected-or-rejected' });

  // 404 - already gone
  globalThis.fetch = async () => ({ status: 404, ok: false, text: async () => '{"message":"Not Found"}' });
  const goneResult = await deleteBranch('owner', 'repo', 'deleted-branch', { token: 'test', apiBase: 'https://api.test' });
  assert.deepEqual(goneResult, { deleted: false, reason: 'already-gone' });

  // 204 - success
  globalThis.fetch = async () => ({ status: 204, ok: true, text: async () => '' });
  const successResult = await deleteBranch('owner', 'repo', 'stale-branch', { token: 'test', apiBase: 'https://api.test' });
  assert.deepEqual(successResult, { deleted: true, reason: 'deleted' });

  // 500 - unexpected error should throw
  globalThis.fetch = async () => ({ status: 500, ok: false, text: async () => 'Internal Server Error' });
  await assert.rejects(
    () => deleteBranch('owner', 'repo', 'broken', { token: 'test', apiBase: 'https://api.test' }),
    { message: /failed: 500/ },
  );

  globalThis.fetch = originalFetch;
});
