import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { buildCandidateArtifact } from "../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs";
import * as producer from "../../../pantheon-base/scripts/foundation-release/candidate-descriptor.mjs";
import * as consumer from "./foundation-candidate.mjs";
import { rebuildFromCandidate } from "./rebuild-from-candidate.mjs";

test("declared metadata count accepts 100000 unique files and refuses 100001", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ops-metadata-count-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const directory = path.join(root, ".harness");
  const seed = path.join(root, "seed.txt");
  fs.mkdirSync(directory);
  fs.writeFileSync(seed, "");
  // Project synthetic leaf names onto one real zero-byte file. The production
  // traversal, hashing, stamp checks, unique inventory and bounds still run;
  // the fixture avoids creating 100001 files on the developer filesystem.
  const ordinary = fs.lstatSync(seed);
  const bigint = fs.lstatSync(seed, { bigint: true });
  const fd = fs.openSync(seed, "r");
  const close = fs.closeSync;
  const leaf = file => typeof file === "string" &&
    path.dirname(file) === directory && /^file-\d{6}\.txt$/u.test(path.basename(file));
  const names = Array.from({ length: 100000 }, (_, index) =>
    `file-${String(index).padStart(6, "0")}.txt`);
  const lstat = fs.lstatSync;
  const readdir = fs.readdirSync;
  const open = fs.openSync;
  const fstat = fs.fstatSync;
  const exists = fs.existsSync;
  const directoryStats = new Map();
  const existingDirectories = new Map();
  const originals = { lstatSync: lstat, readdirSync: readdir, openSync: open,
    fstatSync: fstat, closeSync: close, existsSync: exists };
  fs.existsSync = file => {
    if (leaf(file)) return true;
    if (!existingDirectories.has(file)) existingDirectories.set(file, exists(file));
    return existingDirectories.get(file);
  };
  fs.lstatSync = (file, options) => {
    if (leaf(file)) return options?.bigint ? bigint : ordinary;
    const key = `${file}:${!!options?.bigint}`;
    if (!directoryStats.has(key)) directoryStats.set(key, lstat(file, options));
    return directoryStats.get(key);
  };
  fs.readdirSync = (file, ...args) => file === directory ? [...names] : readdir(file, ...args);
  fs.openSync = (file, ...args) => leaf(file) ? fd : open(file, ...args);
  fs.fstatSync = (file, options) => file === fd ? (options?.bigint ? bigint : ordinary) : fstat(file, options);
  fs.closeSync = file => { if (file !== fd) close(file); };
  try {
    const captured = consumer.captureInputs(root, [".harness", ".harness/file-000000.txt"]);
    assert.deepEqual(consumer.METADATA_LIMITS, { files: 100000 });
    assert.ok(Object.isFrozen(consumer.METADATA_LIMITS));
    assert.equal(consumer.LIMITS.files, 50000);
    assert.equal(captured.files.length, 100000);
    assert.equal(captured.stamps.length, 100001);
    assert.deepEqual(captured.files.map(file => file.path), names.map(name => `.harness/${name}`));
    const emptySha256 = consumer.sha256("");
    assert.ok(captured.files.every(file => file.bytes === 0 && file.sha256 === emptySha256));
    assert.throws(() => consumer.assertInventory(captured.files), /candidate file count/);
    names.push("file-100000.txt");
    assert.throws(() => consumer.captureInputs(root, [".harness"]), /Declared metadata input count exceeds limit/);
  } finally {
    Object.assign(fs, originals);
    close(fd);
  }
});

test("candidate inventory and tar retain the independent 50000 file boundary", (t) => {
  const f = fixture(t);
  const original = producer.verifyCandidateArchive(
    fs.readFileSync(f.candidate.artifactPath), f.candidate.descriptor);
  const entries = [...original, ...Array.from({ length: 50000 - original.length }, (_, index) => ({
    path: `docs/count-${String(index).padStart(6, "0")}.txt`, bytes: Buffer.alloc(0),
  }))].sort((a, b) => a.path < b.path ? -1 : a.path > b.path ? 1 : 0);
  const files = entries.map(entry => ({ path: entry.path, bytes: entry.bytes.length,
    sha256: consumer.sha256(entry.bytes) }));
  for (const validator of [consumer, producer]) {
    assert.doesNotThrow(() => validator.assertInventory(files));
    assert.throws(() => validator.assertInventory([...files, files[0]]), /candidate file count/);
  }
  const archive = producer.encodeCandidateArchive(entries);
  const aggregate = consumer.aggregateSha256(files);
  const descriptor = { ...f.candidate.descriptor, files, aggregateSha256: aggregate,
    candidateIdentity: `candidate-${aggregate}`,
    artifact: { assetName: `candidate-${aggregate}.tar`, sha256: consumer.sha256(archive) } };
  for (const validator of [consumer, producer])
    assert.equal(validator.verifyCandidateArchive(archive, descriptor).length, 50000);
  // Append a valid extra regular header, retaining valid descriptor and archive
  // pins, to reach the tar count refusal before duplicate/inventory mismatch.
  const extra = producer.encodeCandidateArchive([{ path: "docs/extra.txt", bytes: Buffer.alloc(0) }]);
  const overflow = Buffer.concat([archive.subarray(0, -1024), extra]);
  const overflowDescriptor = { ...descriptor,
    artifact: { ...descriptor.artifact, sha256: consumer.sha256(overflow) } };
  for (const validator of [consumer, producer])
    assert.throws(() => validator.verifyCandidateArchive(overflow, overflowDescriptor), /archive entry count exceeds limit/);
});

test("captured retained archives stream within explicit metadata bounds", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ops-captured-archive-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const relative = ".foundation/releases/pantheon-base-v0.18.6/repo.tar";
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, "");
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 1);
  assert.throws(() => consumer.readStableFile(file), /bounded/);
  const originalRead = fs.readSync;
  let reads = 0;
  t.mock.method(fs, "readSync", function (fd, buffer, offset, length, position) {
    assert.ok(buffer.length <= 1024 * 1024);
    assert.ok(length <= 1024 * 1024);
    reads++;
    return originalRead.call(this, fd, buffer, offset, length, position);
  });
  const before = consumer.captureInputs(root, [".foundation"]);
  assert.ok(reads > 32);
  assert.equal(before.files[0].bytes, consumer.LIMITS.fileBytes + 1);
  assert.deepEqual(consumer.captureInputs(root, [".foundation"]), before);
  assert.equal(consumer.LIMITS.fileBytes, 32 * 1024 * 1024);
  assert.equal(consumer.LIMITS.archiveBytes, 512 * 1024 * 1024);
  fs.truncateSync(file, consumer.LIMITS.archiveBytes + 1);
  assert.throws(() => consumer.captureInputs(root, [".foundation"]), /bounded/);
  fs.truncateSync(file, consumer.LIMITS.fileBytes + 1);
  for (const invalid of ["backend/repo.tar", ".foundation/repo.tar",
    ".foundation/releases/private/repo.tar", ".foundation/releases/pantheon-base-v0.18.6/source.go",
    ".foundation/releases/pantheon-base-v0.18.6/foundation-release-pantheon-base-v0.17.6.tgz"]) {
    const target = path.join(root, invalid);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.renameSync(file, target);
    assert.throws(() => consumer.captureInputs(root, [invalid]), /bounded/);
    fs.renameSync(target, file);
  }
  for (const valid of [".foundation/releases/pantheon-base-v0.18.6/foundation-release-pantheon-base-v0.18.6.tgz",
    ".foundation/releases/foundation-release-pantheon-base-v0.18.6.tgz"]) {
    const target = path.join(root, valid);
    fs.renameSync(file, target);
    assert.equal(consumer.captureInputs(root, [valid]).files[0].bytes, consumer.LIMITS.fileBytes + 1);
    fs.renameSync(target, file);
  }
  fs.linkSync(file, path.join(root, "hard-link"));
  assert.throws(() => consumer.captureInputs(root, [relative]), /hard links/);
  fs.unlinkSync(path.join(root, "hard-link"));
  assert.throws(() => consumer.captureInputs(root, [".foundation/../escape"]), /Unsafe/);
  assert.throws(() => consumer.captureInputs(root, [".foundation\\escape"]), /Unsafe/);
  fs.symlinkSync(path.dirname(file), path.join(root, "redirect"), "junction");
  assert.throws(() => consumer.captureInputs(root, ["redirect"]), /real directories/);
});

test("chunked capture rejects actual file changes while reading", async (t) => {
  for (const attack of ["truncate", "grow", "mutate", "replace", "short-read", "open-replace"]) {
    await t.test(attack, (t) => {
      const root = fs.mkdtempSync(path.join(os.tmpdir(), "ops-capture-mutation-"));
      t.after(() => fs.rmSync(root, { recursive: true, force: true }));
      const relative = ".foundation/releases/pantheon-base-v0.18.6/repo.tar";
      const file = path.join(root, relative);
      fs.mkdirSync(path.dirname(file), { recursive: true });
      fs.writeFileSync(file, Buffer.alloc(2 * 1024 * 1024, 65));
      const mutate = () => {
        if (attack === "truncate") fs.truncateSync(file, 1);
        if (attack === "grow") fs.appendFileSync(file, "extra");
        if (attack === "mutate") {
          const before = fs.statSync(file);
          const fd = fs.openSync(file, "r+");
          fs.writeSync(fd, Buffer.from("B"), 0, 1, 0);
          fs.closeSync(fd);
          // Rapid Windows writes may share a timestamp tick. Make this actual
          // on-disk modification observable to the existing stamp contract.
          fs.utimesSync(file, before.atime, new Date(before.mtimeMs + 2000));
        }
        if (attack === "replace" || attack === "open-replace") {
          fs.renameSync(file, file + ".original");
          fs.copyFileSync(file + ".original", file);
        }
      };
      let triggered = false;
      const original = attack === "open-replace" ? fs.openSync : fs.readSync;
      t.mock.method(fs, attack === "open-replace" ? "openSync" : "readSync", function (...args) {
        const result = original.apply(this, args);
        if (!triggered && (attack !== "open-replace" || args[0] === file)) {
          triggered = true;
          mutate();
          if (attack === "short-read") return 0;
        }
        return result;
      });
      assert.throws(() => consumer.captureInputs(root, [relative]), /changed|truncated/);
      assert.equal(triggered, true);
      if (attack === "replace" || attack === "open-replace")
        assert.equal(fs.existsSync(file + ".original"), true);
    });
  }
});

test("public archive contract accepts only exact registry-backed review leaves", (t) => {
  const refs = consumer.PUBLIC_REVIEW_ARCHIVE_PATHS;
  assert.equal(refs.length, 8);
  const config = { schemaVersion: 1, sourceOwner: "pantheon-base",
    reviewLegacyRegistry: { path: "config/review-loop-legacy-artifacts.json", artifactRoot: ".harness/archive" } };
  const registry = { schemaVersion: 1, entries: [...refs] };
  const row = (file, value) => ({ path: file, bytes: Buffer.from(JSON.stringify(value)) });
  const entries = [row("config/foundation-consumer-artifacts.json", config),
    row("config/review-loop-legacy-artifacts.json", registry),
    ...refs.map(file => ({ path: file, bytes: Buffer.from("Historical review fixture.\n") }))];
  assert.doesNotThrow(() => consumer.assertPublicReviewPackaging(entries));
  for (const file of refs) assert.equal(consumer.assertCandidatePath(file), file);
  for (const invalid of [".harness", ".harness/archive", ".harness/archive/extra/review.md",
    refs[0].toUpperCase(), refs[0].replace("/review.md", "/../review.md"),
    refs[0].replace("/review.md", "/credentials.json"), ".harness/evidence/current/review.md",
    ".harness/archive/.aws/credentials", ".harness/archive/cache/review.md"]) {
    assert.throws(() => consumer.assertCandidatePath(invalid));
  }
  for (const missing of [0, 1, 2])
    assert.throws(() => consumer.assertPublicReviewPackaging(entries.filter((_, index) => index !== missing)));
  for (const badRefs of [[...refs, refs[0]], [...refs.slice(1)], [refs[0].toUpperCase()],
    [".harness/archive/extra/review.md"], [".harness/archive/../secret/review.md"]]) {
    const changed = [...entries];
    changed[1] = row("config/review-loop-legacy-artifacts.json", { ...registry, entries: badRefs });
    assert.throws(() => consumer.assertPublicReviewPackaging(changed));
  }
  assert.throws(() => consumer.assertPublicReviewPackaging([...entries,
    { path: ".harness/evidence/private.json", bytes: Buffer.from("{}") }]));
  const changed = [...entries];
  changed[0] = row("config/foundation-consumer-artifacts.json", {
    ...config, reviewLegacyRegistry: { ...config.reviewLegacyRegistry, artifactRoot: ".harness" },
  });
  assert.throws(() => consumer.assertPublicReviewPackaging(changed), /contract/);
});

test("retained archive fixtures survive assembly and both capture paths", (t) => {
  const f = fixture(t);
  const opsRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const evidence = process.env.O4_REAL_RETAINED_ARCHIVE_DIAGNOSTICS;
  const offenders = evidence ? JSON.parse(fs.readFileSync(evidence)).offenders : [
    { path: ".foundation/releases/pantheon-base-v0.17.6/repo.tar", bytes: 45168640 },
    { path: ".foundation/releases/pantheon-base-v0.18.6/repo.tar", bytes: 59422720 },
    { path: ".foundation/releases/pantheon-base-v0.18.6/foundation-release-pantheon-base-v0.18.6.tgz", bytes: 36214646 },
  ];
  assert.equal(offenders.length, evidence ? 9 : 3);
  for (const offender of offenders) {
    consumer.assertRelativeInput(offender.path);
    assert.ok(offender.path.startsWith(".foundation/releases/"));
    const target = path.join(f.ops, offender.path);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    // Real retained metadata in a disposable fixture; the product manifest and
    // caches stay untouched. The existing assembler owns all subsequent copies.
    if (evidence) fs.copyFileSync(path.join(opsRoot, offender.path), target);
    else {
      fs.writeFileSync(target, "");
      fs.truncateSync(target, offender.bytes);
    }
    assert.throws(() => consumer.readStableFile(target), /bounded/);
  }
  const inputs = consumer.captureInputs(f.ops, [".foundation"]);
  const result = rebuildFromCandidate({ opsRoot: f.ops, ...f.candidate,
    expectedDescriptorSha256: f.candidate.descriptorSha256 });
  const assembled = consumer.captureInputs(result.targetRoot, [".foundation"]);
  assert.deepEqual(assembled.files, inputs.files);
  for (const offender of offenders) {
    const source = result.receipt.opsSourceFilesBefore.find(file => file.path === offender.path);
    const target = result.receipt.assembledSourceFiles.find(file => file.path === offender.path);
    assert.equal(source.bytes, offender.bytes);
    assert.deepEqual(target, source);
  }
  assert.equal(result.receipt.officialLockSha256Before, result.receipt.officialLockSha256After);
  assert.equal(result.receipt.sourceStable, true);
  assert.equal(result.receipt.checks.find(check => check.operation === "checkSharedDrift").exitCode, 0);
});

// Only fixture construction imports the explicitly located Base source tooling.
// The production consumer is self-contained and reads pinned artifact bytes.
const write = (root, relative, bytes) => {
  const absolute = path.join(root, relative);
  fs.mkdirSync(path.dirname(absolute), { recursive: true });
  fs.writeFileSync(absolute, bytes);
};

function fixture(t) {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), "ops-candidate-"));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const base = path.join(temp, "base");
  const ops = path.join(temp, "ops");
  const baseFiles = {
    "package.json": '{"scripts":{}}\n',
    "frontend/package.json": '{"scripts":{}}\n',
    "frontend/tests/smoke/README.md": "# Smoke\n",
    "frontend/scripts/cleanup-generated-modules.mjs":
      "const I18N_LOCALES = [];\n",
    "backend/go.mod": "module pantheon-base\n\ngo 1.24.0\n",
    "backend/go.sum": "",
    "backend/pkg/contracts/module.go":
      "package contracts\nfunc RegisterRequiredBackendModules() {}\n",
    "backend/pkg/database/migrations/000031_base.up.sql": "SELECT 31;\n",
    "backend/modules/business/business.go":
      "package business\nfunc InitBusinessModules(r any, db any) {\n\tInitGeneratedBusinessModules(r, db)\n}\n",
    "backend/modules/business/generated_registry.go":
      "package business\nfunc InitGeneratedBusinessModules(r any, db any) {}\n",
    "backend/modules/system/iam/menu/component_registry.go":
      "package menu\nvar registeredMenuComponentKeys = mergeMenuComponentKeys(staticRegisteredMenuComponentKeys, generatedMenuComponentKeys)\n",
    "frontend/src/core/router/modules.ts":
      "import { generatedBusinessModules } from '../../modules/generated/business';\nexport const businessModules: ModuleConfig[] = [...generatedBusinessModules];\n",
    "frontend/src/core/router/componentRegistry.ts":
      "import { generatedComponentRegistry } from './generatedComponentRegistry';\nconst componentRegistry = {\n  ...staticComponentRegistry,\n  ...generatedComponentRegistry,\n};\n",
    "frontend/src/i18n/resources/zh-CN.ts":
      "const zhCNFallback = {};\nexport default zhCNFallback;\n",
    "frontend/src/i18n/resources/en-US.ts":
      "const enUSFallback = {};\nexport default enUSFallback;\n",
    "docs/base.md": "Candidate fixture only.\n",
  };
  for (const [file, bytes] of Object.entries(baseFiles))
    write(base, file, bytes);
  const git = (root, ...args) => {
    const result = spawnSync("git", args, { cwd: root, encoding: "utf8" });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout.trim();
  };
  git(base, "init");
  git(base, "-c", "core.autocrlf=false", "add", ".");
  git(
    base,
    "-c",
    "user.name=Fixture",
    "-c",
    "user.email=fixture@example.invalid",
    "commit",
    "-m",
    "provenance",
  );
  // A dirty source proves the consumer uses archived bytes, not provenance HEAD.
  write(base, "docs/base.md", "Reviewed dirty candidate fixture.\n");
  const sharedPaths = {
    backend: ["backend/pkg", "backend/modules/business/business.go"],
    frontend: ["frontend/src/core/router/modules.ts"],
    docs: ["docs"],
  };
  const candidate = buildCandidateArtifact({
    root: base,
    baseCommit: git(base, "rev-parse", "HEAD"),
    sourcePaths: ["backend", "frontend", "docs", "package.json"],
    sharedPaths,
    evidenceIds: ["fixture-source-evidence"],
    reviewIds: ["fixture-independent-review"],
    capabilities: ["required-business-bootstrap-v1"],
    outputRoot: path.join(temp, "artifact"),
  });
  const manifest = {
    schemaVersion: 1,
    base: { module: "pantheon-base" },
    sourceModule: "pantheon-ops/backend",
    businessPaths: [
      "backend/modules/business/cmdb",
      "frontend/src/modules/business/cmdb",
    ],
    repositoryOverlayPaths: ["foundation-release.lock.json", ".foundation"],
    businessMigrations: [
      "backend/modules/business/cmdb/migrations/000032_fixture.up.sql",
      "backend/modules/business/cmdb/migrations/000032_fixture.down.sql",
    ],
    backendModules: [
      {
        alias: "cmdb",
        importPath: "pantheon-base/modules/business/cmdb",
        init: "Init",
      },
    ],
    frontendModules: [{ export: "CmdbModule", importPath: "./business/cmdb" }],
    components: [
      {
        key: "business/cmdb/CmdbList",
        importPath: "../../modules/business/cmdb/CmdbList",
      },
    ],
    businessSmokeScripts: [],
    goDependencies: {},
  };
  for (const [file, bytes] of Object.entries({
    "business-overlay.json": JSON.stringify(manifest),
    "foundation-release.lock.json":
      '{ "releaseVersion": "pantheon-base-v0.18.6", "fixture": true }\r\n',
    ".foundation/allowed-drift.json": JSON.stringify({
      "backend/modules/business/business.go": "fixture overlay hook",
      "frontend/src/core/router/modules.ts": "fixture overlay hook",
    }),
    "package.json": '{"scripts":{}}',
    "frontend/package.json": '{"scripts":{}}',
    "backend/modules/business/cmdb/module.go":
      'package cmdb\nimport "pantheon-ops/backend/pkg/contracts"\n',
    "backend/modules/business/cmdb/migrations/000032_fixture.up.sql":
      "SELECT 32;\n",
    "backend/modules/business/cmdb/migrations/000032_fixture.down.sql":
      "SELECT 1;\n",
    "frontend/src/modules/business/cmdb/index.ts":
      "export const CmdbModule = {};\n",
  }))
    write(ops, file, bytes);
  git(ops, "init");
  git(ops, "-c", "core.autocrlf=false", "add", ".");
  const cacheRoot = path.join(
    ops,
    ".tmp/foundation-candidates",
    candidate.descriptor.candidateIdentity,
  );
  const options = {
    opsRoot: ops,
    artifactPath: candidate.artifactPath,
    descriptorPath: candidate.descriptorPath,
    expectedDescriptorSha256: candidate.descriptorSha256,
  };
  return {
    temp,
    base,
    ops,
    candidate,
    manifest,
    cacheRoot,
    options,
    resolve: () => consumer.resolveCandidateArtifact({ ...options, cacheRoot }),
    rebuild: (extra = {}) => rebuildFromCandidate({ ...options, ...extra }),
  };
}

function repin(f, descriptor) {
  fs.writeFileSync(f.candidate.descriptorPath, JSON.stringify(descriptor));
  f.options.expectedDescriptorSha256 = consumer.sha256(
    fs.readFileSync(f.candidate.descriptorPath),
  );
}

function generatedFixture(t, sharedPaths = { backend: ["backend/pkg", "backend/modules"], frontend: ["frontend/src"] }) {
  const f = fixture(t);
  for (const locale of ["en-US", "zh-CN", "fr-FR", "ja-JP", "ko-KR"])
    write(f.ops, `frontend/src/modules/business/cmdb/locales/${locale}.json`, JSON.stringify({ "business.cmdb.title": locale }));
  f.manifest.businessSmokeScripts = ["test:smoke:business:cmdb"];
  write(f.ops, "business-overlay.json", JSON.stringify(f.manifest));
  write(f.ops, "frontend/package.json", JSON.stringify({ scripts: {
    "test:smoke:business:cmdb": "playwright test tests/smoke/business/cmdb/pages.spec.ts",
  } }));
  write(f.ops, ".foundation/allowed-drift.json", fs.readFileSync(".foundation/allowed-drift.json"));
  const candidate = buildCandidateArtifact({ root: f.base, baseCommit: f.candidate.descriptor.baseCommit,
    sourcePaths: ["backend", "frontend", "docs", "package.json"], sharedPaths,
    evidenceIds: ["fixture-source-evidence"], reviewIds: ["fixture-independent-review"],
    capabilities: ["required-business-bootstrap-v1"], outputRoot: path.join(f.temp, "wide-artifact") });
  f.options.artifactPath = candidate.artifactPath;
  f.options.descriptorPath = candidate.descriptorPath;
  f.options.expectedDescriptorSha256 = candidate.descriptorSha256;
  f.cacheRoot = path.join(f.ops, ".tmp/foundation-candidates", candidate.descriptor.candidateIdentity);
  return f;
}

test("real generated i18n, import and registry output passes without broadening legacy allowances", (t) => {
  const f = generatedFixture(t);
  const result = f.rebuild();
  assert.equal(result.receipt.sourceStable, true);
  assert.equal(result.receipt.officialConsumption, false);
  for (const locale of ["en-US", "zh-CN", "fr-FR", "ja-JP", "ko-KR"])
    assert.match(fs.readFileSync(path.join(result.targetRoot, `frontend/src/i18n/resources/business/${locale}.ts`), "utf8"), /business.cmdb.title/);
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(result.targetRoot, ".foundation/allowed-drift.json"))), JSON.parse(fs.readFileSync(".foundation/allowed-drift.json")));
});

test("official v0.18.6 frontend coverage retains package, cleanup and smoke contract compatibility", (t) => {
  const f = generatedFixture(t, { backend: ["backend/pkg", "backend/modules"], frontend: [
    "frontend/src", "frontend/package.json", "frontend/scripts/cleanup-generated-modules.mjs", "frontend/tests/smoke/README.md",
  ] });
  const result = f.rebuild();
  assert.equal(result.receipt.sourceStable, true);
  assert.ok(result.receipt.officialLockSha256Before === result.receipt.officialLockSha256After);
});

test("canonical assembly does not hide a genuinely stale in-coverage legacy allowance", (t) => {
  const f = generatedFixture(t, { backend: ["backend/pkg", "backend/modules"], frontend: [
    "frontend/src", "frontend/package.json", "frontend/scripts/cleanup-generated-modules.mjs", "frontend/tests/smoke/README.md",
  ] });
  f.manifest.businessSmokeScripts = [];
  write(f.ops, "business-overlay.json", JSON.stringify(f.manifest));
  assert.throws(f.rebuild, /Candidate shared drift check failed/);
  assert.equal(fs.existsSync(f.cacheRoot), false);
});

test("canonical generated bytes cannot be forged by editing the report or legacy allowlist", async (t) => {
  for (const file of ["frontend/src/i18n/resources/en-US.ts", "frontend/src/i18n/resources/business/fr-FR.ts", "frontend/src/core/router/modules.ts", "frontend/src/modules/businessOverlay.ts", "backend/modules/business/business.go", "frontend/package.json", "frontend/scripts/cleanup-generated-modules.mjs", "frontend/tests/smoke/README.md"])
    await t.test(file, (st) => {
      const f = generatedFixture(st);
      const original = fs.writeFileSync;
      st.mock.method(fs, "writeFileSync", function (target, ...args) {
        const result = original.call(this, target, ...args);
        if (String(target).endsWith("assembled/.business-overlay-report.json".replaceAll("/", path.sep))) {
          const changed = path.join(f.cacheRoot, "assembled", file);
          fs.appendFileSync(changed, "\n// shared byte tamper\n");
          const report = JSON.parse(fs.readFileSync(target));
          report.ownedFiles.find(entry => entry.path === file).sha256 = consumer.sha256(fs.readFileSync(changed));
          original.call(this, target, JSON.stringify(report));
        }
        return result;
      });
      assert.throws(f.rebuild, /Derived business output differs|Canonical business output/);
      assert.equal(fs.existsSync(f.cacheRoot), false, "proven owned assembly was cleaned");
    });
});

test("forged generatedFiles cannot exempt actual shared Base bytes", (t) => {
  const f = generatedFixture(t);
  const original = fs.writeFileSync;
  t.mock.method(fs, "writeFileSync", function (target, ...args) {
    const result = original.call(this, target, ...args);
    if (String(target).endsWith(path.join("assembled", ".business-overlay-report.json"))) {
      const file = "backend/pkg/contracts/module.go";
      const absolute = path.join(f.cacheRoot, "assembled", file);
      fs.appendFileSync(absolute, "\n// shared byte tamper\n");
      const report = JSON.parse(fs.readFileSync(target));
      report.generatedFiles.push(file);
      report.ownedFiles.push({ path: file, sha256: consumer.sha256(fs.readFileSync(absolute)) });
      original.call(this, target, JSON.stringify(report));
    }
    return result;
  });
  assert.throws(f.rebuild, /Candidate shared drift check failed/);
  assert.equal(fs.existsSync(f.cacheRoot), false);
});

test("fresh consumer attempt reuses verified immutable snapshot without touching retained failure", (t) => {
  const f = generatedFixture(t);
  const first = f.rebuild();
  const retained = consumer.captureInputs(first.targetRoot, fs.readdirSync(first.targetRoot).sort());
  assert.throws(f.rebuild, /stage already exists/);
  const second = f.rebuild({ attempt: "fixture-retry" });
  assert.notEqual(second.targetRoot, first.targetRoot);
  assert.equal(second.snapshot.baseRoot, first.snapshot.baseRoot);
  assert.deepEqual(consumer.captureInputs(first.targetRoot, fs.readdirSync(first.targetRoot).sort()), retained);
  assert.throws(() => f.rebuild({ attempt: "fixture-retry" }), /already exists/);
  assert.throws(() => f.rebuild({ attempt: "../escape" }), /attempt/);
  consumer.removeCandidateStage(second.snapshot.cacheRoot);
  assert.equal(fs.existsSync(first.snapshot.baseRoot), true);
  assert.deepEqual(consumer.captureInputs(first.targetRoot, fs.readdirSync(first.targetRoot).sort()), retained);
});

test("fresh consumer attempts reject tampered retained snapshot and preserve the original", (t) => {
  const f = generatedFixture(t);
  const first = f.rebuild();
  const file = path.join(first.snapshot.baseRoot, "docs/base.md");
  fs.appendFileSync(file, "tamper");
  assert.throws(() => f.rebuild({ attempt: "rejected-snapshot" }), /inventory mismatch/);
  assert.match(fs.readFileSync(file, "utf8"), /tamper$/);
  assert.equal(fs.existsSync(first.targetRoot), true);
  assert.equal(fs.existsSync(path.join(f.ops, '.tmp/foundation-candidate-attempts', first.receipt.candidateIdentity, 'rejected-snapshot')), false);
});

function checksum(archive) {
  archive.fill(32, 148, 156);
  const sum = [...archive.subarray(0, 512)].reduce(
    (total, byte) => total + byte,
    0,
  );
  archive.write(`${sum.toString(8).padStart(6, "0")}\0 `, 148, 8, "ascii");
}

function interceptCopy(action, run) {
  const copy = fs.copyFileSync;
  let first = true;
  fs.copyFileSync = (...args) => {
    const result = copy(...args);
    if (first) {
      first = false;
      action(...args);
    }
    return result;
  };
  try {
    return run();
  } finally {
    fs.copyFileSync = copy;
  }
}

function assertCleanupRefused(run, causeMessage) {
  assert.throws(run, (error) => {
    assert.equal(error.name, "CandidateCleanupError");
    assert.equal(error.code, "ERR_CANDIDATE_CLEANUP_OWNERSHIP");
    assert.match(error.message, /cleanup.*(identity|ownership)/i);
    if (causeMessage) assert.match(error.cause?.message ?? "", causeMessage);
    return true;
  });
}

for (const phase of ["extraction", "assembly"]) {
  for (const directory of [
    "stage",
    "snapshot",
    "nested",
    "ancestor",
    ...(phase === "assembly" ? ["assembled", "assembled-nested"] : []),
  ]) {
    test(`cleanup preserves renamed original and real replacement: ${phase}/${directory}`, (t) => {
      const f = fixture(t);
      let attacked;
      const retained = path.join(f.temp, "retained-original");
      const replace = () => {
        attacked =
          directory === "stage"
            ? f.cacheRoot
            : directory === "ancestor"
              ? path.dirname(f.cacheRoot)
              : directory === "nested"
                ? path.join(f.cacheRoot, "snapshot/backend")
                : directory === "assembled-nested"
                  ? path.join(f.cacheRoot, "assembled/backend")
                : path.join(f.cacheRoot, directory);
        fs.renameSync(attacked, retained);
        fs.mkdirSync(attacked);
        write(attacked, "foreign.txt", "preserve foreign replacement");
        throw new Error("injected directory replacement failure");
      };
      if (phase === "assembly") {
        assertCleanupRefused(
          () => interceptCopy(replace, f.rebuild),
          /injected directory replacement failure/,
        );
      } else {
        const originalWrite = fs.writeFileSync;
        let fired = false;
        fs.writeFileSync = (file, ...args) => {
          const result = originalWrite(file, ...args);
          if (
            !fired &&
            String(file).includes(`${path.sep}snapshot${path.sep}`)
          ) {
            fired = true;
            replace();
          }
          return result;
        };
        try {
          assertCleanupRefused(
            f.resolve,
            /injected directory replacement failure/,
          );
        } finally {
          fs.writeFileSync = originalWrite;
        }
      }
      assert.ok(attacked, "the real failure hook ran");
      assert.equal(
        fs.readFileSync(path.join(attacked, "foreign.txt"), "utf8"),
        "preserve foreign replacement",
      );
      assert.equal(fs.lstatSync(retained).isDirectory(), true);
      const retainedFile =
        directory === "stage"
          ? "snapshot/backend/go.mod"
          : directory === "ancestor"
            ? `${path.basename(f.cacheRoot)}/snapshot/backend/go.mod`
            : directory === "snapshot"
              ? "backend/go.mod"
              : directory === "nested" || directory === "assembled-nested"
                ? "go.mod"
                : "backend/go.mod";
      assert.equal(
        fs.readFileSync(path.join(retained, retainedFile), "utf8"),
        "module pantheon-base\n\ngo 1.24.0\n",
      );
    });
  }
}

test("cleanup refuses an unknown descendant before removing any owned files", (t) => {
  const f = fixture(t);
  const snapshot = f.resolve();
  write(f.cacheRoot, "unknown/foreign.txt", "preserve");
  assertCleanupRefused(() => consumer.removeCandidateStage(f.cacheRoot));
  assert.equal(
    fs.readFileSync(path.join(f.cacheRoot, "unknown/foreign.txt"), "utf8"),
    "preserve",
  );
  assert.equal(
    fs.existsSync(path.join(snapshot.baseRoot, "backend/go.mod")),
    true,
  );
});

test("cleanup removes an unchanged extracted directory instance", (t) => {
  const f = fixture(t);
  f.resolve();
  consumer.removeCandidateStage(f.cacheRoot);
  assert.equal(fs.existsSync(f.cacheRoot), false);
});

for (const identity of ["missing", "zero"]) {
  test(`cleanup fails closed when created directory identity is ${identity}`, (t) => {
    const f = fixture(t);
    const originalLstat = fs.lstatSync;
    fs.lstatSync = (file, options) => {
      const stat = originalLstat(file, options);
      if (file === f.cacheRoot && options?.bigint) {
        stat.ino = identity === "missing" ? undefined : 0n;
      }
      return stat;
    };
    try {
      assertCleanupRefused(f.resolve, /Unavailable real directory identity/);
    } finally {
      fs.lstatSync = originalLstat;
    }
    assert.equal(fs.lstatSync(f.cacheRoot).isDirectory(), true);
    assertCleanupRefused(() => consumer.removeCandidateStage(f.cacheRoot));
  });
}

test("source verification refuses an unchanged snapshot moved into a replacement stage", (t) => {
  const f = fixture(t);
  const snapshot = f.resolve();
  const retained = path.join(f.temp, "retained-stage");
  fs.renameSync(f.cacheRoot, retained);
  fs.mkdirSync(f.cacheRoot);
  fs.renameSync(path.join(retained, "snapshot"), snapshot.baseRoot);
  assert.throws(snapshot.verify, /Replaced directory identity/);
  assertCleanupRefused(() => consumer.removeCandidateStage(f.cacheRoot));
  assert.equal(
    fs.existsSync(path.join(snapshot.baseRoot, "backend/go.mod")),
    true,
  );
  assert.equal(fs.lstatSync(retained).isDirectory(), true);
});

test("cleanup refuses a replaced ancestor observed during individual file deletion", (t) => {
  const f = fixture(t);
  f.resolve();
  const retained = path.join(f.temp, "retained-stage");
  const originalUnlink = fs.unlinkSync;
  let fired = false;
  fs.unlinkSync = (file) => {
    const result = originalUnlink(file);
    if (!fired) {
      fired = true;
      fs.renameSync(f.cacheRoot, retained);
      fs.mkdirSync(f.cacheRoot);
      write(f.cacheRoot, "foreign.txt", "preserve");
    }
    return result;
  };
  try {
    assertCleanupRefused(() => consumer.removeCandidateStage(f.cacheRoot));
  } finally {
    fs.unlinkSync = originalUnlink;
  }
  assert.equal(fired, true);
  assert.equal(
    fs.readFileSync(path.join(f.cacheRoot, "foreign.txt"), "utf8"),
    "preserve",
  );
  // One owned file was already deleted: this test makes no atomicity claim.
  assert.equal(
    fs.existsSync(path.join(retained, "snapshot/backend/go.sum")),
    true,
  );
});

test("schema1 source-interface mapping matches the producer and rejects contradictory metadata", (t) => {
  const f = fixture(t);
  assert.deepEqual(consumer.CAPABILITY_EXPORTS, producer.CAPABILITY_EXPORTS);
  assert.deepEqual(consumer.LIMITS, producer.LIMITS);
  assert.deepEqual(
    consumer.readCandidateDescriptor(
      f.options.descriptorPath,
      f.options.expectedDescriptorSha256,
    ),
    f.candidate.descriptor,
  );
  const deltas = [
    { schemaVersion: 2 },
    { kind: "foundation-release" },
    { consumerMode: "official" },
    { sourceRepo: "pantheon-ops" },
    { baseCommit: "fake" },
    { releaseVersion: "pantheon-base-v0.18.6" },
    { aggregateSha256: "0".repeat(64) },
    { candidateIdentity: "candidate-false" },
    { artifact: { assetName: "official.tar", sha256: "0".repeat(64) } },
    { evidenceIds: [] },
    { reviewIds: [] },
    { capabilities: ["false"] },
    { requiredExports: [] },
    { sharedPaths: {} },
    { sharedPaths: { backend: ["missing"] } },
  ];
  for (const delta of deltas) {
    const descriptor = { ...f.candidate.descriptor, ...delta };
    assert.throws(() => consumer.validateCandidateDescriptor(descriptor));
    assert.throws(() => producer.validateCandidateDescriptor(descriptor));
  }
});

test("requires external descriptor pin and rejects descriptor/archive tamper before staging", (t) => {
  const f = fixture(t);
  assert.throws(
    () =>
      consumer.resolveCandidateArtifact({
        ...f.options,
        cacheRoot: f.cacheRoot,
        expectedDescriptorSha256: undefined,
      }),
    /digest/,
  );
  fs.appendFileSync(f.candidate.descriptorPath, " ");
  assert.throws(f.resolve, /digest/);
  repin(f, f.candidate.descriptor);
  const bytes = fs.readFileSync(f.candidate.artifactPath);
  bytes[512] ^= 1;
  fs.writeFileSync(f.candidate.artifactPath, bytes);
  assert.throws(f.resolve, /digest/);
  assert.equal(fs.existsSync(f.cacheRoot), false);
});

test("rejects descriptor edit/revert during extraction and bounded input overflow", (t) => {
  const f = fixture(t);
  const originalWrite = fs.writeFileSync;
  const originalDescriptor = fs.readFileSync(f.candidate.descriptorPath);
  let changed = false;
  fs.writeFileSync = (file, ...args) => {
    const result = originalWrite(file, ...args);
    if (!changed && String(file).includes(`${path.sep}snapshot${path.sep}`)) {
      changed = true;
      originalWrite(f.candidate.descriptorPath, "transient");
      originalWrite(f.candidate.descriptorPath, originalDescriptor);
      fs.utimesSync(
        f.candidate.descriptorPath,
        new Date(),
        new Date(Date.now() + 2000),
      );
    }
    return result;
  };
  try {
    assert.throws(f.resolve, /inputs changed/);
  } finally {
    fs.writeFileSync = originalWrite;
  }
  assert.equal(fs.existsSync(f.cacheRoot), false);
  assert.throws(
    () => consumer.readStableFile(f.candidate.artifactPath, 10),
    /bounded/,
  );
  assert.throws(
    () =>
      consumer.assertInventory([
        {
          path: "large",
          bytes: consumer.LIMITS.fileBytes + 1,
          sha256: consumer.sha256(""),
        },
      ]),
    /metadata/,
  );
  assert.throws(
    () => consumer.CAPABILITY_EXPORTS["required-business-bootstrap-v1"].pop(),
    TypeError,
  );
});

test("rejects hostile regular tar paths, aliases, config and extension/link/special types with valid archive pins", async (t) => {
  const paths = [
    "../escape",
    "/absolute",
    "C:/outside",
    "//host/share",
    "dir\\file",
    "x:y",
    ".env.local",
    "config/secrets.json",
    ".aws/credentials",
    "aux.txt",
    "file.",
    "a//b",
  ];
  for (const name of paths)
    await t.test(name, (st) => {
      const f = fixture(st);
      const archive = fs.readFileSync(f.candidate.artifactPath);
      archive.fill(0, 0, 100);
      archive.write(name, 0, 100);
      checksum(archive);
      fs.writeFileSync(f.candidate.artifactPath, archive);
      repin(f, {
        ...f.candidate.descriptor,
        artifact: {
          ...f.candidate.descriptor.artifact,
          sha256: consumer.sha256(archive),
        },
      });
      assert.throws(f.resolve, /path|Excluded/);
      assert.equal(fs.existsSync(f.cacheRoot), false);
    });
  for (const type of ["1", "2", "3", "5", "x", "g", "L", "K"])
    await t.test(`tar type ${type}`, (st) => {
      const f = fixture(st);
      const archive = fs.readFileSync(f.candidate.artifactPath);
      archive[156] = type.charCodeAt(0);
      checksum(archive);
      fs.writeFileSync(f.candidate.artifactPath, archive);
      repin(f, {
        ...f.candidate.descriptor,
        artifact: {
          ...f.candidate.descriptor.artifact,
          sha256: consumer.sha256(archive),
        },
      });
      assert.throws(f.resolve, /header|type|link/);
      assert.equal(fs.existsSync(f.cacheRoot), false);
    });
});

test("rejects duplicate/case/directory collisions, missing entries, wrong file hashes and corrupt tar framing", (t) => {
  const f = fixture(t);
  const row = (name) => ({ path: name, bytes: 0, sha256: consumer.sha256("") });
  for (const names of [
    ["a", "a"],
    ["A", "a"],
    ["Foo/a", "foo/b"],
    ["a", "a/b"],
  ])
    assert.throws(() => consumer.assertInventory(names.map(row)));
  const archive = fs.readFileSync(f.candidate.artifactPath);
  const firstSize = f.candidate.descriptor.files[0].bytes;
  const secondOffset = 512 + firstSize + ((512 - (firstSize % 512)) % 512);
  for (const name of [
    f.candidate.descriptor.files[0].path,
    f.candidate.descriptor.files[0].path.toUpperCase(),
  ]) {
    const bytes = Buffer.from(archive);
    const header = bytes.subarray(secondOffset, secondOffset + 512);
    header.fill(0, 0, 100);
    header.write(name, 0, 100);
    checksum(header);
    assert.throws(
      () =>
        consumer.verifyCandidateArchive(bytes, {
          ...f.candidate.descriptor,
          artifact: {
            ...f.candidate.descriptor.artifact,
            sha256: consumer.sha256(bytes),
          },
        }),
      /unique|collision|sorted/,
    );
  }
  const firstOnly = Buffer.concat([
    archive.subarray(0, secondOffset),
    Buffer.alloc(1024),
  ]);
  assert.throws(
    () =>
      consumer.verifyCandidateArchive(firstOnly, {
        ...f.candidate.descriptor,
        artifact: {
          ...f.candidate.descriptor.artifact,
          sha256: consumer.sha256(firstOnly),
        },
      }),
    /inventory/,
  );
  const files = f.candidate.descriptor.files.map((file, index) =>
    index ? file : { ...file, sha256: "0".repeat(64) },
  );
  const aggregateSha256 = consumer.aggregateSha256(files);
  assert.throws(
    () =>
      consumer.verifyCandidateArchive(archive, {
        ...f.candidate.descriptor,
        files,
        aggregateSha256,
        candidateIdentity: `candidate-${aggregateSha256}`,
        artifact: {
          ...f.candidate.descriptor.artifact,
          assetName: `candidate-${aggregateSha256}.tar`,
        },
      }),
    /inventory/,
  );
  for (const mutate of [
    (bytes) => {
      bytes[148] = 57;
    },
    (bytes) => {
      bytes[257] = 0;
      checksum(bytes);
    },
    (bytes) => {
      bytes[bytes.length - 1] = 1;
    },
  ]) {
    const bytes = Buffer.from(archive);
    mutate(bytes);
    assert.throws(() =>
      consumer.verifyCandidateArchive(bytes, {
        ...f.candidate.descriptor,
        artifact: {
          ...f.candidate.descriptor.artifact,
          sha256: consumer.sha256(bytes),
        },
      }),
    );
  }
  const shortened = archive.subarray(0, archive.length - 1024);
  assert.throws(
    () =>
      consumer.verifyCandidateArchive(shortened, {
        ...f.candidate.descriptor,
        artifact: {
          ...f.candidate.descriptor.artifact,
          sha256: consumer.sha256(shortened),
        },
      }),
    /trailer/,
  );
});

test("false capability comments cannot replace real declarations even when all digests are consistent", (t) => {
  const f = fixture(t);
  write(
    f.base,
    "backend/pkg/contracts/module.go",
    "package contracts\n// func RegisterRequiredBackendModules() {}\n",
  );
  const entries = producer
    .verifyCandidateArchive(
      fs.readFileSync(f.candidate.artifactPath),
      f.candidate.descriptor,
    )
    .map((entry) =>
      entry.path === "backend/pkg/contracts/module.go"
        ? {
            path: entry.path,
            bytes: Buffer.from(
              "package contracts\n// func RegisterRequiredBackendModules() {}\n",
            ),
          }
        : entry,
    );
  const files = entries.map((entry) => ({
    path: entry.path,
    bytes: entry.bytes.length,
    sha256: consumer.sha256(entry.bytes),
  }));
  const aggregateSha256 = consumer.aggregateSha256(files);
  const identity = `candidate-${aggregateSha256}`;
  const archive = producer.encodeCandidateArchive(entries);
  fs.writeFileSync(f.candidate.artifactPath, archive);
  repin(f, {
    ...f.candidate.descriptor,
    files,
    aggregateSha256,
    candidateIdentity: identity,
    requiredExports: consumer.candidateExports(
      f.candidate.descriptor.capabilities,
      files,
    ),
    artifact: {
      assetName: `${identity}.tar`,
      sha256: consumer.sha256(archive),
    },
  });
  assert.throws(
    () =>
      consumer.resolveCandidateArtifact({
        ...f.options,
        cacheRoot: path.join(f.ops, ".tmp/foundation-candidates", identity),
      }),
    /API export/,
  );
});

test("fresh extraction is sealed; mutations, additions, edit/revert and snapshot reuse refuse", async (t) => {
  for (const mode of ["write", "addition", "revert", "root-revert"])
    await t.test(mode, (st) => {
      const f = fixture(st);
      const snapshot = f.resolve();
      assert.equal(snapshot.verify(), true);
      assert.equal(
        fs.readFileSync(path.join(snapshot.baseRoot, "docs/base.md"), "utf8"),
        "Reviewed dirty candidate fixture.\n",
      );
      assert.throws(f.resolve, /already exists/);
      const file = path.join(snapshot.baseRoot, "docs/base.md");
      if (mode === "write") fs.appendFileSync(file, "changed");
      if (mode === "addition")
        write(snapshot.baseRoot, "docs/new.md", "addition");
      if (mode === "revert") {
        const bytes = fs.readFileSync(file);
        fs.writeFileSync(file, "transient");
        fs.writeFileSync(file, bytes);
        fs.utimesSync(file, new Date(), new Date(Date.now() + 2000));
      }
      if (mode === "root-revert") {
        write(snapshot.baseRoot, "temporary", "x");
        fs.unlinkSync(path.join(snapshot.baseRoot, "temporary"));
        fs.utimesSync(
          snapshot.baseRoot,
          new Date(),
          new Date(Date.now() + 2000),
        );
      }
      assert.throws(snapshot.verify, /snapshot changed/);
    });
});

test("rebuild calls real assembly/drift seams, derives only declared migrations, preserves exact locks and never reads current Base sources", (t) => {
  const f = fixture(t);
  const lock = fs.readFileSync(
    path.join(f.ops, "foundation-release.lock.json"),
  );
  const realLockPath = fileURLToPath(
    new URL("../../foundation-release.lock.json", import.meta.url),
  );
  const realLock = fs.readFileSync(realLockPath);
  write(
    f.base,
    "docs/base.md",
    "Unreviewed current source must not be copied.\n",
  );
  write(
    f.ops,
    "backend/pkg/database/migrations/999999_undeclared.up.sql",
    "DO NOT COPY;",
  );
  const result = f.rebuild();
  assert.equal(result.targetRoot, path.join(f.cacheRoot, "assembled"));
  assert.equal(result.receipt.sourceStable, true);
  assert.equal(result.receipt.officialConsumption, false);
  assert.equal(result.receipt.consumerMode, "candidate-staging");
  assert.equal(result.receipt.baseCommit, f.candidate.descriptor.baseCommit);
  assert.deepEqual(
    result.receipt.opsSourceFilesBefore,
    result.receipt.opsSourceFilesAfter,
  );
  assert.equal(
    result.receipt.officialLockSha256Before,
    result.receipt.officialLockSha256After,
  );
  assert.equal(
    fs.readFileSync(path.join(result.targetRoot, "docs/base.md"), "utf8"),
    "Reviewed dirty candidate fixture.\n",
  );
  assert.equal(
    fs.existsSync(
      path.join(
        result.targetRoot,
        "backend/pkg/database/migrations/999999_undeclared.up.sql",
      ),
    ),
    false,
  );
  for (const migration of result.receipt.generatedBusinessMigrations)
    assert.deepEqual(
      fs.readFileSync(path.join(result.targetRoot, migration.target)),
      fs.readFileSync(path.join(f.ops, migration.source)),
    );
  assert.deepEqual(
    fs.readFileSync(path.join(f.ops, "foundation-release.lock.json")),
    lock,
  );
  assert.deepEqual(
    fs.readFileSync(
      path.join(result.targetRoot, "foundation-release.lock.json"),
    ),
    lock,
  );
  assert.deepEqual(fs.readFileSync(realLockPath), realLock);
  assert.equal(
    JSON.parse(fs.readFileSync(result.receiptPath, "utf8")).kind,
    "pantheon-ops-candidate-assembly",
  );
  assert.throws(f.rebuild, /already exists/);
});

test("protects repository/foreign/existing targets and rejects alternative manifest ownership", (t) => {
  const f = fixture(t);
  const sentinel = path.join(f.ops, "keep.txt");
  fs.writeFileSync(sentinel, "preserve");
  for (const targetRoot of [
    f.ops,
    path.dirname(f.ops),
    f.base,
    path.join(f.ops, "backend"),
    path.join(f.ops, ".git"),
    path.join(f.temp, "foreign"),
    path.join(f.cacheRoot, "../other/assembled"),
  ])
    assert.throws(() => f.rebuild({ targetRoot }), /target/);
  assert.throws(
    () => f.rebuild({ manifestPath: "alternate.json" }),
    /canonical/,
  );
  write(f.cacheRoot, "assembled/keep.txt", "existing");
  assert.throws(f.rebuild, /already exists/);
  assert.throws(
    () => consumer.removeCandidateStage(f.cacheRoot),
    /cleanup target/,
  );
  assert.equal(
    fs.readFileSync(path.join(f.cacheRoot, "assembled/keep.txt"), "utf8"),
    "existing",
  );
  assert.equal(fs.readFileSync(sentinel, "utf8"), "preserve");
});

test("rejects junction ancestors and hard-linked artifact/declared source without touching foreign bytes", (t) => {
  const f = fixture(t);
  const foreign = path.join(f.temp, "foreign");
  fs.mkdirSync(foreign);
  write(foreign, "keep.txt", "preserve");
  fs.symlinkSync(foreign, path.join(f.ops, ".tmp"), "junction");
  assert.throws(f.rebuild, /real directories/);
  assert.equal(
    fs.readFileSync(path.join(foreign, "keep.txt"), "utf8"),
    "preserve",
  );
  fs.unlinkSync(path.join(f.ops, ".tmp"));
  fs.linkSync(f.candidate.artifactPath, path.join(f.temp, "artifact-link"));
  assert.throws(f.resolve, /hard links/);
  fs.unlinkSync(path.join(f.temp, "artifact-link"));
  fs.linkSync(
    path.join(f.ops, "backend/modules/business/cmdb/module.go"),
    path.join(f.temp, "business-link"),
  );
  assert.throws(f.rebuild, /hard links/);
});

test("assembly rejects snapshot/artifact and declared Ops mutations including edit/revert and cleans proven owned output", async (t) => {
  for (const mode of [
    "snapshot",
    "artifact",
    "business",
    "manifest",
    "lock",
    "revert",
    "addition",
    "git-selection",
  ])
    await t.test(mode, (st) => {
      const f = fixture(st);
      const action = (source) => {
        if (mode === "snapshot") fs.appendFileSync(source, "mutation");
        if (mode === "artifact")
          fs.appendFileSync(f.candidate.artifactPath, "mutation");
        if (mode === "business")
          fs.appendFileSync(
            path.join(f.ops, "backend/modules/business/cmdb/module.go"),
            "mutation",
          );
        if (mode === "manifest")
          fs.appendFileSync(path.join(f.ops, "business-overlay.json"), " ");
        if (mode === "lock")
          fs.appendFileSync(
            path.join(f.ops, "foundation-release.lock.json"),
            " ",
          );
        if (mode === "addition")
          write(f.ops, "backend/modules/business/cmdb/new.go", "package cmdb");
        if (mode === "git-selection") {
          const relative = "backend/modules/business/cmdb/module.go";
          const removed = spawnSync(
            "git",
            ["update-index", "--force-remove", relative],
            { cwd: f.ops, encoding: "utf8" },
          );
          assert.equal(removed.status, 0, removed.stderr);
          write(f.ops, ".git/info/exclude", `${relative}\n`);
        }
        if (mode === "revert") {
          const file = path.join(
            f.ops,
            "backend/modules/business/cmdb/module.go",
          );
          const bytes = fs.readFileSync(file);
          fs.writeFileSync(file, "temporary");
          fs.writeFileSync(file, bytes);
          fs.utimesSync(file, new Date(), new Date(Date.now() + 2000));
        }
      };
      const causeMessage =
        mode === "snapshot"
          ? /Candidate snapshot changed during assembly/
          : mode === "artifact"
            ? /Candidate artifact inputs changed during assembly/
            : /Declared Ops source\/manifest\/lock changed during candidate assembly/;
      assert.throws(() => interceptCopy(action, f.rebuild), causeMessage);
      assert.equal(fs.existsSync(f.cacheRoot), false);
      assert.equal(
        fs.existsSync(
          path.join(
            f.cacheRoot,
            "assembled/.foundation-candidate-receipt.json",
          ),
        ),
        false,
      );
      assert.equal(fs.existsSync(f.ops), true);
    });
});

test("drift and migration conflicts refuse candidate publication", async (t) => {
  await t.test("unallowlisted shared modification", (st) => {
    const f = fixture(st);
    write(f.ops, ".foundation/allowed-drift.json", "{}");
    const copy = fs.copyFileSync;
    st.mock.method(fs, "copyFileSync", function (source, target, ...args) {
      const result = copy.call(this, source, target, ...args);
      if (String(target).endsWith(path.join("assembled", "docs", "base.md"))) fs.appendFileSync(target, "shared tamper");
      return result;
    });
    assert.throws(f.rebuild, /Candidate shared drift check failed/);
    assert.equal(fs.existsSync(f.cacheRoot), false);
    assert.equal(
      fs.existsSync(
        path.join(f.cacheRoot, "assembled/.foundation-candidate-receipt.json"),
      ),
      false,
    );
  });
  await t.test("Base migration conflict", (st) => {
    const f = fixture(st);
    const manifest = {
      ...f.manifest,
      businessMigrations: [
        "backend/modules/business/cmdb/migrations/000031_collision.up.sql",
        "backend/modules/business/cmdb/migrations/000031_collision.down.sql",
      ],
    };
    write(
      f.ops,
      "backend/modules/business/cmdb/migrations/000031_collision.up.sql",
      "SELECT 0;",
    );
    write(
      f.ops,
      "backend/modules/business/cmdb/migrations/000031_collision.down.sql",
      "SELECT 0;",
    );
    write(f.ops, "business-overlay.json", JSON.stringify(manifest));
    assert.throws(
      f.rebuild,
      /Business migration version conflicts with Base: 000031/,
    );
    assert.equal(fs.existsSync(f.cacheRoot), false);
    assert.equal(
      fs.existsSync(
        path.join(f.cacheRoot, "assembled/.foundation-candidate-receipt.json"),
      ),
      false,
    );
  });
});

test("independently pinned build receipt binds source capability identity and exact executable bytes", (t) => {
  const f = fixture(t);
  const executable = Buffer.from("fixture executable, never launched");
  const executablePath = path.join(f.temp, "fixture.exe");
  fs.writeFileSync(executablePath, executable);
  const buildReceiptPath = path.join(f.temp, "build.json");
  const receipt = {
    schemaVersion: 1,
    kind: "pantheon-candidate-build",
    candidateIdentity: f.candidate.descriptor.candidateIdentity,
    descriptorSha256: f.options.expectedDescriptorSha256,
    aggregateSha256: f.candidate.descriptor.aggregateSha256,
    capabilities: f.candidate.descriptor.capabilities,
    executable: {
      bytes: executable.length,
      sha256: consumer.sha256(executable),
    },
    evidenceIds: ["fixture-build"],
    reviewIds: ["fixture-build-review"],
  };
  fs.writeFileSync(buildReceiptPath, JSON.stringify(receipt));
  const options = {
    ...f.options,
    executablePath,
    expectedExecutableSha256: consumer.sha256(executable),
    buildReceiptPath,
    expectedBuildReceiptSha256: consumer.sha256(
      fs.readFileSync(buildReceiptPath),
    ),
  };
  assert.deepEqual(consumer.verifyCandidateExecutable(options), receipt);
  assert.throws(
    () =>
      consumer.verifyCandidateExecutable({
        ...options,
        expectedBuildReceiptSha256: undefined,
      }),
    /Pinned/,
  );
  assert.throws(
    () =>
      consumer.verifyCandidateExecutable({
        ...options,
        expectedExecutableSha256: "0".repeat(64),
      }),
    /executable hash/,
  );
  for (const delta of [
    { candidateIdentity: "candidate-false" },
    { descriptorSha256: "0".repeat(64) },
    { aggregateSha256: "0".repeat(64) },
    { capabilities: ["false"] },
    { releaseVersion: "official" },
  ]) {
    fs.writeFileSync(
      buildReceiptPath,
      JSON.stringify({ ...receipt, ...delta }),
    );
    assert.throws(
      () =>
        consumer.verifyCandidateExecutable({
          ...options,
          expectedBuildReceiptSha256: consumer.sha256(
            fs.readFileSync(buildReceiptPath),
          ),
        }),
      /identity|capability/,
    );
  }
  fs.writeFileSync(buildReceiptPath, JSON.stringify(receipt));
  fs.appendFileSync(executablePath, "mutation");
  assert.throws(
    () => consumer.verifyCandidateExecutable(options),
    /executable hash/,
  );
});

test("CLI help is precise; unsupported, duplicate, unpinned and formal-source options refuse", () => {
  const entry = fileURLToPath(
    new URL("./rebuild-from-candidate.mjs", import.meta.url),
  );
  const help = spawnSync(process.execPath, [entry, "--help"], {
    encoding: "utf8",
  });
  assert.equal(help.status, 0, help.stderr);
  assert.match(help.stdout, /descriptor-sha256/);
  assert.match(help.stdout, /assembled/);
  for (const args of [
    [],
    ["--base", "../pantheon-base"],
    ["--descriptor", "a", "--descriptor", "b"],
    ["--descriptor", "a", "--artifact", "b"],
    ["--help", "--target", "."],
  ])
    assert.notEqual(
      spawnSync(process.execPath, [entry, ...args], { encoding: "utf8" })
        .status,
      0,
    );
});

test("CLI reports the assembly cause together with safe cleanup refusal", (t) => {
  const f = fixture(t);
  const preload = path.join(f.temp, "inject-foreign.mjs");
  fs.writeFileSync(preload, `import fs from 'node:fs'; import path from 'node:path';
    const copy = fs.copyFileSync; fs.copyFileSync = (source, target, ...args) => {
      const result = copy(source, target, ...args);
      if (target.endsWith(path.join('assembled', 'docs', 'base.md'))) {
        const unknown = path.join(path.dirname(target), 'foreign');
        fs.mkdirSync(unknown); fs.writeFileSync(path.join(unknown, 'keep.txt'), 'preserve');
        throw new Error('injected primary shared drift failure');
      } return result;
    };`);
  const entry = fileURLToPath(
    new URL("./rebuild-from-candidate.mjs", import.meta.url),
  );
  const result = spawnSync(
    process.execPath,
    [
      "--import", pathToFileURL(preload).href,
      entry,
      "--descriptor",
      f.options.descriptorPath,
      "--descriptor-sha256",
      f.options.expectedDescriptorSha256,
      "--artifact",
      f.options.artifactPath,
      "--ops-root",
      f.ops,
    ],
    { encoding: "utf8" },
  );
  assert.equal(result.status, 1, result.stderr);
  assert.match(result.stderr, /cleanup.*(identity|ownership)/i);
  assert.match(result.stderr, /Caused by: injected primary shared drift failure/);
  assert.equal(fs.existsSync(f.cacheRoot), true);
  assert.equal(fs.readFileSync(path.join(f.cacheRoot, 'assembled/docs/foreign/keep.txt'), 'utf8'), 'preserve');
  assert.equal(
    fs.existsSync(
      path.join(f.cacheRoot, "assembled/.foundation-candidate-receipt.json"),
    ),
    false,
  );
});
