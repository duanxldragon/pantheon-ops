import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

export function parseJsonFile(filePath, description) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    throw new Error(`${description} is invalid JSON: ${error.message}`);
  }
}

function parseRepoParts(repoFullName) {
  const [owner, repo] = String(repoFullName ?? '').split('/');
  if (!owner || !repo) {
    throw new Error(`invalid repo full name: ${repoFullName}`);
  }
  return { owner, repo };
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

function ensureItem(snapshot, itemId) {
  const item = (snapshot.items ?? []).find((entry) => entry.id === itemId);
  if (!item) {
    throw new Error(`snapshot item not found: ${itemId}`);
  }
  return item;
}

function ensureDiscussion(snapshot, discussionNumber) {
  const discussion = (snapshot.discussions ?? []).find(
    (entry) => entry.number === discussionNumber,
  );
  if (!discussion) {
    throw new Error(`snapshot discussion not found: #${discussionNumber}`);
  }
  return discussion;
}

function uniqueValues(values = []) {
  return [...new Set(values)];
}

function findExistingAnsweredDiscussionReply(snapshot, discussionNumber, parentCommentId) {
  return (snapshot.items ?? []).find(
    (entry) =>
      entry.sourceType === 'discussion-comment' &&
      entry.discussionNumber === discussionNumber &&
      entry.parentCommentId === parentCommentId &&
      entry.isAnswer === true,
  );
}

export function planWritebackActions(snapshot, plan) {
  const actions = [];
  const replies = plan?.replies ?? {};
  const repliedDiscussionItemIdsToAnswer = new Set(
    uniqueValues(plan?.markRepliedDiscussionItemIdsAsAnswer ?? []),
  );

  for (const [itemId, body] of Object.entries(replies)) {
    const item = ensureItem(snapshot, itemId);
    if (!String(body ?? '').trim()) {
      throw new Error(`reply body is empty for item: ${itemId}`);
    }
    if (item.sourceType === 'pull-request-review') {
      if (!Number.isInteger(item.commentDatabaseId)) {
        throw new Error(`pull request review comment is missing database id: ${itemId}`);
      }
      actions.push({
        type: 'reply-review-comment',
        itemId,
        pullRequestNumber: item.pullRequestNumber,
        commentDatabaseId: item.commentDatabaseId,
        body: String(body),
      });
      continue;
    }
    if (item.sourceType === 'issue-comment') {
      actions.push({
        type: 'reply-issue-comment',
        itemId,
        issueNumber: item.issueNumber,
        body: String(body),
      });
      continue;
    }
    if (item.sourceType === 'discussion-comment') {
      ensureDiscussion(snapshot, item.discussionNumber);
      actions.push({
        type: 'reply-discussion-comment',
        itemId,
        discussionNumber: item.discussionNumber,
        replyToId: item.id,
        body: String(body),
        markReplyAsAnswer: repliedDiscussionItemIdsToAnswer.has(itemId),
      });
      continue;
    }
    throw new Error(`unsupported reply source type: ${item.sourceType}`);
  }

  for (const threadId of uniqueValues(plan?.resolveReviewThreadIds ?? [])) {
    const item = (snapshot.items ?? []).find((entry) => entry.threadId === threadId);
    if (!item) {
      throw new Error(`review thread not found in snapshot: ${threadId}`);
    }
    actions.push({
      type: 'resolve-review-thread',
      threadId,
      pullRequestNumber: item.pullRequestNumber,
    });
  }

  for (const commentId of uniqueValues(plan?.markDiscussionAnswerIds ?? [])) {
    const item = ensureItem(snapshot, commentId);
    actions.push({
      type: 'mark-discussion-comment-as-answer',
      commentId,
      discussionNumber: item.discussionNumber,
    });
  }

  for (const issueNumber of uniqueValues(plan?.closeIssueNumbers ?? [])) {
    actions.push({
      type: 'close-issue',
      issueNumber,
    });
  }

  return actions;
}

export function buildGhOperationSpecs(repoFullName, actions) {
  const specs = [];
  const { owner, repo } = parseRepoParts(repoFullName);

  for (const action of actions) {
    switch (action.type) {
      case 'reply-review-comment':
        specs.push({
          type: action.type,
          description: `Reply to PR review comment ${action.commentDatabaseId}`,
          args: [
            'api',
            '-X',
            'POST',
            `repos/${owner}/${repo}/pulls/${action.pullRequestNumber}/comments/${action.commentDatabaseId}/replies`,
            '-f',
            `body=${action.body}`,
          ],
        });
        break;
      case 'reply-issue-comment':
        specs.push({
          type: action.type,
          description: `Reply on issue #${action.issueNumber}`,
          args: [
            'api',
            '-X',
            'POST',
            `repos/${owner}/${repo}/issues/${action.issueNumber}/comments`,
            '-f',
            `body=${action.body}`,
          ],
        });
        break;
      case 'reply-discussion-comment':
        specs.push({
          type: action.type,
          description: action.markReplyAsAnswer
            ? `Reply in discussion #${action.discussionNumber} and mark reply as answer`
            : `Reply in discussion #${action.discussionNumber}`,
          args: [
            'api',
            'graphql',
            '-f',
            'query=mutation($replyToId:ID!, $body:String!) { addDiscussionComment(input:{replyToId:$replyToId, body:$body}) { comment { id } } }',
            '-f',
            `replyToId=${action.replyToId}`,
            '-f',
            `body=${action.body}`,
          ],
          markReplyAsAnswer: action.markReplyAsAnswer === true,
          followUpArgs:
            action.markReplyAsAnswer === true
              ? [
                  'api',
                  'graphql',
                  '-f',
                  'query=mutation($id:ID!) { markDiscussionCommentAsAnswer(input:{id:$id}) { comment { id isAnswer } } }',
                ]
              : null,
        });
        break;
      case 'resolve-review-thread':
        specs.push({
          type: action.type,
          description: `Resolve review thread ${action.threadId}`,
          args: [
            'api',
            'graphql',
            '-f',
            'query=mutation($threadId:ID!) { resolveReviewThread(input:{threadId:$threadId}) { thread { id isResolved } } }',
            '-f',
            `threadId=${action.threadId}`,
          ],
        });
        break;
      case 'mark-discussion-comment-as-answer':
        specs.push({
          type: action.type,
          description: `Mark discussion comment ${action.commentId} as answer`,
          args: [
            'api',
            'graphql',
            '-f',
            'query=mutation($id:ID!) { markDiscussionCommentAsAnswer(input:{id:$id}) { comment { id isAnswer } } }',
            '-f',
            `id=${action.commentId}`,
          ],
        });
        break;
      case 'close-issue':
        specs.push({
          type: action.type,
          description: `Close issue #${action.issueNumber}`,
          args: [
            'api',
            '-X',
            'PATCH',
            `repos/${owner}/${repo}/issues/${action.issueNumber}`,
            '-f',
            'state=closed',
          ],
        });
        break;
      default:
        throw new Error(`unsupported action type: ${action.type}`);
    }
  }

  return specs;
}

export function executeGhOperationSpecs(specs) {
  const results = [];
  for (const spec of specs) {
    const stdout = runGhCommand(spec.args, spec.description);
    let followUpStdout = null;
    if (spec.markReplyAsAnswer === true && spec.followUpArgs) {
      let commentId;
      try {
        const payload = JSON.parse(stdout);
        commentId = payload?.data?.addDiscussionComment?.comment?.id ?? null;
      } catch (error) {
        commentId = null;
      }
      if (!commentId) {
        throw new Error(`${spec.description} did not return a discussion comment id to mark as answer`);
      }
      followUpStdout = runGhCommand(
        [...spec.followUpArgs, '-f', `id=${commentId}`],
        `${spec.description} (mark answer)`,
      );
    }
    results.push({
      type: spec.type,
      description: spec.description,
      stdout: stdout.trim(),
      followUpStdout: followUpStdout ? followUpStdout.trim() : null,
    });
  }
  return results;
}

function parseArgs(argv) {
  const options = {
    dryRun: false,
    json: false,
    help: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    switch (arg) {
      case '--snapshot':
        options.snapshotPath = argv[index + 1];
        index += 1;
        break;
      case '--plan':
        options.planPath = argv[index + 1];
        index += 1;
        break;
      case '--dry-run':
        options.dryRun = true;
        break;
      case '--json':
        options.json = true;
        break;
      case '--help':
      case '-h':
        options.help = true;
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }

  if (!options.snapshotPath) {
    throw new Error('missing --snapshot <snapshot.json>');
  }
  if (!options.planPath) {
    throw new Error('missing --plan <plan.json>');
  }

  return options;
}

function printUsage() {
  console.log(`Usage:
  node scripts/address-github-feedback.mjs --snapshot <snapshot.json> --plan <plan.json> --dry-run --json
  node scripts/address-github-feedback.mjs --snapshot <snapshot.json> --plan <plan.json>

Options:
  --snapshot <path>    Snapshot JSON generated from fetch-github-feedback.
  --plan <path>        Writeback action plan JSON.
  --dry-run            Print planned gh operations without executing them.
  --json               Emit JSON output.
  --help               Show this help message.
`);
}

export function main(argv) {
  const options = parseArgs(argv);
  if (options.help) {
    printUsage();
    return;
  }

  const snapshot = parseJsonFile(options.snapshotPath, 'snapshot');
  const plan = parseJsonFile(options.planPath, 'plan');
  const actions = planWritebackActions(snapshot, plan);
  const operations = buildGhOperationSpecs(snapshot.repoFullName, actions);
  const output = {
    repoFullName: snapshot.repoFullName,
    pullRequest: snapshot.pullRequest ?? null,
    actions,
    operations,
  };

  if (options.dryRun) {
    console.log(JSON.stringify(output, null, 2));
    return;
  }

  const results = executeGhOperationSpecs(operations);
  const executedOutput = {
    ...output,
    results,
  };

  if (options.json) {
    console.log(JSON.stringify(executedOutput, null, 2));
    return;
  }

  console.log(`Applied ${results.length} GitHub feedback writeback operations.`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2));
}
