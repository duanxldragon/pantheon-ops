import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { evaluate } from "../../scripts/harness/check-checker-wiring.mjs";

const checkerPath = fileURLToPath(
  new URL("../../scripts/harness/check-checker-wiring.mjs", import.meta.url),
);

function makeRoot() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "checker-wiring-"));
  fs.mkdirSync(path.join(root, "scripts", "harness"), { recursive: true });
  fs.mkdirSync(path.join(root, ".github", "workflows"), { recursive: true });

  for (const name of [
    "check-wired.mjs",
    "check-in-workflow.mjs",
    "check-unwired.mjs",
    // Uses the documented mirror-only name so the exception path is exercised.
    "check-doc-frontmatter.mjs",
  ]) {
    fs.writeFileSync(path.join(root, "scripts", "harness", name), "");
  }

  writePackageJson(root, "node scripts/harness/check-wired.mjs");
  fs.writeFileSync(
    path.join(root, ".github", "workflows", "quality.yml"),
    "jobs:\n  docs:\n    steps:\n      - run: node scripts/harness/check-in-workflow.mjs\n",
  );

  return root;
}

function writePackageJson(root, wiredScript) {
  fs.writeFileSync(
    path.join(root, "package.json"),
    `${JSON.stringify({ scripts: { "check:wired": wiredScript } }, null, 2)}\n`,
  );
}

function run(root) {
  try {
    const stdout = execFileSync(
      process.execPath,
      [checkerPath, "--root", root],
      {
        encoding: "utf8",
        stdio: ["ignore", "pipe", "pipe"],
        windowsHide: true,
      },
    );
    return { code: 0, output: stdout };
  } catch (error) {
    return {
      code: error.status,
      output: `${error.stdout ?? ""}${error.stderr ?? ""}`,
    };
  }
}

test("fails when a checker is referenced by neither package.json nor a workflow", () => {
  const root = makeRoot();

  const result = run(root);

  assert.equal(result.code, 1);
  assert.match(
    result.output,
    /NOT WIRED\s+scripts\/harness\/check-unwired\.mjs/,
  );
  assert.doesNotMatch(
    result.output,
    /NOT WIRED\s+scripts\/harness\/check-wired\.mjs/,
  );
  assert.doesNotMatch(
    result.output,
    /NOT WIRED\s+scripts\/harness\/check-in-workflow\.mjs/,
  );
});

test("accepts a documented exception and ignores it in the failure count", () => {
  const root = makeRoot();

  const result = run(root);

  assert.match(
    result.output,
    /exception\s+scripts\/harness\/check-doc-frontmatter\.mjs/,
  );
  assert.match(
    result.output,
    /4 checkers \/ 2 wired \/ 1 exception \/ 1 unwired/,
  );
});

test("passes once the unwired checker gains a package.json script", () => {
  const root = makeRoot();
  writePackageJson(
    root,
    "node scripts/harness/check-wired.mjs && node scripts/harness/check-unwired.mjs",
  );

  const result = run(root);

  assert.equal(result.code, 0);
  assert.match(
    result.output,
    /4 checkers \/ 3 wired \/ 1 exception \/ 0 unwired/,
  );
});

test("reports machine-readable JSON for evidence capture", () => {
  const root = makeRoot();
  // Fully wire the tree so the JSON report is written on a passing run.
  writePackageJson(
    root,
    "node scripts/harness/check-wired.mjs && node scripts/harness/check-unwired.mjs",
  );

  const stdout = execFileSync(
    process.execPath,
    [checkerPath, "--root", root, "--json"],
    {
      encoding: "utf8",
      windowsHide: true,
    },
  );
  const report = JSON.parse(stdout);

  assert.equal(report.total, 4);
  assert.equal(report.wired, 3);
  assert.deepEqual(report.unwired, []);
});

test("metadata, workflow names and standalone comments do not create wiring", () => {
  const root = makeRoot();
  fs.writeFileSync(
    path.join(root, "package.json"),
    JSON.stringify({
      description: "scripts/harness/check-unwired.mjs",
      scripts: { "check:wired": "node scripts/harness/check-wired.mjs" },
    }),
  );
  fs.writeFileSync(
    path.join(root, ".github/workflows/quality.yml"),
    "# scripts/harness/check-unwired.mjs\njobs:\n  docs:\n    name: scripts/harness/check-unwired.mjs\n    steps:\n      - run: |\n          # scripts/harness/check-unwired.mjs\n          node scripts/harness/check-in-workflow.mjs\n",
  );
  const result = evaluate(root).find(
    (item) => item.name === "check-unwired.mjs",
  );
  assert.equal(result.status, "unwired");
});

test("npm indirection distinguishes CI references from manual-only tools", () => {
  const root = makeRoot();
  fs.writeFileSync(
    path.join(root, "package.json"),
    JSON.stringify({
      scripts: {
        "check:wired": "node scripts/harness/check-wired.mjs",
        "check:manual": "node scripts/harness/check-unwired.mjs",
        verify: "npm run check:wired && npm run verify",
      },
    }),
  );
  fs.writeFileSync(
    path.join(root, ".github/workflows/quality.yml"),
    "jobs:\n  docs:\n    steps:\n      - run: npm run verify\n",
  );
  const results = evaluate(root);
  assert.equal(
    results.find((item) => item.name === "check-wired.mjs").invocation,
    "ci-reference",
  );
  assert.equal(
    results.find((item) => item.name === "check-unwired.mjs").invocation,
    "manual-only",
  );
});

test("inline comments, echo data and another script argument cannot fabricate an invocation", (t) => {
  const root = makeRoot();
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  for (const command of [
    "echo ok # scripts/harness/check-unwired.mjs",
    "echo scripts/harness/check-unwired.mjs",
    'echo "node scripts/harness/check-unwired.mjs"',
    "node other.mjs scripts/harness/check-unwired.mjs",
    "node -e \"console.log('scripts/harness/check-unwired.mjs')\"",
  ]) {
    writePackageJson(root, command);
    fs.writeFileSync(
      path.join(root, ".github/workflows/quality.yml"),
      `jobs:\n  docs:\n    steps:\n      - run: ${command}\n`,
    );
    assert.equal(
      evaluate(root).find((item) => item.name === "check-unwired.mjs").status,
      "unwired",
      command,
    );
  }
});

test("commented npm aliases stay manual while quoted hashes preserve actual command separators", (t) => {
  const root = makeRoot();
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  writePackageJson(root, "node scripts/harness/check-unwired.mjs");
  const workflow = path.join(root, ".github/workflows/quality.yml");
  fs.writeFileSync(
    workflow,
    "jobs:\n  docs:\n    steps:\n      - run: echo ready # npm run check:wired\n",
  );
  assert.equal(
    evaluate(root).find((item) => item.name === "check-unwired.mjs").invocation,
    "manual-only",
  );
  fs.writeFileSync(
    workflow,
    'jobs:\n  docs:\n    steps:\n      - run: | # npm run missing\n          echo "# quoted hash" && node "./scripts/harness/check-unwired.mjs" # explanation\n',
  );
  assert.equal(
    evaluate(root).find((item) => item.name === "check-unwired.mjs").invocation,
    "ci-reference",
  );
});

test("folded YAML joins command lines before inspection and fails visibly on unsupported folding", (t) => {
  const root = makeRoot();
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const workflow = path.join(root, ".github/workflows/quality.yml");
  fs.writeFileSync(
    workflow,
    "jobs:\n  docs:\n    steps:\n      - run: >\n          echo\n          node scripts/harness/check-unwired.mjs\n        name: folded echo\n",
  );
  assert.equal(
    evaluate(root).find((item) => item.name === "check-unwired.mjs").status,
    "unwired",
  );
  fs.writeFileSync(
    workflow,
    "jobs:\n  docs:\n    steps:\n      - run: >-\n          node\n          scripts/harness/check-unwired.mjs\n",
  );
  assert.equal(
    evaluate(root).find((item) => item.name === "check-unwired.mjs").invocation,
    "ci-reference",
  );
  fs.writeFileSync(
    workflow,
    "jobs:\n  docs:\n    steps:\n      - run: >\n          echo\n\n          node scripts/harness/check-unwired.mjs\n",
  );
  assert.throws(() => evaluate(root), /Complex folded run/);
});
