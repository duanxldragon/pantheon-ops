import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { buildCandidateArtifact } from "../../../pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs";
import * as producer from "../../../pantheon-base/scripts/foundation-release/candidate-descriptor.mjs";
import * as consumer from "./foundation-candidate.mjs";
import { rebuildFromCandidate } from "./rebuild-from-candidate.mjs";

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
    ...(phase === "assembly" ? ["assembled"] : []),
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
              : directory === "nested"
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

test("assembly rejects snapshot/artifact and declared Ops mutations including edit/revert; unknown assembled ownership retains stage", async (t) => {
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
      // The unchanged assembler has no directory creation receipt. Its output
      // must be retained when cleanup cannot prove the directory instances.
      const causeMessage =
        mode === "snapshot"
          ? /Candidate snapshot changed during assembly/
          : mode === "artifact"
            ? /Candidate artifact inputs changed during assembly/
            : /Declared Ops source\/manifest\/lock changed during candidate assembly/;
      assertCleanupRefused(() => interceptCopy(action, f.rebuild), causeMessage);
      assert.equal(fs.existsSync(f.cacheRoot), true);
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
    assertCleanupRefused(f.rebuild, /Candidate shared drift check failed/);
    assert.equal(fs.existsSync(f.cacheRoot), true);
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
    assertCleanupRefused(
      f.rebuild,
      /Business migration version conflicts with Base: 000031/,
    );
    assert.equal(fs.existsSync(f.cacheRoot), true);
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
  write(f.ops, ".foundation/allowed-drift.json", "{}");
  const entry = fileURLToPath(
    new URL("./rebuild-from-candidate.mjs", import.meta.url),
  );
  const result = spawnSync(
    process.execPath,
    [
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
  assert.match(result.stderr, /Caused by: Candidate shared drift check failed/);
  assert.equal(fs.existsSync(f.cacheRoot), true);
  assert.equal(
    fs.existsSync(
      path.join(f.cacheRoot, "assembled/.foundation-candidate-receipt.json"),
    ),
    false,
  );
});
