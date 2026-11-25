import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { scanInheritance } from "../../scripts/harness/check-inheritance-contract.mjs";

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "inheritance-command-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const write = (name, text) => {
    const file = path.join(root, name);
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, text);
  };
  write(
    "pantheon-ops/package.json",
    JSON.stringify({ scripts: { "rebuild:from-base": "node rebuild.mjs" } }),
  );
  write(
    "docs/WORKSPACE_INHERITANCE.md",
    "`pantheon-base`: the only authority\n`pantheon-ops`: a derived business repository\nChange base rules in `pantheon-base`, then let business repositories upgrade\nnpm run rebuild:from-base\n",
  );
  write(
    "pantheon-ops/CLAUDE.md",
    "pantheon-ops inherits pantheon-base as its foundation\nBusiness work reading order\n../docs/WORKSPACE_INHERITANCE.md\ndocs/PROJECT_INHERITANCE.md\n../pantheon-base/DESIGN.md\n../pantheon-base/AGENTS.md\nDo not fix platform or system-domain drift locally in pantheon-ops",
  );
  const commit = "a".repeat(40);
  write(
    "pantheon-ops/docs/PROJECT_INHERITANCE.md",
    `Base repository: \`../pantheon-base\`\nBase version: \`pantheon-base-v0.18.9\` (\`${commit}\`)\nbusiness/cmdb business/deploy\nIf a foundation rule must change, update \`pantheon-base\` first`,
  );
  write(
    "pantheon-ops/foundation-release.lock.json",
    JSON.stringify({
      releaseVersion: "pantheon-base-v0.18.9",
      baseCommit: commit,
    }),
  );
  write(
    ".github/pull_request_template.md",
    "Base/ops inheritance Base version checked generic drift pseudo-drift business-only",
  );
  return { root, write };
}

test("valid commands and exact lock anchors pass; removed commands fail despite retained markers", (t) => {
  const { root, write } = fixture(t);
  assert.deepEqual(scanInheritance(root).findings, []);
  write(
    "docs/WORKSPACE_INHERITANCE.zh.md",
    "npm run upgrade:foundation:local-apply\n",
  );
  assert.ok(
    scanInheritance(root).findings.some((item) =>
      item.reason.includes("documented Ops command does not exist"),
    ),
  );
});

test("a plausible version marker cannot hide lock drift or malformed package metadata", (t) => {
  const { root, write } = fixture(t);
  write(
    "pantheon-ops/foundation-release.lock.json",
    JSON.stringify({
      releaseVersion: "pantheon-base-v0.18.8",
      baseCommit: "b".repeat(40),
    }),
  );
  assert.ok(
    scanInheritance(root).findings.some((item) =>
      item.reason.includes("must match foundation-release"),
    ),
  );
  write("pantheon-ops/package.json", "{");
  assert.ok(
    scanInheritance(root).findings.some((item) =>
      item.reason.includes("invalid package.json"),
    ),
  );
});
