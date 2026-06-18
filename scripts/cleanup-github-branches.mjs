import process from 'node:process';

const DEFAULT_API_BASE = 'https://api.github.com';

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function createHeaders(token) {
  return {
    Accept: 'application/vnd.github+json',
    Authorization: `Bearer ${token}`,
    'User-Agent': 'pantheon-branch-hygiene',
    'X-GitHub-Api-Version': '2022-11-28',
  };
}

async function githubRequest(pathname, { method = 'GET', token, apiBase = DEFAULT_API_BASE } = {}) {
  const response = await fetch(`${apiBase}${pathname}`, {
    method,
    headers: createHeaders(token),
  });
  if (response.status === 404) {
    return { status: 404, data: null };
  }
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`GitHub API ${method} ${pathname} failed: ${response.status} ${body}`);
  }
  if (response.status === 204) {
    return { status: 204, data: null };
  }
  return { status: response.status, data: await response.json() };
}

async function paginate(pathname, { token, apiBase = DEFAULT_API_BASE } = {}) {
  const items = [];
  let page = 1;
  while (true) {
    const separator = pathname.includes('?') ? '&' : '?';
    const { data } = await githubRequest(`${pathname}${separator}per_page=100&page=${page}`, {
      token,
      apiBase,
    });
    if (!Array.isArray(data) || data.length === 0) {
      break;
    }
    items.push(...data);
    if (data.length < 100) {
      break;
    }
    page += 1;
  }
  return items;
}

export function collectBranchCleanupCandidates(pullRequests, { repo, defaultBranch }) {
  const latestByBranch = new Map();
  for (const pullRequest of pullRequests) {
    if (pullRequest.state !== 'closed') {
      continue;
    }
    if (pullRequest.base?.ref !== defaultBranch) {
      continue;
    }
    if (pullRequest.head?.repo?.full_name !== repo) {
      continue;
    }
    const branchName = pullRequest.head?.ref;
    const headSha = pullRequest.head?.sha;
    if (!branchName || !headSha || branchName === defaultBranch) {
      continue;
    }
    const current = latestByBranch.get(branchName);
    if (!current || new Date(pullRequest.closed_at ?? 0) > new Date(current.closedAt ?? 0)) {
      latestByBranch.set(branchName, {
        branchName,
        closedAt: pullRequest.closed_at,
        headSha,
        number: pullRequest.number,
      });
    }
  }
  return [...latestByBranch.values()].sort((left, right) => {
    return new Date(right.closedAt) - new Date(left.closedAt);
  });
}

export function buildDeletionDecision({ candidate, currentBranchSha, hasOpenPullRequest }) {
  if (!currentBranchSha) {
    return { action: 'skip-missing', reason: 'branch-missing' };
  }
  if (hasOpenPullRequest) {
    return { action: 'skip-open-pr', reason: 'open-pr-still-uses-branch' };
  }
  if (currentBranchSha !== candidate.headSha) {
    return { action: 'skip-recreated', reason: 'branch-recreated-or-advanced' };
  }
  return { action: 'delete', reason: 'closed-pr-head-branch-residue' };
}

export function buildBranchLookupPath({ owner, repo, branchName }) {
  return `/repos/${owner}/${repo}/branches/${branchName}`;
}

export function buildBranchDeletePath({ owner, repo, branchName }) {
  return `/repos/${owner}/${repo}/git/refs/heads/${branchName}`;
}

async function branchHeadSha(owner, repo, branchName, { token, apiBase }) {
  const { data, status } = await githubRequest(buildBranchLookupPath({ owner, repo, branchName }), {
    token,
    apiBase,
  });
  if (status === 404) {
    return null;
  }
  return data?.commit?.sha ?? null;
}

async function hasOpenPullRequestForBranch(owner, repo, branchName, { token, apiBase }) {
  const pulls = await paginate(`/repos/${owner}/${repo}/pulls?state=open&head=${owner}:${encodeURIComponent(branchName)}`, {
    token,
    apiBase,
  });
  return pulls.length > 0;
}

async function deleteBranch(owner, repo, branchName, { token, apiBase }) {
  await githubRequest(buildBranchDeletePath({ owner, repo, branchName }), {
    method: 'DELETE',
    token,
    apiBase,
  });
}

async function run() {
  const token = requireEnv('GH_TOKEN');
  const repoFullName = requireEnv('GH_REPO');
  const defaultBranch = process.env.DEFAULT_BRANCH || 'main';
  const apiBase = process.env.GITHUB_API_URL || DEFAULT_API_BASE;
  const [owner, repo] = repoFullName.split('/');
  if (!owner || !repo) {
    throw new Error(`Invalid GH_REPO value: ${repoFullName}`);
  }

  const closedPullRequests = await paginate(`/repos/${owner}/${repo}/pulls?state=closed&base=${encodeURIComponent(defaultBranch)}&sort=updated&direction=desc`, {
    token,
    apiBase,
  });
  const candidates = collectBranchCleanupCandidates(closedPullRequests, {
    repo: repoFullName,
    defaultBranch,
  });

  const summary = {
    repo: repoFullName,
    defaultBranch,
    scannedClosedPullRequests: closedPullRequests.length,
    candidates: candidates.length,
    deleted: [],
    skipped: [],
  };

  for (const candidate of candidates) {
    const currentBranchSha = await branchHeadSha(owner, repo, candidate.branchName, { token, apiBase });
    const openPullRequestUsesBranch = await hasOpenPullRequestForBranch(owner, repo, candidate.branchName, {
      token,
      apiBase,
    });
    const decision = buildDeletionDecision({
      candidate,
      currentBranchSha,
      hasOpenPullRequest: openPullRequestUsesBranch,
    });

    if (decision.action === 'delete') {
      await deleteBranch(owner, repo, candidate.branchName, { token, apiBase });
      summary.deleted.push({
        branchName: candidate.branchName,
        number: candidate.number,
        reason: decision.reason,
      });
      continue;
    }

    summary.skipped.push({
      branchName: candidate.branchName,
      number: candidate.number,
      reason: decision.reason,
    });
  }

  process.stdout.write(`${JSON.stringify(summary, null, 2)}\n`);
}

if (import.meta.url === `file://${process.argv[1].replace(/\\/g, '/')}`) {
  run().catch((error) => {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  });
}
