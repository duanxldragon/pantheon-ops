// Self-contained schema1 consumer. Keep protocol semantics aligned with the Base
// producer; this file never loads tooling or source from a live Base checkout.
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";

export const LIMITS = Object.freeze({
  files: 50000,
  fileBytes: 32 * 1024 * 1024,
  archiveBytes: 512 * 1024 * 1024,
  descriptorBytes: 16 * 1024 * 1024,
});
// Declared Ops metadata also includes retained official cache/evidence. Its
// bounded capture is separate from the schema1 candidate archive inventory.
export const METADATA_LIMITS = Object.freeze({ files: 100000 });
export const sha256 = (bytes) =>
  crypto.createHash("sha256").update(bytes).digest("hex");
export const aggregateSha256 = (files) =>
  sha256(Buffer.from(JSON.stringify(files)));
const digest = /^[a-f0-9]{64}$/u;
const compare = (a, b) => (a < b ? -1 : a > b ? 1 : 0);

// Capabilities describe reviewed source interfaces, not runtime acceptance.
export const CAPABILITY_EXPORTS = Object.freeze({
  "business-tenant-v2": [
    [
      "backend/pkg/contracts/business_tenant_groups.go",
      "BusinessTenantGroupOptions",
    ],
    [
      "backend/pkg/contracts/business_tenant_groups.go",
      "TenantScopedBusinessGroup",
    ],
    [
      "backend/pkg/contracts/business_tenant_groups.go",
      "RegisterBusinessTenantModeReader",
    ],
    [
      "backend/pkg/contracts/business_tenant_groups.go",
      "RegisterBusinessAuthorizationReader",
    ],
    ["backend/pkg/tenant/mode_reader.go", "ModeReader"],
    ["backend/pkg/contracts/businessauth/businessauth.go", "Reader"],
    [
      "backend/internal/middleware/business_token_middleware.go",
      "BusinessTokenAuthMiddleware",
    ],
  ],
  "required-business-bootstrap-v1": [
    ["backend/pkg/contracts/module.go", "RegisterRequiredBackendModules"],
  ],
  "audit-metadata-only-v1": [
    ["backend/pkg/auditpolicy/policy.go", "RegisterMetadataOnly"],
    ["backend/pkg/auditpolicy/policy.go", "Seal"],
    ["backend/pkg/auditpolicy/policy.go", "SetMetadata"],
  ],
  "permission-route-registration-v1": [
    ["backend/pkg/permissionroutes/registry.go", "Register"],
  ],
});

// Curated PUBLIC review leaves required by the existing consumer-artifact
// contract. Keep this block byte-equivalent in both schema1 validators. This
// permits no recursive selection or other private .harness payload.
export const PUBLIC_REVIEW_ARCHIVE_PATHS = Object.freeze([
  ".harness/archive/2026-08/evidence/2026-08-31-base-operational-workbench-design/review.md",
  ".harness/archive/2026-09/evidence/2026-09-04-readme-v0.11.0-update/review.md",
  ".harness/archive/2026-09/evidence/2026-09-10-tenant-canary-slice/review.md",
  ".harness/archive/2026-09/evidence/2026-09-10-tenant-contract-design/review.md",
  ".harness/archive/2026-09/evidence/2026-09-10-tenant-core-auth-iam/review.md",
  ".harness/archive/2026-09/evidence/2026-09-10-tenant-migration-runbook/review.md",
  ".harness/archive/2026-09/evidence/2026-09-10-tenant-ready-guardrails/review.md",
  ".harness/archive/2026-09/evidence/2026-09-24-fix-report-residual-closeout/review.md"
]);

export function assertPublicReviewPackaging(entries) {
  const payload = new Map(entries.map(entry => [entry.path, entry.bytes]));
  const configPath = 'config/foundation-consumer-artifacts.json';
  const registryPath = 'config/review-loop-legacy-artifacts.json';
  const archival = entries.filter(entry => entry.path.startsWith('.harness/'));
  if (!payload.has(configPath) && !payload.has(registryPath) && !archival.length) return;
  if (!payload.has(configPath) || !payload.has(registryPath))
    throw new Error('Public review packaging requires both original registry configurations');
  const parse = file => JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(payload.get(file)));
  const config = parse(configPath);
  if (config.schemaVersion !== 1 || config.sourceOwner !== 'pantheon-base' ||
      config.reviewLegacyRegistry?.path !== registryPath ||
      config.reviewLegacyRegistry?.artifactRoot !== '.harness/archive')
    throw new Error('Invalid Base consumer review packaging contract');
  const registry = parse(registryPath);
  const refs = registry.entries;
  if (registry.schemaVersion !== 1 || !Array.isArray(refs) ||
      refs.length > PUBLIC_REVIEW_ARCHIVE_PATHS.length || new Set(refs).size !== refs.length ||
      refs.some(ref => !PUBLIC_REVIEW_ARCHIVE_PATHS.includes(ref)))
    throw new Error('Registry must reference only curated public review leaves');
  if (refs.some(ref => !payload.has(ref)) ||
      archival.some(entry => !refs.includes(entry.path)))
    throw new Error('Public review archive payload must exactly match registry references');
}

export function assertCandidatePath(value) {
  if (
    typeof value !== "string" ||
    !value ||
    value !== value.normalize("NFC") ||
    /[\\:<>"|?*\x00-\x1f\x7f]/u.test(value) ||
    value.startsWith("/") ||
    value
      .split("/")
      .some(
        (part) =>
          !part ||
          part === "." ||
          part === ".." ||
          /[. ]$/u.test(part) ||
          /^(?:con|prn|aux|nul|com[0-9¹²³]|lpt[0-9¹²³])(?:\.|$)/iu.test(part),
      )
  ) {
    throw new Error(`Unsafe candidate path: ${value}`);
  }
  if (PUBLIC_REVIEW_ARCHIVE_PATHS.includes(value)) return value;
  const parts = value.toLowerCase().split("/");
  if (
    parts.some(
      (part) =>
        [
          ".git",
          ".omx",
          ".harness",
          ".aws",
          ".ssh",
          ".codex",
          ".claude",
          ".tmp",
          "node_modules",
          "dist",
          "build",
          "coverage",
          "uploads",
          "cache",
          "caches",
        ].includes(part) ||
        part.startsWith(".env") ||
        /^(?:credentials|secrets?)(?:\.|$)/u.test(part) ||
        /\.(?:pem|key|p12|pfx)$/u.test(part),
    )
  ) {
    throw new Error(`Excluded candidate path: ${value}`);
  }
  return value;
}

export function assertRealPath(
  absolute,
  { directory = false, missing = false } = {},
) {
  const resolved = path.resolve(absolute);
  let current = path.parse(resolved).root;
  for (const part of resolved
    .slice(current.length)
    .split(path.sep)
    .filter(Boolean)) {
    current = path.join(current, part);
    if (!fs.existsSync(current)) {
      // existsSync follows links; lstat also detects dangling links.
      try {
        fs.lstatSync(current);
      } catch (error) {
        if (error.code === "ENOENT" && missing) return resolved;
        throw error;
      }
    }
    const stat = fs.lstatSync(current);
    if (stat.isSymbolicLink() || (!stat.isDirectory() && current !== resolved))
      throw new Error(
        "Candidate paths must use real directories and regular files",
      );
    if (
      current === resolved &&
      (directory ? !stat.isDirectory() : !stat.isFile())
    )
      throw new Error("Candidate path has unsupported file type");
  }
  return resolved;
}

export function assertInventory(files) {
  if (!Array.isArray(files) || !files.length || files.length > LIMITS.files)
    throw new Error("Invalid candidate file count");
  const seen = new Set();
  const directories = new Set();
  const spellings = new Map();
  let previous = "";
  let total = 0;
  for (const file of files) {
    assertCandidatePath(file.path);
    if (
      Object.keys(file).sort().join(",") !== "bytes,path,sha256" ||
      !Number.isSafeInteger(file.bytes) ||
      file.bytes < 0 ||
      file.bytes > LIMITS.fileBytes ||
      !digest.test(file.sha256)
    )
      throw new Error("Invalid candidate file metadata");
    if (compare(previous, file.path) >= 0)
      throw new Error("Candidate inventory must be sorted and unique");
    previous = file.path;
    const folded = file.path.toLowerCase();
    if (seen.has(folded) || directories.has(folded))
      throw new Error("Candidate path collision");
    const parts = folded.split("/");
    const originalParts = file.path.split("/");
    for (let i = 1; i <= parts.length; i++) {
      const foldedPath = parts.slice(0, i).join("/");
      const spelling = originalParts.slice(0, i).join("/");
      if (spellings.has(foldedPath) && spellings.get(foldedPath) !== spelling)
        throw new Error("Candidate case-fold collision");
      spellings.set(foldedPath, spelling);
    }
    for (let i = 1; i < parts.length; i++) {
      const parent = parts.slice(0, i).join("/");
      if (seen.has(parent))
        throw new Error("Candidate file/directory collision");
      directories.add(parent);
    }
    seen.add(folded);
    total += file.bytes + 512 + ((512 - (file.bytes % 512)) % 512);
  }
  if (total + 1024 > LIMITS.archiveBytes)
    throw new Error("Candidate archive size exceeds limit");
  return files;
}

function ids(values, label) {
  if (
    !Array.isArray(values) ||
    !values.length ||
    values.length > 100 ||
    new Set(values).size !== values.length ||
    values.some(
      (id) =>
        typeof id !== "string" ||
        !/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$/u.test(id),
    )
  )
    throw new Error(`Invalid ${label}`);
}

export function candidateExports(capabilities, files) {
  ids(capabilities, "capabilities");
  const inventory = new Map(files.map((file) => [file.path, file]));
  const exports = new Map();
  for (const capability of capabilities) {
    if (!Object.hasOwn(CAPABILITY_EXPORTS, capability))
      throw new Error(`Unknown candidate capability: ${capability}`);
    for (const [file, name] of CAPABILITY_EXPORTS[capability]) {
      if (!inventory.has(file))
        throw new Error(`Missing required API: ${file}#${name}`);
      exports.set(`${file}#${name}`, {
        path: file,
        name,
        sha256: inventory.get(file).sha256,
      });
    }
  }
  return [...exports.values()].sort((a, b) =>
    compare(`${a.path}#${a.name}`, `${b.path}#${b.name}`),
  );
}

export function validateCandidateDescriptor(descriptor) {
  if (
    !descriptor ||
    Object.keys(descriptor).sort().join(",") !==
      "aggregateSha256,artifact,baseCommit,candidateIdentity,capabilities,consumerMode,evidenceIds,files,kind,requiredExports,reviewIds,schemaVersion,sharedPaths,sourceRepo" ||
    descriptor.schemaVersion !== 1 ||
    descriptor.kind !== "pantheon-base-candidate" ||
    descriptor.consumerMode !== "candidate-staging" ||
    descriptor.sourceRepo !== "pantheon-base" ||
    !/^[a-f0-9]{40}$/u.test(descriptor.baseCommit)
  )
    throw new Error("Invalid candidate descriptor schema/provenance");
  assertInventory(descriptor.files);
  const aggregate = aggregateSha256(descriptor.files);
  if (
    descriptor.aggregateSha256 !== aggregate ||
    descriptor.candidateIdentity !== `candidate-${aggregate}`
  )
    throw new Error("Candidate aggregate/identity mismatch");
  if (
    Object.keys(descriptor.artifact ?? {})
      .sort()
      .join(",") !== "assetName,sha256" ||
    descriptor.artifact.assetName !== `${descriptor.candidateIdentity}.tar` ||
    !digest.test(descriptor.artifact.sha256)
  )
    throw new Error("Invalid candidate archive identity");
  ids(descriptor.evidenceIds, "evidence IDs");
  ids(descriptor.reviewIds, "review IDs");
  const expectedExports = candidateExports(
    descriptor.capabilities,
    descriptor.files,
  );
  if (
    JSON.stringify(expectedExports) !==
    JSON.stringify(descriptor.requiredExports)
  )
    throw new Error("Candidate capability/export mismatch");
  const shared = descriptor.sharedPaths;
  if (
    !shared ||
    Array.isArray(shared) ||
    !Object.keys(shared).length ||
    Object.keys(shared).some(
      (group) => !["backend", "frontend", "docs"].includes(group),
    )
  )
    throw new Error("Invalid candidate sharedPaths");
  for (const entries of Object.values(shared)) {
    if (
      !Array.isArray(entries) ||
      !entries.length ||
      new Set(entries).size !== entries.length
    )
      throw new Error("Empty/duplicate candidate shared paths");
    for (const entry of entries) {
      assertCandidatePath(entry);
      if (
        !descriptor.files.some(
          (file) => file.path === entry || file.path.startsWith(`${entry}/`),
        )
      )
        throw new Error(`Missing shared inventory: ${entry}`);
    }
  }
  return descriptor;
}

export function readCandidateDescriptor(
  descriptorPath,
  expectedDescriptorSha256,
) {
  if (!digest.test(expectedDescriptorSha256))
    throw new Error("Pinned descriptor SHA256 is required");
  const bytes = readStableFile(descriptorPath, LIMITS.descriptorBytes);
  if (sha256(bytes) !== expectedDescriptorSha256)
    throw new Error("Candidate descriptor digest mismatch");
  return validateCandidateDescriptor(
    JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)),
  );
}

// Only regular ustar entries are emitted/accepted. No native extractor ever
// sees unchecked headers, including PAX/GNU extensions or hard/symbolic links.
export function verifyCandidateArchive(archive, descriptor) {
  validateCandidateDescriptor(descriptor);
  if (
    archive.length > LIMITS.archiveBytes ||
    archive.length % 512 ||
    sha256(archive) !== descriptor.artifact.sha256
  )
    throw new Error("Candidate archive digest/size mismatch");
  const entries = [];
  let offset = 0;
  while (offset + 512 <= archive.length) {
    const header = archive.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) {
      if (
        archive.length - offset < 1024 ||
        !archive.subarray(offset).every((byte) => byte === 0)
      )
        throw new Error("Invalid candidate tar trailer");
      break;
    }
    const field = (start, length) => {
      const raw = header.subarray(start, start + length);
      const nul = raw.indexOf(0);
      if (nul >= 0 && raw.subarray(nul).some((byte) => byte !== 0))
        throw new Error("Invalid candidate tar field");
      const value = raw.subarray(0, nul < 0 ? raw.length : nul);
      const decoded = new TextDecoder("utf-8", { fatal: true }).decode(value);
      return decoded;
    };
    const numeric = (start, length) => {
      const value = header
        .subarray(start, start + length)
        .toString("ascii")
        .replace(/[\0 ]+$/u, "");
      if (!/^[0-7]+$/u.test(value))
        throw new Error("Unsupported candidate tar number");
      return parseInt(value, 8);
    };
    const checksum = numeric(148, 8);
    const sum = [...header].reduce(
      (total, byte, index) => total + (index >= 148 && index < 156 ? 32 : byte),
      0,
    );
    if (
      checksum !== sum ||
      field(257, 6) !== "ustar" ||
      field(263, 2) !== "00" ||
      ![0, 48].includes(header[156]) ||
      field(157, 100)
    )
      throw new Error("Unsupported candidate tar header/type/link");
    const prefix = field(345, 155);
    const file = assertCandidatePath(
      `${prefix ? `${prefix}/` : ""}${field(0, 100)}`,
    );
    const size = numeric(124, 12);
    const end = offset + 512 + size;
    const next = end + ((512 - (size % 512)) % 512);
    if (
      size > LIMITS.fileBytes ||
      next > archive.length ||
      archive.subarray(end, next).some((byte) => byte !== 0)
    )
      throw new Error("Invalid candidate tar content/padding");
    entries.push({ path: file, bytes: archive.subarray(offset + 512, end) });
    if (entries.length > LIMITS.files)
      throw new Error("Candidate archive entry count exceeds limit");
    offset = next;
  }
  if (offset + 1024 > archive.length)
    throw new Error("Missing candidate tar trailer");
  const files = entries.map((entry) => ({
    path: entry.path,
    bytes: entry.bytes.length,
    sha256: sha256(entry.bytes),
  }));
  assertInventory(files);
  if (JSON.stringify(files) !== JSON.stringify(descriptor.files))
    throw new Error("Candidate archive inventory mismatch");
  assertPublicReviewPackaging(entries);
  return entries;
}

export function verifyCandidateExecutable({
  descriptorPath,
  expectedDescriptorSha256,
  buildReceiptPath,
  expectedBuildReceiptSha256,
  executablePath,
  expectedExecutableSha256,
}) {
  const descriptor = readCandidateDescriptor(
    descriptorPath,
    expectedDescriptorSha256,
  );
  if (
    !digest.test(expectedBuildReceiptSha256) ||
    !digest.test(expectedExecutableSha256)
  )
    throw new Error("Pinned build/executable digests are required");
  const bytes = readStableFile(buildReceiptPath, LIMITS.descriptorBytes);
  if (
    bytes.length > LIMITS.descriptorBytes ||
    sha256(bytes) !== expectedBuildReceiptSha256
  )
    throw new Error("Candidate build receipt digest mismatch");
  const receipt = JSON.parse(bytes.toString("utf8"));
  if (
    Object.keys(receipt).sort().join(",") !==
      "aggregateSha256,candidateIdentity,capabilities,descriptorSha256,evidenceIds,executable,kind,reviewIds,schemaVersion" ||
    receipt.schemaVersion !== 1 ||
    receipt.kind !== "pantheon-candidate-build" ||
    receipt.candidateIdentity !== descriptor.candidateIdentity ||
    receipt.descriptorSha256 !== expectedDescriptorSha256 ||
    receipt.aggregateSha256 !== descriptor.aggregateSha256 ||
    JSON.stringify(receipt.capabilities) !==
      JSON.stringify(descriptor.capabilities)
  )
    throw new Error("Candidate build identity/capability mismatch");
  ids(receipt.evidenceIds, "build evidence IDs");
  ids(receipt.reviewIds, "build review IDs");
  if (
    Object.keys(receipt.executable ?? {})
      .sort()
      .join(",") !== "bytes,sha256"
  )
    throw new Error("Invalid candidate executable schema");
  const executable = readStableFile(executablePath, LIMITS.archiveBytes);
  if (
    receipt.executable?.sha256 !== expectedExecutableSha256 ||
    sha256(executable) !== expectedExecutableSha256 ||
    receipt.executable.bytes !== executable.length
  )
    throw new Error("Candidate executable hash/size mismatch");
  return receipt;
}

const stamp = (stat) => [
  stat.dev,
  stat.ino,
  stat.size,
  stat.mtimeMs,
  stat.ctimeMs,
  stat.mode,
  stat.nlink,
];
const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);
const ownedStages = new Map();

export class CandidateCleanupError extends Error {
  constructor(root, reason, cause) {
    super(
      `Candidate cleanup ownership/identity refused: ${reason}; retained stage ${root}${cause ? `; original failure: ${cause.message}` : ""}`,
      { cause },
    );
    this.name = "CandidateCleanupError";
    this.code = "ERR_CANDIDATE_CLEANUP_OWNERSHIP";
  }
}

function directoryIdentity(absolute) {
  // Windows file IDs can exceed Number.MAX_SAFE_INTEGER. Never round them or
  // substitute timestamps/markers when the filesystem cannot supply an ID.
  const stat = fs.lstatSync(absolute, { bigint: true });
  if (
    !stat.isDirectory() ||
    stat.isSymbolicLink() ||
    typeof stat.dev !== "bigint" ||
    stat.dev < 0n ||
    typeof stat.ino !== "bigint" ||
    stat.ino <= 0n
  )
    throw new Error(`Unavailable real directory identity: ${absolute}`);
  return [stat.dev, stat.ino];
}

function directoryChain(absolute) {
  const chain = [];
  for (let current = absolute; ; current = path.dirname(current)) {
    chain.unshift(current);
    if (current === path.dirname(current)) return chain;
  }
}

function assertDirectoryIdentity(absolute, identities) {
  const expected = identities.get(absolute);
  if (!expected) throw new Error(`Unknown directory ownership: ${absolute}`);
  const current = directoryIdentity(absolute);
  if (current[0] !== expected[0] || current[1] !== expected[1])
    throw new Error(`Replaced directory identity: ${absolute}`);
}

function assertOwnedChain(absolute, identities) {
  for (const directory of directoryChain(absolute))
    assertDirectoryIdentity(directory, identities);
}

function createStageDirectory(absolute, stage) {
  const identities = stage.directories;
  assertOwnedChain(path.dirname(absolute), identities);
  if (identities.has(absolute)) {
    assertDirectoryIdentity(absolute, identities);
    return;
  }
  // Node/Windows has no atomic mkdir+authenticated handle primitive. This pin
  // can detect later replacement; it NEVER grants directory deletion authority.
  // Record attempts before mkdir so an exception after native creation retains it.
  stage.unprovenDirectories.add(absolute);
  fs.mkdirSync(absolute);
  identities.set(absolute, directoryIdentity(absolute));
  assertOwnedChain(absolute, identities);
  if (fs.readdirSync(absolute).length)
    throw new Error(`Unproven directory creation contains unknown entries: ${absolute}`);
}

function fileIdentity(stat, absolute) {
  if (!stat.isFile() || stat.isSymbolicLink() || stat.nlink !== 1n ||
      typeof stat.dev !== "bigint" || typeof stat.ino !== "bigint" || stat.ino <= 0n)
    throw new Error(`Unavailable regular file identity: ${absolute}`);
  return [stat.dev, stat.ino, stat.birthtimeNs];
}

function assertFileIdentity(absolute, stage, fd) {
  assertOwnedChain(path.dirname(absolute), stage.directories);
  const expected = stage.files.get(absolute);
  if (!expected) throw new Error(`Unknown file ownership: ${absolute}`);
  const current = fileIdentity(fs.lstatSync(absolute, { bigint: true }), absolute);
  const matches = value => value.every((part, i) => part === expected[i]);
  if (!matches(current) || (fd !== undefined &&
      !matches(fileIdentity(fs.fstatSync(fd, { bigint: true }), absolute))))
    throw new Error(`Replaced regular file identity: ${absolute}`);
  assertOwnedChain(path.dirname(absolute), stage.directories);
}

function writeStageFile(absolute, bytes, options, stage) {
  assertOwnedChain(path.dirname(absolute), stage.directories);
  const settings = typeof options === "string" ? { encoding: options } : (options ?? {});
  const exclusive = String(settings.flag ?? "w").includes("x");
  const existing = stage.files.has(absolute);
  if (existing) assertFileIdentity(absolute, stage);
  else {
    try {
      fs.lstatSync(absolute);
      throw new Error(`Unknown file ownership: ${absolute}`);
    } catch (error) { if (error.code !== "ENOENT") throw error; }
  }
  // No O_TRUNC on path open, no native copy-to-path. An exclusive-created fd
  // authenticates that file instance even if the namespace changes afterward.
  const flags = fs.constants.O_WRONLY | (fs.constants.O_NOFOLLOW ?? 0) |
    ((!existing || exclusive) ? fs.constants.O_CREAT | fs.constants.O_EXCL : 0);
  let fd;
  let primaryFailure;
  try {
    fd = fs.openSync(absolute, flags, settings.mode ?? 0o644);
    if (!existing) stage.files.set(absolute,
      fileIdentity(fs.fstatSync(fd, { bigint: true }), absolute));
    assertFileIdentity(absolute, stage, fd);
    const buffer = Buffer.isBuffer(bytes) ? bytes :
      (ArrayBuffer.isView(bytes) ? Buffer.from(bytes.buffer, bytes.byteOffset, bytes.byteLength) : Buffer.from(bytes, settings.encoding ?? "utf8"));
    fs.ftruncateSync(fd, buffer.length);
    assertFileIdentity(absolute, stage, fd);
    fs.writeFileSync(fd, buffer);
    assertFileIdentity(absolute, stage, fd);
  } catch (error) {
    primaryFailure = error;
    throw error;
  } finally {
    if (fd !== undefined) {
      try { fs.closeSync(fd); }
      catch (closeError) { if (!primaryFailure) throw closeError; }
    }
  }
  assertFileIdentity(absolute, stage);
}
const freeze = (value) => {
  if (value && typeof value === "object") {
    for (const child of Object.values(value)) freeze(child);
    Object.freeze(value);
  }
  return value;
};
freeze(CAPABILITY_EXPORTS);

export function readStableFile(absolute, maxBytes = LIMITS.fileBytes) {
  assertRealPath(absolute);
  const before = fs.lstatSync(absolute);
  if (before.nlink !== 1 || before.size > maxBytes)
    throw new Error(
      "Candidate input must be a bounded regular file without hard links",
    );
  const fd = fs.openSync(
    absolute,
    fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW ?? 0),
  );
  let bytes;
  try {
    if (!same(stamp(before), stamp(fs.fstatSync(fd))))
      throw new Error("Candidate input changed while opening");
    bytes = fs.readFileSync(fd);
    if (
      bytes.length > maxBytes ||
      !same(stamp(before), stamp(fs.fstatSync(fd)))
    )
      throw new Error("Candidate input changed while reading");
  } finally {
    fs.closeSync(fd);
  }
  assertRealPath(absolute);
  if (!same(stamp(before), stamp(fs.lstatSync(absolute))))
    throw new Error("Candidate input path changed while reading");
  return bytes;
}

// Also used for declared Ops metadata, whose paths can legitimately contain
// .harness/.foundation. Archive inputs always use the stricter protocol check.
export function assertRelativeInput(value) {
  if (
    typeof value !== "string" ||
    !value ||
    value.length > 240 ||
    value !== value.normalize("NFC") ||
    value.includes("\\") ||
    value.startsWith("/") ||
    /[:<>"|?*\x00-\x1f\x7f]/u.test(value) ||
    value
      .split("/")
      .some(
        (part) => !part || part === "." || part === ".." || /[. ]$/u.test(part) ||
          /^(?:con|prn|aux|nul|com[0-9¹²³]|lpt[0-9¹²³])(?:\.|$)/iu.test(part),
      )
  )
    throw new Error("Unsafe declared Ops input path");
  return value;
}

// Ops uses explicit Git-selected leaves. Full snapshot/assembled capture below
// deliberately retains its recursive semantics and all existing bounds.
export function captureSelectedInputs(root, selection) {
  assertRealPath(root, { directory: true });
  const parents = new Map();
  const declarations = new Map();
  const fileStamps = new Map();
  const pinParents = absolute => {
    for (const parent of directoryChain(path.dirname(absolute))) {
      if (parents.has(parent)) continue;
      assertRealPath(parent, { directory: true });
      const stat = fs.lstatSync(parent, { bigint: true });
      parents.set(parent, [...directoryIdentity(parent), stat.mode].map(String));
    }
  };
  for (const relative of selection.declarations) {
    assertRelativeInput(relative);
    const absolute = path.join(root, relative);
    const stat = fs.lstatSync(absolute, { bigint: true });
    assertRealPath(absolute, { directory: stat.isDirectory() });
    pinParents(absolute);
    declarations.set(relative, [stat.dev, stat.ino, stat.mode].map(String));
    if (stat.isDirectory()) parents.set(absolute, [...directoryIdentity(absolute), stat.mode].map(String));
  }
  for (const relative of selection.files) {
    assertRelativeInput(relative);
    const absolute = path.join(root, relative);
    assertRealPath(absolute);
    pinParents(absolute);
    const stat = fs.lstatSync(absolute, { bigint: true });
    fileStamps.set(relative, [stat.dev, stat.ino, stat.size, stat.mtimeNs, stat.ctimeNs, stat.mode, stat.nlink].map(String));
  }
  const captured = captureInputs(root, selection.files);
  for (const [absolute, identity] of parents) {
    assertRealPath(absolute, { directory: true });
    const stat = fs.lstatSync(absolute, { bigint: true });
    if (!same(identity, [...directoryIdentity(absolute), stat.mode].map(String)))
      throw new Error('Selected Ops input ancestor changed during capture');
  }
  for (const [relative, identity] of fileStamps) {
    const stat = fs.lstatSync(path.join(root, relative), { bigint: true });
    if (!same(identity, [stat.dev, stat.ino, stat.size, stat.mtimeNs, stat.ctimeNs, stat.mode, stat.nlink].map(String)))
      throw new Error('Selected Ops input changed during capture');
  }
  return { files: captured.files, stamps: [...fileStamps], parents: [...parents].sort(([a], [b]) => compare(a, b)),
    declarations: [...declarations] };
}

// Bind the bytes actually returned by a read to the previously captured leaves.
// Stat/hash recapture alone cannot authenticate a cached or injected read result.
export function frozenInputReader(root, captured) {
  const files = new Map(captured.files.map(file => [file.path, file]));
  const stamps = new Map(captured.stamps ?? []);
  const parents = new Map(captured.parents ?? []);
  const check = (absolute, relative) => {
    for (const parent of directoryChain(path.dirname(absolute))) {
      assertRealPath(parent, { directory: true });
      const expected = parents.get(parent);
      if (expected) {
        const stat = fs.lstatSync(parent, { bigint: true });
        if (!same(expected, [...directoryIdentity(parent), stat.mode].map(String)))
          throw new Error('Frozen input ancestor changed: ' + relative);
      }
    }
    assertRealPath(absolute);
    const expected = stamps.get(relative);
    if (expected) {
      const stat = fs.lstatSync(absolute, { bigint: true });
      if (!same(expected, [stat.dev, stat.ino, stat.size, stat.mtimeNs, stat.ctimeNs, stat.mode, stat.nlink].map(String)))
        throw new Error('Frozen input stamp changed: ' + relative);
    }
  };
  return (absolute, encoding) => {
    const relative = assertRelativeInput(path.relative(root, absolute).replaceAll('\\', '/'));
    const expected = files.get(relative);
    if (!expected) throw new Error('Unselected frozen source input: ' + relative);
    check(absolute, relative);
    const bytes = readStableFile(absolute, inputByteLimit(relative));
    if (bytes.length !== expected.bytes || sha256(bytes) !== expected.sha256)
      throw new Error('Frozen source bytes/hash mismatch: ' + relative);
    check(absolute, relative);
    return encoding ? bytes.toString(encoding) : bytes;
  };
}

/** @param {string} relative */
function inputByteLimit(relative) {
  // Retained opaque source-evidence archives use the same bounded read policy
  // as official release archives; other metadata keeps the ordinary file bound.
  const archive = /^\.foundation\/releases\/(?<release>(?:pantheon-)?base-v\d+\.\d+\.\d+)\/(?:repo\.tar|foundation-release-\k<release>\.tgz)$/u.test(relative) ||
    /^\.foundation\/releases\/foundation-release-(?:pantheon-)?base-v\d+\.\d+\.\d+\.tgz$/u.test(relative);
  const evidenceArchive = /^\.harness\/evidence\/[a-z0-9][a-z0-9-]*\/verification\/(?:[a-z0-9][a-z0-9._-]*\/)+exact-source-helper-inputs\.tar\.gz$/u.test(relative);
  return archive || evidenceArchive ? LIMITS.archiveBytes : LIMITS.fileBytes;
}

export function captureInputs(root, paths) {
  assertRealPath(root, { directory: true });
  const files = new Map();
  const stamps = new Map();
  const visit = (relative) => {
    assertRelativeInput(relative);
    const absolute = path.join(root, relative);
    const stat = fs.lstatSync(absolute);
    assertRealPath(absolute, { directory: stat.isDirectory() });
    stamps.set(relative, stamp(stat));
    if (stat.isDirectory()) {
      for (const name of fs.readdirSync(absolute).sort())
        visit(`${relative}/${name}`);
    } else {
      const captured = hashStableInput(absolute, inputByteLimit(relative));
      files.set(relative, {
        path: relative,
        ...captured,
      });
      if (files.size > METADATA_LIMITS.files)
        throw new Error("Declared metadata input count exceeds limit");
    }
  };
  for (const relative of paths) visit(relative);
  const sort = ([a], [b]) => compare(a, b);
  return {
    files: [...files.entries()].sort(sort).map(([, file]) => file),
    stamps: [...stamps.entries()].sort(sort),
  };
}

function hashStableInput(absolute, maxBytes) {
  assertRealPath(absolute);
  const before = fs.lstatSync(absolute, { bigint: true });
  const identity = (stat) => [stat.dev, stat.ino, stat.size, stat.mtimeNs,
    stat.ctimeNs, stat.mode, stat.nlink].map(String);
  if (!before.isFile() || before.isSymbolicLink() || before.nlink !== 1n ||
      before.ino <= 0n || before.size > BigInt(maxBytes))
    throw new Error("Candidate input must be a bounded regular file without hard links");
  const fd = fs.openSync(absolute, fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW ?? 0));
  try {
    if (!same(identity(before), identity(fs.fstatSync(fd, { bigint: true }))))
      throw new Error("Candidate input changed while opening");
    const bytes = Number(before.size);
    const chunk = Buffer.alloc(Math.min(1024 * 1024, bytes || 1));
    const hash = crypto.createHash("sha256");
    let offset = 0;
    while (offset < bytes) {
      const count = fs.readSync(fd, chunk, 0, Math.min(chunk.length, bytes - offset), offset);
      if (!count) throw new Error("Candidate input truncated while reading");
      hash.update(chunk.subarray(0, count));
      offset += count;
    }
    if (!same(identity(before), identity(fs.fstatSync(fd, { bigint: true }))))
      throw new Error("Candidate input changed while reading");
    assertRealPath(absolute);
    if (!same(identity(before), identity(fs.lstatSync(absolute, { bigint: true }))))
      throw new Error("Candidate input path changed while reading");
    return { bytes, sha256: hash.digest("hex") };
  } finally {
    fs.closeSync(fd);
  }
}

function assertExports(entries, requiredExports) {
  const sources = new Map(
    entries.map((entry) => [entry.path, entry.bytes.toString("utf8")]),
  );
  for (const entry of requiredExports) {
    const source = sources
      .get(entry.path)
      ?.replace(
        /\/\*[\s\S]*?\*\/|\/\/[^\n]*|`[^`]*`|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'/gu,
        (match) => match.replace(/[^\n]/gu, " "),
      );
    if (
      !source ||
      !new RegExp(
        `^(?:func\\s+${entry.name}\\s*\\(|type\\s+${entry.name}\\s+)`,
        "mu",
      ).test(source)
    )
      throw new Error("Missing required API export");
  }
}

export function resolveCandidateArtifact({
  artifactPath,
  descriptorPath,
  expectedDescriptorSha256,
  cacheRoot,
  retainedSnapshotRoot,
} = {}) {
  const inputStamps = [descriptorPath, artifactPath].map((file) => {
    assertRealPath(file);
    return stamp(fs.lstatSync(file));
  });
  const descriptorBytes = readStableFile(
    descriptorPath,
    LIMITS.descriptorBytes,
  );
  if (
    !digest.test(expectedDescriptorSha256) ||
    sha256(descriptorBytes) !== expectedDescriptorSha256
  )
    throw new Error("Pinned candidate descriptor digest mismatch");
  const descriptor = freeze(
    validateCandidateDescriptor(
      JSON.parse(
        new TextDecoder("utf-8", { fatal: true }).decode(descriptorBytes),
      ),
    ),
  );
  if (!cacheRoot) throw new Error("A bounded candidate cacheRoot is required");
  const root = path.resolve(cacheRoot);
  if (!isCandidateStageRoot(root, descriptor.candidateIdentity))
    throw new Error(
      "Candidate cache must be .tmp/foundation-candidates/<candidateId>",
    );
  assertRealPath(root, { directory: true, missing: true });
  if (fs.existsSync(root))
    throw new Error(
      "Candidate snapshot identity already exists; use a fresh verified snapshot",
    );
  const sources = [path.resolve(artifactPath), path.resolve(descriptorPath)];
  if (
    sources.some(
      (source) => source === root || source.startsWith(root + path.sep),
    )
  )
    throw new Error("Candidate cache overlaps inputs");
  const entries = verifyCandidateArchive(
    readStableFile(artifactPath, LIMITS.archiveBytes),
    descriptor,
  );
  assertExports(entries, descriptor.requiredExports);
  if (retainedSnapshotRoot) {
    const expected = path.join(path.dirname(path.dirname(path.dirname(root))),
      "foundation-candidates", descriptor.candidateIdentity, "snapshot");
    if (path.basename(path.dirname(path.dirname(root))) !== "foundation-candidate-attempts" ||
        path.resolve(retainedSnapshotRoot) !== expected)
      throw new Error("Retained candidate snapshot must be the original immutable identity snapshot");
    assertRealPath(expected, { directory: true });
  }
  fs.mkdirSync(path.dirname(root), { recursive: true });
  assertRealPath(path.dirname(root), { directory: true });
  const identities = new Map(
    directoryChain(path.dirname(root)).map((directory) => [
      directory,
      directoryIdentity(directory),
    ]),
  );
  // Keep unknown identity failures fail-closed even if creation already happened.
  const stage = { directories: identities, files: new Map(), unprovenDirectories: new Set() };
  ownedStages.set(root, stage);
  const baseRoot = retainedSnapshotRoot ? path.resolve(retainedSnapshotRoot) : path.join(root, "snapshot");
  // Existing source directories are identity pins for reads, never deletion ownership.
  const sourceIdentities = retainedSnapshotRoot ? new Map() : identities;
  try {
    createStageDirectory(root, stage);
    if (retainedSnapshotRoot) {
      for (const directory of directoryChain(baseRoot))
        sourceIdentities.set(directory, directoryIdentity(directory));
      const visit = (directory) => {
        for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
          if (entry.isDirectory()) {
            const child = path.join(directory, entry.name);
            sourceIdentities.set(child, directoryIdentity(child));
            visit(child);
          }
        }
      };
      visit(baseRoot);
    } else {
      createStageDirectory(baseRoot, stage);
    }
    for (const entry of retainedSnapshotRoot ? [] : entries) {
      const target = path.join(baseRoot, entry.path);
      let directory = baseRoot;
      for (const part of entry.path.split("/").slice(0, -1)) {
        directory = path.join(directory, part);
        createStageDirectory(directory, stage);
      }
      assertRealPath(target, { missing: true });
      writeStageFile(target, entry.bytes, { flag: "wx", mode: 0o644 }, stage);
    }
    const verifyDirectoryIdentities = () => {
      assertOwnedChain(root, identities);
      assertOwnedChain(baseRoot, sourceIdentities);
      for (const directory of sourceIdentities.keys()) {
        if (directory.startsWith(baseRoot + path.sep))
          assertOwnedChain(directory, sourceIdentities);
      }
    };
    verifyDirectoryIdentities();
    const paths = fs.readdirSync(baseRoot).sort();
    const captured = captureInputs(baseRoot, paths);
    const rootStamp = stamp(fs.lstatSync(baseRoot));
    if (!same(captured.files, descriptor.files))
      throw new Error("Extracted candidate inventory mismatch");
    const verify = () => {
      verifyDirectoryIdentities();
      const current = captureInputs(baseRoot, fs.readdirSync(baseRoot).sort());
      if (
        !same(current, captured) ||
        !same(rootStamp, stamp(fs.lstatSync(baseRoot)))
      )
        throw new Error("Candidate snapshot changed during assembly");
      if (
        !same(
          inputStamps,
          [descriptorPath, artifactPath].map((file) =>
            stamp(fs.lstatSync(file)),
          ),
        ) ||
        sha256(readStableFile(descriptorPath, LIMITS.descriptorBytes)) !==
          expectedDescriptorSha256 ||
        sha256(readStableFile(artifactPath, LIMITS.archiveBytes)) !==
          descriptor.artifact.sha256
      )
        throw new Error("Candidate artifact inputs changed during assembly");
      return true;
    };
    verify();
    return {
      baseRoot,
      baseCommit: descriptor.baseCommit,
      manifest: descriptor,
      descriptor,
      candidateIdentity: descriptor.candidateIdentity,
      descriptorSha256: expectedDescriptorSha256,
      cacheRoot: root,
      verify,
    };
  } catch (error) {
    removeCandidateStage(root, { cause: error });
    throw error;
  }
}

function isCandidateStageRoot(root, identity) {
  const candidate = /^candidate-[a-f0-9]{64}$/u;
  return (candidate.test(path.basename(root)) && (!identity || path.basename(root) === identity) &&
    path.basename(path.dirname(root)) === "foundation-candidates" &&
    path.basename(path.dirname(path.dirname(root))) === ".tmp") ||
    (/^[a-z0-9][a-z0-9-]{0,63}$/u.test(path.basename(root)) &&
      candidate.test(path.basename(path.dirname(root))) && (!identity || path.basename(path.dirname(root)) === identity) &&
      path.basename(path.dirname(path.dirname(root))) === "foundation-candidate-attempts" &&
      path.basename(path.dirname(path.dirname(path.dirname(root)))) === ".tmp");
}

function preflightStageDirectory(root, stage) {
  const identities = stage.directories;
  const seenDirectories = new Set(), seenFiles = new Set();
  const visit = (directory) => {
    assertOwnedChain(directory, identities);
    seenDirectories.add(directory);
    for (const name of fs.readdirSync(directory)) {
      assertOwnedChain(directory, identities);
      const absolute = path.join(directory, name);
      const stat = fs.lstatSync(absolute);
      if (stat.isDirectory() && !stat.isSymbolicLink()) visit(absolute);
      else if (stat.isFile() && !stat.isSymbolicLink()) {
        assertFileIdentity(absolute, stage);
        seenFiles.add(absolute);
      } else throw new Error(`Unsupported cleanup entry: ${absolute}`);
    }
    assertOwnedChain(directory, identities);
  };
  visit(root);
  const within = file => file === root || file.startsWith(root + path.sep);
  for (const directory of identities.keys())
    if (within(directory) && !seenDirectories.has(directory))
      throw new Error(`Missing observed directory identity: ${directory}`);
  for (const file of stage.files.keys())
    if (within(file) && !seenFiles.has(file))
      throw new Error(`Missing owned regular file identity: ${file}`);
}

/**
 * @param {Array<{root: string, captured: Parameters<typeof frozenInputReader>[1]}>} sourceInputs
 * @param {() => unknown} [verifySources]
 */
export function candidateAssemblyFileSystem(root, sourceInputs = [], verifySources) {
  const stage = ownedStages.get(root);
  if (!stage || !isCandidateStageRoot(root)) throw new Error("Unknown candidate stage ownership");
  const identities = stage.directories;
  const assembled = path.join(root, "assembled");
  const sourceReaders = sourceInputs.map(input => ({
    root: path.resolve(input.root), read: frozenInputReader(input.root, input.captured),
  }));
  /** @param {string} file @param {BufferEncoding} [encoding] */
  const readSource = (file, encoding) => {
    const absolute = path.resolve(file);
    // Base snapshot is nested within Ops staging; use its descriptor reader first.
    const source = sourceReaders.find(input => absolute.startsWith(input.root + path.sep));
    if (!source) throw new Error("Unselected frozen source input: " + file);
    try {
      return source.read(absolute, encoding);
    } catch (error) {
      // Preserve the existing source-change error contract on actual mutation.
      // Unchanged captures leave the original returned-byte failure intact.
      verifySources?.();
      throw error;
    }
  };
  const bounded = (file) => {
    const absolute = path.resolve(file);
    if (absolute !== assembled && !absolute.startsWith(assembled + path.sep))
      throw new Error("Candidate assembler directory mutation outside owned target");
    return absolute;
  };
  return {
    // Expose only assembler reads and guarded mutations, not native path-based
    // open/append/copy/rename/unlink methods that could bypass this lifecycle.
    existsSync: fs.existsSync,
    /** @param {string} file @param {BufferEncoding} [encoding] */
    readFileSync(file, encoding) {
      const absolute = path.resolve(file);
      // Derived target reads are outside copied inputs. All source reads bind
      // the returned bytes to the descriptor or selected Ops capture.
      if (absolute.startsWith(assembled + path.sep))
        return encoding ? fs.readFileSync(absolute, encoding) : fs.readFileSync(absolute);
      return readSource(absolute, encoding);
    },
    lstatSync: fs.lstatSync,
    readdirSync: fs.readdirSync,
    assertTargetStable() {
      preflightStageDirectory(assembled, stage);
    },
    copyFileSync(source, file) {
      const absolute = bounded(file);
      assertOwnedChain(path.dirname(absolute), identities);
      writeStageFile(absolute, readSource(source), undefined, stage);
    },
    writeFileSync(file, bytes, options) {
      writeStageFile(bounded(file), bytes, options, stage);
    },
    mkdirSync(file) {
      const absolute = bounded(file);
      let current = root;
      assertOwnedChain(current, identities);
      for (const part of path.relative(root, absolute).split(path.sep)) {
        current = path.join(current, part);
        createStageDirectory(current, stage);
      }
    },
    rmSync(file) {
      const absolute = bounded(file);
      // A pre-copy exclusion can have a missing parent. Validate every observed
      // ancestor first, and never accept an existing unregistered directory.
      for (const directory of directoryChain(path.dirname(absolute))) {
        if (!identities.has(directory)) {
          try { fs.lstatSync(directory); }
          catch (error) { if (error.code === "ENOENT") return; throw error; }
        }
        assertDirectoryIdentity(directory, identities);
      }
      let stat;
      try { stat = fs.lstatSync(absolute); }
      catch (error) { if (error.code === "ENOENT") return; throw error; }
      if (stat.isFile() && !stat.isSymbolicLink()) {
        // Exclusions run before copying. No pathname deletion is necessary;
        // retain present files rather than expose a final unlink race.
        preflightStageDirectory(root, stage);
        assertFileIdentity(absolute, stage);
        throw new Error(`File removal unsupported; cleanup deferred: ${absolute}`);
      }
      preflightStageDirectory(absolute, stage);
      throw new Error(`Unproven directory creation; cleanup deferred: ${absolute}`);
    },
    runGo(args, options) {
      // Go rewrites modfiles by rename and offers no authenticated target-fd
      // interface. Keep those unproven outputs outside assembled; never enroll
      // them by scanning or delete their workspace. The final target receipt
      // validates only bytes imported through our existing exclusive file fds.
      preflightStageDirectory(assembled, stage);
      const work = path.join(root, `go-work-${crypto.randomUUID()}`);
      createStageDirectory(work, stage);
      const modfile = path.join(work, "go.mod");
      const sumfile = path.join(work, "go.sum");
      const goMod = bounded(path.join(options.cwd, "go.mod"));
      const goSum = bounded(path.join(options.cwd, "go.sum"));
      writeStageFile(modfile, readStableFile(goMod), { flag: "wx" }, stage);
      writeStageFile(sumfile, readStableFile(goSum), { flag: "wx" }, stage);
      assertOwnedChain(work, identities);
      preflightStageDirectory(assembled, stage);
      const command = [...args];
      command.splice(args[0] === "mod" ? 2 : 1, 0, `-modfile=${modfile}`);
      // Go may replace these exact two files. Their creation receipts cease to
      // authenticate the outputs. They are retained, never cleanup-authorized.
      stage.files.delete(modfile);
      stage.files.delete(sumfile);
      let result, primaryFailure;
      try { result = spawnSync("go", command, options); }
      catch (error) { primaryFailure = error; throw error; }
      finally {
        try {
          assertOwnedChain(work, identities);
          preflightStageDirectory(assembled, stage);
        } catch (identityError) {
          if (primaryFailure) {
            // The caller's exact primary object/cause must survive the finally
            // validation. removeCandidateStage records the identity refusal.
          } else if (result && result.status !== 0) {
            // Do not replace a real subprocess failure with the later refusal.
            const failure = result.error ?? new Error(result.stderr?.trim() || "Go command failed");
            failure.targetIdentityError = identityError;
            throw failure;
          } else throw identityError;
        }
      }
      if (result.status === 0) {
        const mod = readStableFile(modfile), sum = readStableFile(sumfile);
        assertOwnedChain(work, identities);
        writeStageFile(goMod, mod, undefined, stage);
        writeStageFile(goSum, sum, undefined, stage);
        preflightStageDirectory(assembled, stage);
      }
      return result;
    },
  };
}

export function removeCandidateStage(root, { cause } = {}) {
  // Never delete through a changed junction or remove arbitrary caller paths.
  if (
    !ownedStages.has(root) ||
    !isCandidateStageRoot(root)
  )
    throw new Error("Unsafe candidate cleanup target");
  const stage = ownedStages.get(root);
  try {
    preflightStageDirectory(root, stage);
    throw new Error("Unproven directory creation; cleanup deferred without deletion (Node mkdir has no authenticated creation handle)");
  } catch (error) {
    const refused = new CandidateCleanupError(root, error.message, cause ?? error);
    refused.cleanupStatus = "deferred-unproven";
    refused.retainedDirectories = [...stage.unprovenDirectories];
    throw refused;
  }
}
