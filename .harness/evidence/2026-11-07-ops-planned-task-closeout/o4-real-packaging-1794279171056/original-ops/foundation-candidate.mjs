// Self-contained schema1 consumer. Keep protocol semantics aligned with the Base
// producer; this file never loads tooling or source from a live Base checkout.
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

export const LIMITS = Object.freeze({
  files: 50000,
  fileBytes: 32 * 1024 * 1024,
  archiveBytes: 512 * 1024 * 1024,
  descriptorBytes: 16 * 1024 * 1024,
});
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

export function assertCandidatePath(value) {
  if (
    typeof value !== "string" ||
    !value ||
    value.length > 240 ||
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

function createStageDirectory(absolute, identities) {
  assertOwnedChain(path.dirname(absolute), identities);
  if (identities.has(absolute)) {
    assertDirectoryIdentity(absolute, identities);
    return;
  }
  // mkdir is exclusive; capture immediately rather than enrolling a directory
  // found later by a recursive scan. Node provides no atomic mkdir+identity API.
  fs.mkdirSync(absolute);
  identities.set(absolute, directoryIdentity(absolute));
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
    value.includes("\\") ||
    value.startsWith("/") ||
    /[:\x00-\x1f]/u.test(value) ||
    value
      .split("/")
      .some(
        (part) => !part || part === "." || part === ".." || /[. ]$/u.test(part),
      )
  )
    throw new Error("Unsafe declared Ops input path");
  return value;
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
      const bytes = readStableFile(absolute);
      files.set(relative, {
        path: relative,
        bytes: bytes.length,
        sha256: sha256(bytes),
      });
      if (files.size > LIMITS.files)
        throw new Error("Candidate input count exceeds limit");
    }
  };
  for (const relative of paths) visit(relative);
  const sort = ([a], [b]) => compare(a, b);
  return {
    files: [...files.entries()].sort(sort).map(([, file]) => file),
    stamps: [...stamps.entries()].sort(sort),
  };
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
  if (
    path.basename(root) !== descriptor.candidateIdentity ||
    path.basename(path.dirname(root)) !== "foundation-candidates" ||
    path.basename(path.dirname(path.dirname(root))) !== ".tmp"
  )
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
  fs.mkdirSync(path.dirname(root), { recursive: true });
  assertRealPath(path.dirname(root), { directory: true });
  const identities = new Map(
    directoryChain(path.dirname(root)).map((directory) => [
      directory,
      directoryIdentity(directory),
    ]),
  );
  // Keep unknown identity failures fail-closed even if creation already happened.
  ownedStages.set(root, identities);
  const baseRoot = path.join(root, "snapshot");
  try {
    createStageDirectory(root, identities);
    createStageDirectory(baseRoot, identities);
    for (const entry of entries) {
      const target = path.join(baseRoot, entry.path);
      let directory = baseRoot;
      for (const part of entry.path.split("/").slice(0, -1)) {
        directory = path.join(directory, part);
        createStageDirectory(directory, identities);
      }
      assertRealPath(target, { missing: true });
      fs.writeFileSync(target, entry.bytes, { flag: "wx", mode: 0o644 });
    }
    const verifyDirectoryIdentities = () => {
      assertOwnedChain(baseRoot, identities);
      for (const directory of identities.keys()) {
        if (directory.startsWith(baseRoot + path.sep))
          assertOwnedChain(directory, identities);
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

export function removeCandidateStage(root, { cause } = {}) {
  // Never delete through a changed junction or remove arbitrary caller paths.
  if (
    !ownedStages.has(root) ||
    !/^candidate-[a-f0-9]{64}$/u.test(path.basename(root)) ||
    path.basename(path.dirname(root)) !== "foundation-candidates" ||
    path.basename(path.dirname(path.dirname(root))) !== ".tmp"
  )
    throw new Error("Unsafe candidate cleanup target");
  const identities = ownedStages.get(root);
  try {
    const visit = (directory, remove) => {
      assertOwnedChain(directory, identities);
      for (const name of fs.readdirSync(directory)) {
        assertOwnedChain(directory, identities);
        const absolute = path.join(directory, name);
        const stat = fs.lstatSync(absolute);
        if (stat.isDirectory() && !stat.isSymbolicLink()) {
          visit(absolute, remove);
        } else if (stat.isFile()) {
          if (remove) {
            assertOwnedChain(directory, identities);
            fs.unlinkSync(absolute);
          }
        } else {
          throw new Error(`Unsupported cleanup entry: ${absolute}`);
        }
      }
      if (remove) {
        assertOwnedChain(directory, identities);
        fs.rmdirSync(directory);
      }
    };
    // Guard the complete subtree first so a known foreign directory preserves
    // the stage intact. Assembler-created directories lack a creation receipt
    // in this consumer and remain unknown; never infer ownership from location.
    visit(root, false);
    // Path checks and deletion are separate syscalls, not an atomic namespace
    // lock. Recheck every ancestor before each unlink/rmdir, and never hand a
    // recursively deleting primitive a path that could be replaced afterward.
    visit(root, true);
  } catch (error) {
    throw new CandidateCleanupError(root, error.message, cause ?? error);
  }
  ownedStages.delete(root);
}
