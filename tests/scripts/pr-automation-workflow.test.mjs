import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/pr-automation.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

test('pr automation validates governance and GitHub feedback before enabling auto-merge', () => {
  assert.match(
    workflowSource,
    /governance-prereq:\s*\n[\s\S]*name:\s*PR Governance Prereq/i,
    'pr automation should define a governance prerequisite job',
  );
  assert.match(
    workflowSource,
    /Validate PR governance body[\s\S]*node scripts\/check-pr-governance\.mjs --event "\$GITHUB_EVENT_PATH"/i,
    'pr automation should validate the pull request body against repository governance rules',
  );
  assert.match(
    workflowSource,
    /feedback-prereq:\s*\n[\s\S]*name:\s*GitHub Feedback Prereq/i,
    'pr automation should define a GitHub feedback prerequisite job',
  );
  assert.match(
    workflowSource,
    /Check GitHub feedback gate[\s\S]*npm run check:github-feedback -- --repo "\$GITHUB_REPOSITORY" --pr "\$PR_NUMBER"/i,
    'pr automation should run the unified GitHub feedback gate before enabling auto-merge',
  );
  assert.match(
    workflowSource,
    /automate-solo-pr:[\s\S]*needs:[\s\S]*-\s*governance-prereq[\s\S]*-\s*feedback-prereq/i,
    'auto-merge job should depend on governance and GitHub feedback prerequisites',
  );
  assert.match(
    workflowSource,
    /needs\.governance-prereq\.outputs\.pr_body_ready == 'true'/i,
    'auto-merge should only be considered after the governance prerequisite succeeds',
  );
  assert.match(
    workflowSource,
    /needs\.feedback-prereq\.outputs\.feedback_ready == 'true'/i,
    'auto-merge should only be considered after the GitHub feedback prerequisite succeeds',
  );
});

test('pr automation deletes merged solo feature branches on pull request close', () => {
  assert.match(
    workflowSource,
    /pull_request:\s*\n\s*types:\s*\n[\s\S]*-\s*closed/i,
    'pr automation should listen for pull_request.closed events',
  );
  assert.match(
    workflowSource,
    /delete-merged-head-branch:\s*\n[\s\S]*name:\s*Delete Merged Head Branch/i,
    'pr automation should define a merged-branch cleanup job',
  );
  assert.match(
    workflowSource,
    /github\.event\.action == 'closed'[\s\S]*github\.event\.pull_request\.merged == true/i,
    'merged-branch cleanup should only run for merged pull requests',
  );
  assert.match(
    workflowSource,
    /github\.event\.pull_request\.head\.repo\.full_name == github\.repository/i,
    'merged-branch cleanup should only delete branches from the same repository',
  );
  assert.match(
    workflowSource,
    /gh api -X DELETE "repos\/\$GH_REPO\/git\/refs\/heads\/\$HEAD_REF"/i,
    'merged-branch cleanup should delete the remote head branch explicitly',
  );
});
