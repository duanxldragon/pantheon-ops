import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import {
  assembleConsumerTree,
  assertSafeManifest,
  planBusinessMigrations,
} from "./rebuild-from-base.mjs";
import { checkSharedDrift } from "./check-shared-drift.mjs";
import {
  assertRealPath,
  assertRelativeInput,
  captureInputs,
  LIMITS,
  readCandidateDescriptor,
  readStableFile,
  removeCandidateStage,
  resolveCandidateArtifact,
  sha256,
} from "./foundation-candidate.mjs";

const defaultOpsRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);
const equal = (a, b) => JSON.stringify(a) === JSON.stringify(b);

function declaredGitFiles(root, manifest) {
  const listed = spawnSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "-z"],
    { cwd: root, encoding: "utf8" },
  );
  if (listed.status !== 0)
    throw new Error("Cannot freeze declared Ops Git file selection");
  const declared = [
    ...manifest.businessPaths,
    ...manifest.repositoryOverlayPaths,
  ];
  return listed.stdout
    .split("\0")
    .filter(
      (file) =>
        file &&
        declared.some(
          (entry) => file === entry || file.startsWith(`${entry}/`),
        ),
    )
    .sort();
}

export function rebuildFromCandidate({
  opsRoot = defaultOpsRoot,
  artifactPath,
  descriptorPath,
  expectedDescriptorSha256,
  targetRoot,
  manifestPath,
} = {}) {
  opsRoot = path.resolve(opsRoot);
  assertRealPath(opsRoot, { directory: true });
  const canonicalManifest = path.join(opsRoot, "business-overlay.json");
  const resolvedManifest = path.resolve(
    opsRoot,
    manifestPath ?? canonicalManifest,
  );
  // The existing assembler copies the canonical manifest into the product.
  // An alternative declaration would give drift a different ownership contract.
  if (resolvedManifest !== canonicalManifest)
    throw new Error(
      "Candidate assembly requires the canonical Ops business-overlay.json",
    );
  const manifestBytes = readStableFile(
    resolvedManifest,
    LIMITS.descriptorBytes,
  );
  const manifest = JSON.parse(manifestBytes.toString("utf8"));
  assertSafeManifest(manifest);
  const declared = [
    ...(manifest.businessPaths ?? []),
    ...(manifest.repositoryOverlayPaths ?? []),
    ...(manifest.businessMigrations ?? []),
    ...(manifest.replaceRoots ?? []),
    ...(manifest.baseExcludedPaths ?? []),
  ];
  for (const relative of declared) assertRelativeInput(relative);
  // These files are read implicitly by the existing assembler even when they
  // aren't explicitly copied as repository overlays.
  const inputPaths = [
    ...new Set([
      ...manifest.businessPaths,
      ...manifest.repositoryOverlayPaths,
      ...(manifest.businessMigrations ?? []),
      "business-overlay.json",
      "foundation-release.lock.json",
      "package.json",
      "frontend/package.json",
      ...(Object.keys(manifest.goDependencies ?? {}).length
        ? ["backend/go.sum"]
        : []),
    ]),
  ].sort();
  if (
    inputPaths.some(
      (relative) =>
        relative === ".tmp" ||
        relative.startsWith(".tmp/") ||
        relative === ".git" ||
        relative.startsWith(".git/"),
    )
  )
    throw new Error("Candidate staging cannot be an Ops source input");
  const before = captureInputs(opsRoot, inputPaths);
  const gitFilesBefore = declaredGitFiles(opsRoot, manifest);
  const lockBefore = readStableFile(
    path.join(opsRoot, "foundation-release.lock.json"),
  );
  const descriptor = readCandidateDescriptor(
    descriptorPath,
    expectedDescriptorSha256,
  );
  const cacheRoot = path.join(
    opsRoot,
    ".tmp/foundation-candidates",
    descriptor.candidateIdentity,
  );
  const assembled = path.join(cacheRoot, "assembled");
  if (targetRoot && path.resolve(opsRoot, targetRoot) !== assembled)
    throw new Error(
      "Candidate target must be .tmp/foundation-candidates/<candidateId>/assembled",
    );
  assertRealPath(cacheRoot, { directory: true, missing: true });
  if (fs.existsSync(cacheRoot))
    throw new Error(
      "Candidate stage already exists; refusing target destruction or identity reuse",
    );
  const snapshot = resolveCandidateArtifact({
    artifactPath,
    descriptorPath,
    expectedDescriptorSha256,
    cacheRoot,
  });
  try {
    const verifySources = () => {
      snapshot.verify();
      const current = captureInputs(opsRoot, inputPaths);
      if (
        !equal(before, current) ||
        !equal(gitFilesBefore, declaredGitFiles(opsRoot, manifest)) ||
        !lockBefore.equals(
          readStableFile(path.join(opsRoot, "foundation-release.lock.json")),
        )
      )
        throw new Error(
          "Declared Ops source/manifest/lock changed during candidate assembly",
        );
      return current;
    };
    verifySources();
    assertRealPath(assembled, { directory: true, missing: true });
    if (fs.existsSync(assembled))
      throw new Error("Candidate assembly target was created concurrently");
    const result = assembleConsumerTree({
      opsRoot,
      snapshot,
      targetRoot: assembled,
      manifestPath: resolvedManifest,
    });
    const after = verifySources();
    const assembledInputs = captureInputs(
      assembled,
      fs.readdirSync(assembled).sort(),
    );
    const driftExitCode = checkSharedDrift({ root: assembled, snapshot });
    verifySources();
    if (driftExitCode !== 0)
      throw new Error("Candidate shared drift check failed");
    const migrations = planBusinessMigrations(manifest).map(
      ({ source, target }) => {
        const sourceBytes = readStableFile(path.join(opsRoot, source));
        if (!sourceBytes.equals(readStableFile(path.join(assembled, target))))
          throw new Error(
            "Derived candidate migration differs from declared business source",
          );
        return { source, target, sha256: sha256(sourceBytes) };
      },
    );
    if (
      !lockBefore.equals(
        readStableFile(path.join(assembled, "foundation-release.lock.json")),
      )
    )
      throw new Error(
        "Assembled candidate must preserve the official lock bytes",
      );
    if (
      !equal(
        assembledInputs,
        captureInputs(assembled, fs.readdirSync(assembled).sort()),
      )
    )
      throw new Error("Assembled candidate changed during verification");
    const sharedRoots = Object.values(descriptor.sharedPaths).flat();
    const receipt = {
      schemaVersion: 1,
      kind: "pantheon-ops-candidate-assembly",
      consumerMode: "candidate-staging",
      candidateIdentity: descriptor.candidateIdentity,
      descriptorSha256: expectedDescriptorSha256,
      artifact: descriptor.artifact,
      baseCommit: descriptor.baseCommit,
      capabilities: descriptor.capabilities,
      baseFiles: descriptor.files,
      aggregateSha256: descriptor.aggregateSha256,
      opsManifestSha256: sha256(manifestBytes),
      opsSourceFilesBefore: before.files,
      opsSourceFilesAfter: after.files,
      opsDeclaredGitFilesBefore: gitFilesBefore,
      opsDeclaredGitFilesAfter: declaredGitFiles(opsRoot, manifest),
      opsSourceStampsSha256: sha256(Buffer.from(JSON.stringify(before.stamps))),
      assembledSourceFiles: assembledInputs.files,
      assembledSourceSha256: sha256(
        Buffer.from(JSON.stringify(assembledInputs.files)),
      ),
      assembledSharedFiles: assembledInputs.files.filter((file) =>
        sharedRoots.some(
          (root) => file.path === root || file.path.startsWith(`${root}/`),
        ),
      ),
      generatedBusinessMigrations: migrations,
      officialLockSha256Before: sha256(lockBefore),
      officialLockSha256After: sha256(
        readStableFile(path.join(opsRoot, "foundation-release.lock.json")),
      ),
      evidenceIds: descriptor.evidenceIds,
      reviewIds: descriptor.reviewIds,
      checks: [
        { operation: "assembleConsumerTree", result: "passed" },
        { operation: "checkSharedDrift", exitCode: driftExitCode },
        { operation: "source-stability-before-and-after", result: "passed" },
      ],
      sourceStable: true,
      officialConsumption: false,
      runtimeEvidence:
        "Not collected by this staging tool; independent API/tool review, build receipt and paired runtime remain required.",
    };
    const receiptPath = path.join(
      assembled,
      ".foundation-candidate-receipt.json",
    );
    verifySources();
    fs.writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, {
      flag: "wx",
    });
    return { ...result, snapshot, receiptPath, receipt };
  } catch (error) {
    removeCandidateStage(cacheRoot, { cause: error });
    throw error;
  }
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  try {
    const args = process.argv.slice(2);
    if (args.length === 1 && args[0] === "--help") {
      console.log(
        "Usage: node scripts/business-overlay/rebuild-from-candidate.mjs --descriptor <candidate-descriptor.json> --descriptor-sha256 <reviewed-64-hex-digest> --artifact <candidate.tar> [--ops-root <Ops-root>] [--target <.tmp/foundation-candidates/candidate-aggregate/assembled>]",
      );
      console.log(
        "Example: node scripts/business-overlay/rebuild-from-candidate.mjs --descriptor ../pantheon-base/.tmp/foundation-candidates/reviewed/candidate-descriptor.json --descriptor-sha256 <digest-from-independent-review> --artifact ../pantheon-base/.tmp/foundation-candidates/reviewed/candidate-<aggregateSha256>.tar",
      );
      console.log(
        "Only a new candidate stage is accepted. This does not build/start a product or change the official release lock.",
      );
    } else {
      const allowed = new Set([
        "--descriptor",
        "--descriptor-sha256",
        "--artifact",
        "--ops-root",
        "--target",
      ]);
      const options = new Map();
      for (let index = 0; index < args.length; index += 2) {
        if (
          !allowed.has(args[index]) ||
          options.has(args[index]) ||
          !args[index + 1] ||
          args[index + 1].startsWith("--")
        )
          throw new Error(
            "Invalid or duplicate candidate CLI argument; use --help",
          );
        options.set(args[index], args[index + 1]);
      }
      if (
        !["--descriptor", "--descriptor-sha256", "--artifact"].every((key) =>
          options.has(key),
        )
      )
        throw new Error(
          "Descriptor, externally pinned SHA256 and artifact are required; use --help",
        );
      const result = rebuildFromCandidate({
        opsRoot: options.get("--ops-root"),
        descriptorPath: options.get("--descriptor"),
        expectedDescriptorSha256: options.get("--descriptor-sha256"),
        artifactPath: options.get("--artifact"),
        targetRoot: options.get("--target"),
      });
      console.log(
        JSON.stringify(
          {
            consumerMode: result.receipt.consumerMode,
            candidateIdentity: result.receipt.candidateIdentity,
            targetRoot: result.targetRoot,
            receiptPath: result.receiptPath,
            officialConsumption: false,
          },
          null,
          2,
        ),
      );
    }
  } catch (error) {
    console.error(error.message);
    if (error.cause) console.error(`Caused by: ${error.cause.message}`);
    process.exitCode = 1;
  }
}
