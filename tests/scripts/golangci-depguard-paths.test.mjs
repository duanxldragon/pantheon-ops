// Guards the depguard boundary rule against silent drift.
//
// The business boundary rule is defence in depth behind
// scripts/harness/check-boundaries.mjs. It only works when the `pkg` entries
// match real import paths, so a stale value (for example a bare
// "pantheon-base/modules/system" prefix while the module is
// "github.com/<org>/pantheon-base/backend") makes the rule a no-op without any
// visible failure.

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);

const REQUIRED_DENY_SUFFIXES = [
  "/modules/system",
  "/modules/auth",
  "/modules/platform",
];

function readGoModulePath() {
  const goMod = fs.readFileSync(
    path.join(repoRoot, "backend", "go.mod"),
    "utf8",
  );
  const match = goMod.match(/^module\s+(\S+)\s*$/m);
  assert.ok(match, "backend/go.mod must declare a module path");
  return match[1];
}

function readBusinessBoundaryDenyPaths() {
  const config = fs.readFileSync(
    path.join(repoRoot, "backend", ".golangci.yml"),
    "utf8",
  );
  const start = config.indexOf("business-boundary:");
  assert.ok(
    start >= 0,
    ".golangci.yml must declare the business-boundary depguard rule",
  );

  // Bound the rule body so later blocks (gosec, goconst, ...) cannot contribute
  // unrelated `pkg` entries if they ever add any.
  const rest = config.slice(start);
  const end = rest.search(/\n {4}[a-z][a-z0-9-]*:\s*$/m);
  const block = end >= 0 ? rest.slice(0, end) : rest;

  return [...block.matchAll(/^\s*-\s*pkg:\s*(\S+)\s*$/gm)].map(
    (match) => match[1],
  );
}

test("depguard business boundary deny paths use the real Go module path", () => {
  const modulePath = readGoModulePath();
  const denyPaths = readBusinessBoundaryDenyPaths();

  assert.ok(
    denyPaths.length > 0,
    "business-boundary rule must deny at least one package prefix",
  );

  for (const denyPath of denyPaths) {
    assert.ok(
      denyPath === modulePath || denyPath.startsWith(`${modulePath}/`),
      `depguard deny path "${denyPath}" does not start with the Go module path "${modulePath}" and can never match an import`,
    );
  }
});

test("depguard business boundary covers system, auth and platform internals", () => {
  const modulePath = readGoModulePath();
  const denyPaths = new Set(readBusinessBoundaryDenyPaths());

  for (const suffix of REQUIRED_DENY_SUFFIXES) {
    assert.ok(
      denyPaths.has(`${modulePath}${suffix}`),
      `business-boundary rule must deny "${modulePath}${suffix}"`,
    );
  }
});
