import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/ci.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

// The advisory bench job hands an artifact (bench.txt) from the benchmark step
// to the summary step. Both steps must therefore run in the same
// working-directory: the writer used to `tee bench.txt` inside backend/ while
// the `cat bench.txt` step ran at the repository root, so the summary step
// failed on every run and painted main red even though the benchmarks passed.
function jobBlock(jobName) {
  const start = workflowSource.search(new RegExp(`^  ${jobName}:$`, 'm'));
  assert.notEqual(start, -1, `ci.yml should still define the ${jobName} job`);
  const rest = workflowSource.slice(start + 1);
  const nextJobOffset = rest.search(/^  [a-z][a-z0-9-]*:$/m);
  return nextJobOffset === -1 ? rest : rest.slice(0, nextJobOffset);
}

function stepBlocks(jobSource) {
  // Steps in ci.yml are list items indented six spaces under a job's `steps:` key.
  return jobSource.split(/\n(?= {6}- )/).filter((block) => /^\s*-\s*name:/.test(block));
}

function workingDirectoryOf(block) {
  const match = block.match(/^\s*working-directory:\s*(\S+)\s*$/m);
  return match ? match[1] : '';
}

test('bench-perf-smoke keeps the bench artifact and its summary step in the same working-directory', () => {
  const artifactSteps = stepBlocks(jobBlock('bench-perf-smoke')).filter((block) =>
    /bench\.txt/.test(block),
  );

  assert.ok(
    artifactSteps.length >= 2,
    'bench-perf-smoke should both write and report bench.txt (writer + summary step)',
  );

  const directories = artifactSteps.map((block) => workingDirectoryOf(block));
  assert.ok(
    directories.every((directory) => directory.length > 0),
    `every step touching bench.txt must pin working-directory; got ${JSON.stringify(directories)}`,
  );
  assert.equal(
    new Set(directories).size,
    1,
    `bench.txt steps must agree on working-directory; got ${JSON.stringify(directories)}`,
  );
});

test('bench-perf-smoke stays advisory', () => {
  const job = jobBlock('bench-perf-smoke');

  assert.match(
    job,
    /name:\s*Bench Perf Smoke \(advisory\)/,
    'the job must stay labelled advisory while it is report-only',
  );
  assert.match(
    job,
    /-benchtime\s+1x/,
    'the advisory job should keep the cheap single-iteration bench sampling',
  );
  assert.doesNotMatch(
    jobBlock('ci-summary'),
    /bench-perf-smoke/,
    'bench-perf-smoke must not be added to ci-summary (that would make it blocking)',
  );
});
