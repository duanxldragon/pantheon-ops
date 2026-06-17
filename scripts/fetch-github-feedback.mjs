import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const GITHUB_REFERENCE_PATTERN =
  /https:\/\/github\.com\/([^/\s]+)\/([^/\s]+)\/(issues|discussions)\/(\d+)/gi;
const CLOSED_STATUSES = new Set(['resolved', 'closed', 'outdated', 'answered']);
const OUT_OF_SCOPE_PATTERNS = [
  /\bfollow[- ]up pr\b/i,
  /\bseparate pr\b/i,
  /\bseparate issue\b/i,
  /\bout of scope\b/i,
  /\bbelongs in\b/i,
  /\bnot in this pr\b/i,
];
const ACTIONABLE_PATTERNS = [
  /\bplease\b/i,
  /\b(add|update|remove|rename|fix|document|cover|test|align|split|move|revert|handle|use)\b/i,
  /\b(regression test|unit test|docs|workflow|gate|evidence|validation)\b/i,
];
const QUESTION_PATTERNS = [
  /^(why|how|what|which|where|when)\b/i,
  /^(can|could|would|will|is|are|do|does|did)\b/i,
];
const ALREADY_SATISFIED_PATTERNS = [
  /\blooks good\b/i,
  /\blgtm\b/i,
  /\balready (fixed|covered|handled|addressed)\b/i,
  /\bfixed in\b/i,
  /\baddressed in\b/i,
];

function normalizeRepoFullName(repoFullName) {
  return String(repoFullName ?? '').trim().toLowerCase();
}

function normalizeBody(body) {
  return String(body ?? '').replace(/\r\n/g, '\n').trim();
}

function toCanonicalReferenceUrl(repoFullName, type, number) {
  return `https://github.com/${repoFullName}/${type}/${number}`;
}

function appendUniqueReference(target, bucket, number, url) {
  const exists = target[bucket].some((entry) => entry.number === number);
  if (!exists) {
    target[bucket].push({ number, url });
  }
}

function incrementCounter(counter, key) {
  counter[key] = (counter[key] ?? 0) + 1;
}

function bodyMatchesAny(body, patterns) {
  return patterns.some((pattern) => pattern.test(body));
}

function isQuestionOnly(body) {
  if (!body) {
    return false;
  }
  const normalized = body.trim();
  return (
    normalized.includes('?') ||
    QUESTION_PATTERNS.some((pattern) => pattern.test(normalized))
  );
}

function parseRepoParts(repoFullName) {
  const [owner, repo] = String(repoFullName ?? '').split('/');
  if (!owner || !repo) {
    throw new Error(`invalid repo full name: ${repoFullName}`);
  }
  return { owner, repo };
}

function parseJsonOutput(rawOutput, description) {
  try {
    return JSON.parse(rawOutput);
  } catch (error) {
    throw new Error(`${description} returned invalid JSON: ${error.message}`);
  }
}

function runGhCommand(args, description) {
  try {
    return execFileSync('gh', args, {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    });
  } catch (error) {
    const stderr = error.stderr ? String(error.stderr).trim() : '';
    const suffix = stderr ? `: ${stderr}` : '';
    throw new Error(`${description} failed${suffix}`);
  }
}

function runGhJson(args, description) {
  return parseJsonOutput(runGhCommand(args, description), description);
}

function runGhGraphql(query, variables, description) {
  const args = ['api', 'graphql', '-f', `query=${query}`];
  for (const [name, value] of Object.entries(variables)) {
    if (value === undefined || value === null) {
      continue;
    }
    if (typeof value === 'number') {
      args.push('-F', `${name}=${value}`);
      continue;
    }
    args.push('-f', `${name}=${value}`);
  }
  return runGhJson(args, description);
}

function resolveCurrentRepoFullName() {
  const payload = runGhJson(
    ['repo', 'view', '--json', 'nameWithOwner'],
    'gh repo view --json nameWithOwner',
  );
  if (!payload?.nameWithOwner) {
    throw new Error('unable to resolve current repository nameWithOwner');
  }
  return payload.nameWithOwner;
}

function resolveCurrentPullRequestNumber() {
  const payload = runGhJson(['pr', 'view', '--json', 'number'], 'gh pr view --json number');
  if (typeof payload?.number !== 'number') {
    throw new Error('unable to resolve current pull request number');
  }
  return payload.number;
}

function normalizeAuthor(author) {
  return { login: author?.login ?? null };
}

function normalizePullRequestComment(comment) {
  return {
    id: comment.id,
    commentDatabaseId: comment.databaseId ?? null,
    body: comment.body ?? '',
    url: comment.url ?? '',
    createdAt: comment.createdAt ?? null,
    author: normalizeAuthor(comment.author),
  };
}

function normalizeIssueComment(comment) {
  return {
    id: comment.id,
    body: comment.body ?? '',
    url: comment.url ?? '',
    createdAt: comment.createdAt ?? null,
    author: normalizeAuthor(comment.author),
  };
}

function normalizeDiscussionReply(reply) {
  return {
    id: reply.id,
    body: reply.body ?? '',
    url: reply.url ?? '',
    createdAt: reply.createdAt ?? null,
    author: normalizeAuthor(reply.author),
    isAnswer: Boolean(reply.isAnswer),
  };
}

function normalizeDiscussionComment(comment) {
  return {
    id: comment.id,
    body: comment.body ?? '',
    url: comment.url ?? '',
    createdAt: comment.createdAt ?? null,
    author: normalizeAuthor(comment.author),
    isAnswer: Boolean(comment.isAnswer),
    replies: (comment.replies?.nodes ?? []).map(normalizeDiscussionReply),
  };
}

const PULL_REQUEST_QUERY = `
query($owner:String!, $repo:String!, $pullRequestNumber:Int!, $reviewThreadsCursor:String) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$pullRequestNumber) {
      number
      url
      title
      body
      reviewThreads(first:100, after:$reviewThreadsCursor) {
        nodes {
          id
          isResolved
          isOutdated
          comments(first:100) {
            nodes {
              id
              databaseId
              body
              url
              createdAt
              author {
                login
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`;

function fetchPullRequest({ repoFullName, pullRequestNumber }) {
  const { owner, repo } = parseRepoParts(repoFullName);
  let reviewThreadsCursor;
  let pullRequest;
  const reviewThreads = [];

  do {
    const payload = runGhGraphql(
      PULL_REQUEST_QUERY,
      { owner, repo, pullRequestNumber, reviewThreadsCursor },
      `gh api graphql pull request ${repoFullName}#${pullRequestNumber}`,
    );
    pullRequest = payload?.data?.repository?.pullRequest;
    if (!pullRequest) {
      throw new Error(`pull request not found: ${repoFullName}#${pullRequestNumber}`);
    }
    for (const thread of pullRequest.reviewThreads?.nodes ?? []) {
      reviewThreads.push({
        id: thread.id,
        isResolved: Boolean(thread.isResolved),
        isOutdated: Boolean(thread.isOutdated),
        comments: (thread.comments?.nodes ?? []).map(normalizePullRequestComment),
      });
    }
    reviewThreadsCursor = pullRequest.reviewThreads?.pageInfo?.hasNextPage
      ? pullRequest.reviewThreads.pageInfo.endCursor
      : null;
  } while (reviewThreadsCursor);

  return {
    number: pullRequest.number,
    url: pullRequest.url,
    title: pullRequest.title,
    body: pullRequest.body ?? '',
    reviewThreads,
  };
}

const ISSUE_QUERY = `
query($owner:String!, $repo:String!, $issueNumber:Int!, $commentsCursor:String) {
  repository(owner:$owner, name:$repo) {
    issue(number:$issueNumber) {
      number
      url
      title
      state
      comments(first:100, after:$commentsCursor) {
        nodes {
          id
          body
          url
          createdAt
          author {
            login
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`;

function fetchIssue({ repoFullName, issueNumber }) {
  const { owner, repo } = parseRepoParts(repoFullName);
  let commentsCursor;
  let issue;
  const comments = [];

  do {
    const payload = runGhGraphql(
      ISSUE_QUERY,
      { owner, repo, issueNumber, commentsCursor },
      `gh api graphql issue ${repoFullName}#${issueNumber}`,
    );
    issue = payload?.data?.repository?.issue;
    if (!issue) {
      return null;
    }
    comments.push(...(issue.comments?.nodes ?? []).map(normalizeIssueComment));
    commentsCursor = issue.comments?.pageInfo?.hasNextPage
      ? issue.comments.pageInfo.endCursor
      : null;
  } while (commentsCursor);

  return {
    number: issue.number,
    url: issue.url,
    title: issue.title,
    state: issue.state,
    comments,
  };
}

const DISCUSSION_QUERY = `
query($owner:String!, $repo:String!, $discussionNumber:Int!, $commentsCursor:String) {
  repository(owner:$owner, name:$repo) {
    discussion(number:$discussionNumber) {
      number
      url
      title
      isAnswered
      comments(first:100, after:$commentsCursor) {
        nodes {
          id
          body
          url
          createdAt
          isAnswer
          author {
            login
          }
          replies(first:100) {
            nodes {
              id
              body
              url
              createdAt
              isAnswer
              author {
                login
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
`;

function fetchDiscussion({ repoFullName, discussionNumber }) {
  const { owner, repo } = parseRepoParts(repoFullName);
  let commentsCursor;
  let discussion;
  const comments = [];

  do {
    const payload = runGhGraphql(
      DISCUSSION_QUERY,
      { owner, repo, discussionNumber, commentsCursor },
      `gh api graphql discussion ${repoFullName}#${discussionNumber}`,
    );
    discussion = payload?.data?.repository?.discussion;
    if (!discussion) {
      return null;
    }
    comments.push(...(discussion.comments?.nodes ?? []).map(normalizeDiscussionComment));
    commentsCursor = discussion.comments?.pageInfo?.hasNextPage
      ? discussion.comments.pageInfo.endCursor
      : null;
  } while (commentsCursor);

  return {
    number: discussion.number,
    url: discussion.url,
    title: discussion.title,
    isAnswered: Boolean(discussion.isAnswered),
    comments,
  };
}

function fetchGitHubFeedbackSnapshot({ repoFullName, pullRequestNumber }) {
  const pullRequest = fetchPullRequest({ repoFullName, pullRequestNumber });
  const linked = extractGitHubReferences(pullRequest.body, repoFullName);

  const issues = linked.issues
    .map((issue) => fetchIssue({ repoFullName, issueNumber: issue.number }))
    .filter(Boolean);
  const discussions = linked.discussions
    .map((discussion) =>
      fetchDiscussion({ repoFullName, discussionNumber: discussion.number }),
    )
    .filter(Boolean);

  return {
    repoFullName,
    pullRequest,
    issues,
    discussions,
  };
}

function createBaseItem(sourceType, data) {
  return {
    id: data.id,
    sourceType,
    sourceUrl: data.url ?? '',
    body: data.body ?? '',
    status: data.status,
    createdAt: data.createdAt ?? null,
    authorLogin: data.author?.login ?? null,
    parentUrl: data.parentUrl ?? null,
    pullRequestNumber: data.pullRequestNumber ?? null,
    issueNumber: data.issueNumber ?? null,
    discussionNumber: data.discussionNumber ?? null,
    threadId: data.threadId ?? null,
    parentCommentId: data.parentCommentId ?? null,
    isAnswer: Boolean(data.isAnswer),
  };
}

function flattenPullRequestItems(pullRequest) {
  const items = [];
  for (const thread of pullRequest?.reviewThreads ?? []) {
    const status = thread.isResolved ? 'resolved' : thread.isOutdated ? 'outdated' : 'open';
    for (const comment of thread.comments ?? []) {
      items.push(
        createBaseItem('pull-request-review', {
          ...comment,
          status,
          parentUrl: pullRequest.url ?? null,
          pullRequestNumber: pullRequest.number ?? null,
          threadId: thread.id ?? null,
        }),
      );
    }
  }
  return items;
}

function flattenIssueItems(issues) {
  const items = [];
  for (const issue of issues ?? []) {
    const status = String(issue.state ?? '').toUpperCase() === 'CLOSED' ? 'closed' : 'open';
    for (const comment of issue.comments ?? []) {
      items.push(
        createBaseItem('issue-comment', {
          ...comment,
          status,
          parentUrl: issue.url ?? null,
          issueNumber: issue.number ?? null,
        }),
      );
    }
  }
  return items;
}

function flattenDiscussionItems(discussions) {
  const items = [];
  for (const discussion of discussions ?? []) {
    const discussionAnswered = Boolean(discussion.isAnswered);
    for (const comment of discussion.comments ?? []) {
      items.push(
        createBaseItem('discussion-comment', {
          ...comment,
          status: comment.isAnswer || discussionAnswered ? 'answered' : 'open',
          parentUrl: discussion.url ?? null,
          discussionNumber: discussion.number ?? null,
          isAnswer: comment.isAnswer,
        }),
      );
      for (const reply of comment.replies ?? []) {
        items.push(
          createBaseItem('discussion-comment', {
            ...reply,
            status: reply.isAnswer || discussionAnswered ? 'answered' : 'open',
            parentUrl: discussion.url ?? null,
            discussionNumber: discussion.number ?? null,
            parentCommentId: comment.id ?? null,
            isAnswer: reply.isAnswer,
          }),
        );
      }
    }
  }
  return items;
}

function summarizeItems(items) {
  const sourceCounts = {};
  const classificationCounts = {};
  const recommendedActionCounts = {};

  for (const item of items) {
    incrementCounter(sourceCounts, item.sourceType);
    incrementCounter(classificationCounts, item.classification);
    incrementCounter(recommendedActionCounts, item.recommendedAction);
  }

  return {
    totalItems: items.length,
    sourceCounts,
    classificationCounts,
    recommendedActionCounts,
  };
}

function summarizeBlockingItems(items) {
  const blockingCounts = {};
  for (const item of items) {
    incrementCounter(blockingCounts, item.classification);
  }
  return blockingCounts;
}

function pickConfidence(level) {
  return level;
}

export function extractGitHubReferences(text, repoFullName) {
  const targetRepo = normalizeRepoFullName(repoFullName);
  const references = { issues: [], discussions: [] };
  const content = String(text ?? '');

  for (const match of content.matchAll(GITHUB_REFERENCE_PATTERN)) {
    const owner = match[1];
    const repo = match[2];
    const type = match[3];
    const number = Number.parseInt(match[4], 10);
    const matchedRepo = normalizeRepoFullName(`${owner}/${repo}`);
    if (!Number.isInteger(number) || matchedRepo !== targetRepo) {
      continue;
    }
    const bucket = type === 'issues' ? 'issues' : 'discussions';
    appendUniqueReference(
      references,
      bucket,
      number,
      toCanonicalReferenceUrl(targetRepo, type, number),
    );
  }

  return references;
}

export function classifyFeedbackItem(item) {
  const body = normalizeBody(item?.body);
  const normalizedStatus = String(item?.status ?? 'open').toLowerCase();
  const sourceType = item?.sourceType ?? 'unknown';

  if (CLOSED_STATUSES.has(normalizedStatus) || item?.isAnswer === true) {
    return {
      classification: 'closed',
      recommendedAction: 'skip',
      confidence: pickConfidence('high'),
      reason: `status:${normalizedStatus}`,
    };
  }

  if (bodyMatchesAny(body, OUT_OF_SCOPE_PATTERNS)) {
    return {
      classification: 'out-of-scope',
      recommendedAction: 'reply',
      confidence: pickConfidence('high'),
      reason: 'matched out-of-scope phrasing',
    };
  }

  const actionable = bodyMatchesAny(body, ACTIONABLE_PATTERNS);
  const questionOnly = isQuestionOnly(body) && !actionable;

  if (questionOnly) {
    return {
      classification: 'explanation-only',
      recommendedAction: 'reply',
      confidence: pickConfidence('high'),
      reason: 'question without requested code change',
    };
  }

  if (actionable || sourceType === 'pull-request-review') {
    return {
      classification: 'actionable-change',
      recommendedAction: 'change',
      confidence: pickConfidence(actionable ? 'high' : 'medium'),
      reason: actionable
        ? 'matched actionable change phrasing'
        : 'open pull request review defaults to actionable',
    };
  }

  if (bodyMatchesAny(body, ALREADY_SATISFIED_PATTERNS)) {
    return {
      classification: 'already-satisfied',
      recommendedAction: 'reply',
      confidence: pickConfidence('medium'),
      reason: 'matched already-satisfied phrasing',
    };
  }

  return {
    classification: 'needs-triage',
    recommendedAction: 'review',
    confidence: pickConfidence('low'),
    reason: 'no strong classification signal matched',
  };
}

export function buildFeedbackSnapshot({ repoFullName, pullRequest, issues = [], discussions = [] }) {
  const linked = extractGitHubReferences(pullRequest?.body ?? '', repoFullName);
  const items = [
    ...flattenPullRequestItems(pullRequest),
    ...flattenIssueItems(issues),
    ...flattenDiscussionItems(discussions),
  ].map((item) => ({
    ...item,
    ...classifyFeedbackItem(item),
  }));

  return {
    repoFullName,
    generatedAt: new Date().toISOString(),
    pullRequest: pullRequest
      ? {
          number: pullRequest.number,
          url: pullRequest.url,
          title: pullRequest.title,
        }
      : null,
    issues: issues.map((issue) => ({
      number: issue.number,
      url: issue.url,
      title: issue.title,
      state: issue.state,
    })),
    discussions: discussions.map((discussion) => ({
      id: discussion.id ?? null,
      number: discussion.number,
      url: discussion.url,
      title: discussion.title,
      isAnswered: discussion.isAnswered,
    })),
    linked,
    items,
    summary: summarizeItems(items),
  };
}

export function evaluateFeedbackGate(snapshot) {
  const items = snapshot?.items ?? [];
  const blockingItems = items.filter((item) => item.classification !== 'closed');
  return {
    ready: blockingItems.length === 0,
    blockingItems,
    blockingCounts: summarizeBlockingItems(blockingItems),
  };
}

function parseArgs(argv) {
  const options = {
    check: false,
    json: false,
    pretty: false,
    help: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    switch (arg) {
      case '--repo':
        options.repoFullName = argv[index + 1];
        index += 1;
        break;
      case '--pr':
        options.pullRequestNumber = Number.parseInt(argv[index + 1], 10);
        index += 1;
        break;
      case '--input':
        options.inputPath = argv[index + 1];
        index += 1;
        break;
      case '--json':
        options.json = true;
        break;
      case '--check':
        options.check = true;
        break;
      case '--pretty':
        options.pretty = true;
        break;
      case '--help':
      case '-h':
        options.help = true;
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }

  return options;
}

function readSnapshotInput(inputPath) {
  return parseJsonOutput(
    fs.readFileSync(inputPath, 'utf8'),
    `snapshot input ${inputPath}`,
  );
}

function formatBodySnippet(body) {
  const normalized = normalizeBody(body).replace(/\s+/g, ' ');
  return normalized.length > 96 ? `${normalized.slice(0, 93)}...` : normalized;
}

function printHumanSummary(snapshot) {
  const gate = evaluateFeedbackGate(snapshot);
  const prLabel = snapshot.pullRequest
    ? `${snapshot.repoFullName}#${snapshot.pullRequest.number}`
    : snapshot.repoFullName;

  console.log(`GitHub feedback snapshot: ${prLabel}`);
  console.log(
    `Items: ${snapshot.summary.totalItems} | actionable: ${snapshot.summary.classificationCounts['actionable-change'] ?? 0} | explanation: ${snapshot.summary.classificationCounts['explanation-only'] ?? 0} | out-of-scope: ${snapshot.summary.classificationCounts['out-of-scope'] ?? 0} | closed: ${snapshot.summary.classificationCounts.closed ?? 0}`,
  );

  if (snapshot.linked.issues.length > 0) {
    console.log(
      `Linked issues: ${snapshot.linked.issues.map((issue) => `#${issue.number}`).join(', ')}`,
    );
  }
  if (snapshot.linked.discussions.length > 0) {
    console.log(
      `Linked discussions: ${snapshot.linked.discussions
        .map((discussion) => `#${discussion.number}`)
        .join(', ')}`,
    );
  }
  console.log(`Feedback gate: ${gate.ready ? 'ready' : 'blocked'}`);

  for (const item of snapshot.items) {
    console.log(
      `- [${item.recommendedAction}] ${item.sourceType} ${item.authorLogin ?? 'unknown'} ${item.sourceUrl} :: ${formatBodySnippet(item.body)}`,
    );
  }
}

function printUsage() {
  console.log(`Usage:
  node scripts/fetch-github-feedback.mjs --repo <owner/repo> --pr <number> --json
  node scripts/fetch-github-feedback.mjs --repo <owner/repo> --pr <number> --check
  node scripts/fetch-github-feedback.mjs --input <snapshot.json> --json

Options:
  --repo <owner/repo>  GitHub repository nameWithOwner. Defaults to current gh repo.
  --pr <number>        Pull request number. Defaults to current branch PR via gh pr view.
  --input <path>       Read a raw snapshot fixture instead of calling gh.
  --check              Exit non-zero when any non-closed feedback item remains.
  --json               Print JSON output.
  --pretty             Print a human summary.
  --help               Show this help message.
`);
}

function main(argv) {
  const options = parseArgs(argv);
  if (options.help) {
    printUsage();
    return;
  }

  let rawSnapshot;
  if (options.inputPath) {
    rawSnapshot = readSnapshotInput(options.inputPath);
  } else {
    const repoFullName = options.repoFullName ?? resolveCurrentRepoFullName();
    const pullRequestNumber = options.pullRequestNumber ?? resolveCurrentPullRequestNumber();
    rawSnapshot = fetchGitHubFeedbackSnapshot({ repoFullName, pullRequestNumber });
  }

  const snapshot = buildFeedbackSnapshot(rawSnapshot);
  const gate = evaluateFeedbackGate(snapshot);
  const output = {
    ...snapshot,
    gate,
  };

  if (options.pretty || (!options.json && !options.check && process.stdout.isTTY)) {
    printHumanSummary(snapshot);
  } else if (!options.check || options.json) {
    console.log(JSON.stringify(output, null, 2));
  }

  if (options.check && !gate.ready) {
    for (const item of gate.blockingItems) {
      console.error(
        `BLOCKED ${item.classification} ${item.sourceType} ${item.sourceUrl} :: ${formatBodySnippet(item.body)}`,
      );
    }
    process.exit(1);
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2));
}
