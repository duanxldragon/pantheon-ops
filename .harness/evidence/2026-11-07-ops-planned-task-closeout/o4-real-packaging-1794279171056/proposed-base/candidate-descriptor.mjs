import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

export const LIMITS = Object.freeze({ files: 50000, fileBytes: 32 * 1024 * 1024, archiveBytes: 512 * 1024 * 1024, descriptorBytes: 16 * 1024 * 1024 });
export const sha256 = (bytes) => crypto.createHash('sha256').update(bytes).digest('hex');
export const aggregateSha256 = (files) => sha256(Buffer.from(JSON.stringify(files)));
const digest = /^[a-f0-9]{64}$/u;
const compare = (a, b) => a < b ? -1 : a > b ? 1 : 0;

// Capabilities describe reviewed source interfaces, not runtime acceptance.
export const CAPABILITY_EXPORTS = Object.freeze({
  'business-tenant-v2': [
    ['backend/pkg/contracts/business_tenant_groups.go', 'BusinessTenantGroupOptions'],
    ['backend/pkg/contracts/business_tenant_groups.go', 'TenantScopedBusinessGroup'],
    ['backend/pkg/contracts/business_tenant_groups.go', 'RegisterBusinessTenantModeReader'],
    ['backend/pkg/contracts/business_tenant_groups.go', 'RegisterBusinessAuthorizationReader'],
    ['backend/pkg/tenant/mode_reader.go', 'ModeReader'],
    ['backend/pkg/contracts/businessauth/businessauth.go', 'Reader'],
    ['backend/internal/middleware/business_token_middleware.go', 'BusinessTokenAuthMiddleware'],
  ],
  'required-business-bootstrap-v1': [['backend/pkg/contracts/module.go', 'RegisterRequiredBackendModules']],
  'audit-metadata-only-v1': [['backend/pkg/auditpolicy/policy.go', 'RegisterMetadataOnly'], ['backend/pkg/auditpolicy/policy.go', 'Seal'], ['backend/pkg/auditpolicy/policy.go', 'SetMetadata']],
  'permission-route-registration-v1': [['backend/pkg/permissionroutes/registry.go', 'Register']],
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
  if (typeof value !== 'string' || !value || value.length > 240 || value !== value.normalize('NFC') ||
      /[\\:<>"|?*\x00-\x1f\x7f]/u.test(value) || value.startsWith('/') ||
      value.split('/').some((part) => !part || part === '.' || part === '..' || /[. ]$/u.test(part) ||
        /^(?:con|prn|aux|nul|com[0-9¹²³]|lpt[0-9¹²³])(?:\.|$)/iu.test(part))) {
    throw new Error(`Unsafe candidate path: ${value}`);
  }
  if (PUBLIC_REVIEW_ARCHIVE_PATHS.includes(value)) return value;
  const parts = value.toLowerCase().split('/');
  if (parts.some((part) => ['.git', '.omx', '.harness', '.aws', '.ssh', '.codex', '.claude', '.tmp', 'node_modules', 'dist', 'build', 'coverage', 'uploads', 'cache', 'caches'].includes(part) ||
      part.startsWith('.env') || /^(?:credentials|secrets?)(?:\.|$)/u.test(part) || /\.(?:pem|key|p12|pfx)$/u.test(part))) {
    throw new Error(`Excluded candidate path: ${value}`);
  }
  return value;
}

export function assertRealPath(absolute, { directory = false, missing = false } = {}) {
  const resolved = path.resolve(absolute);
  let current = path.parse(resolved).root;
  for (const part of resolved.slice(current.length).split(path.sep).filter(Boolean)) {
    current = path.join(current, part);
    if (!fs.existsSync(current)) {
      // existsSync follows links; lstat also detects dangling links.
      try { fs.lstatSync(current); } catch (error) { if (error.code === 'ENOENT' && missing) return resolved; throw error; }
    }
    const stat = fs.lstatSync(current);
    if (stat.isSymbolicLink() || (!stat.isDirectory() && current !== resolved)) throw new Error('Candidate paths must use real directories and regular files');
    if (current === resolved && (directory ? !stat.isDirectory() : !stat.isFile())) throw new Error('Candidate path has unsupported file type');
  }
  return resolved;
}

export function assertInventory(files) {
  if (!Array.isArray(files) || !files.length || files.length > LIMITS.files) throw new Error('Invalid candidate file count');
  const seen = new Set();
  const directories = new Set();
  const spellings = new Map();
  let previous = '';
  let total = 0;
  for (const file of files) {
    assertCandidatePath(file.path);
    if (Object.keys(file).sort().join(',') !== 'bytes,path,sha256' || !Number.isSafeInteger(file.bytes) || file.bytes < 0 || file.bytes > LIMITS.fileBytes || !digest.test(file.sha256)) throw new Error('Invalid candidate file metadata');
    if (compare(previous, file.path) >= 0) throw new Error('Candidate inventory must be sorted and unique');
    previous = file.path;
    const folded = file.path.toLowerCase();
    if (seen.has(folded) || directories.has(folded)) throw new Error('Candidate path collision');
    const parts = folded.split('/');
    const originalParts = file.path.split('/');
    for (let i = 1; i <= parts.length; i++) {
      const foldedPath = parts.slice(0, i).join('/');
      const spelling = originalParts.slice(0, i).join('/');
      if (spellings.has(foldedPath) && spellings.get(foldedPath) !== spelling) throw new Error('Candidate case-fold collision');
      spellings.set(foldedPath, spelling);
    }
    for (let i = 1; i < parts.length; i++) {
      const parent = parts.slice(0, i).join('/');
      if (seen.has(parent)) throw new Error('Candidate file/directory collision');
      directories.add(parent);
    }
    seen.add(folded);
    total += file.bytes + 512 + (512 - file.bytes % 512) % 512;
  }
  if (total + 1024 > LIMITS.archiveBytes) throw new Error('Candidate archive size exceeds limit');
  return files;
}

function ids(values, label) {
  if (!Array.isArray(values) || !values.length || values.length > 100 || new Set(values).size !== values.length || values.some((id) => typeof id !== 'string' || !/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$/u.test(id))) throw new Error(`Invalid ${label}`);
}

export function candidateExports(capabilities, files) {
  ids(capabilities, 'capabilities');
  const inventory = new Map(files.map((file) => [file.path, file]));
  const exports = new Map();
  for (const capability of capabilities) {
    if (!Object.hasOwn(CAPABILITY_EXPORTS, capability)) throw new Error(`Unknown candidate capability: ${capability}`);
    for (const [file, name] of CAPABILITY_EXPORTS[capability]) {
      if (!inventory.has(file)) throw new Error(`Missing required API: ${file}#${name}`);
      exports.set(`${file}#${name}`, { path: file, name, sha256: inventory.get(file).sha256 });
    }
  }
  return [...exports.values()].sort((a, b) => compare(`${a.path}#${a.name}`, `${b.path}#${b.name}`));
}

export function validateCandidateDescriptor(descriptor) {
  if (!descriptor || Object.keys(descriptor).sort().join(',') !== 'aggregateSha256,artifact,baseCommit,candidateIdentity,capabilities,consumerMode,evidenceIds,files,kind,requiredExports,reviewIds,schemaVersion,sharedPaths,sourceRepo' ||
      descriptor.schemaVersion !== 1 || descriptor.kind !== 'pantheon-base-candidate' || descriptor.consumerMode !== 'candidate-staging' || descriptor.sourceRepo !== 'pantheon-base' || !/^[a-f0-9]{40}$/u.test(descriptor.baseCommit)) throw new Error('Invalid candidate descriptor schema/provenance');
  assertInventory(descriptor.files);
  const aggregate = aggregateSha256(descriptor.files);
  if (descriptor.aggregateSha256 !== aggregate || descriptor.candidateIdentity !== `candidate-${aggregate}`) throw new Error('Candidate aggregate/identity mismatch');
  if (Object.keys(descriptor.artifact ?? {}).sort().join(',') !== 'assetName,sha256' || descriptor.artifact.assetName !== `${descriptor.candidateIdentity}.tar` || !digest.test(descriptor.artifact.sha256)) throw new Error('Invalid candidate archive identity');
  ids(descriptor.evidenceIds, 'evidence IDs'); ids(descriptor.reviewIds, 'review IDs');
  const expectedExports = candidateExports(descriptor.capabilities, descriptor.files);
  if (JSON.stringify(expectedExports) !== JSON.stringify(descriptor.requiredExports)) throw new Error('Candidate capability/export mismatch');
  const shared = descriptor.sharedPaths;
  if (!shared || Array.isArray(shared) || !Object.keys(shared).length || Object.keys(shared).some((group) => !['backend', 'frontend', 'docs'].includes(group))) throw new Error('Invalid candidate sharedPaths');
  for (const entries of Object.values(shared)) {
    if (!Array.isArray(entries) || !entries.length || new Set(entries).size !== entries.length) throw new Error('Empty/duplicate candidate shared paths');
    for (const entry of entries) {
      assertCandidatePath(entry);
      if (!descriptor.files.some((file) => file.path === entry || file.path.startsWith(`${entry}/`))) throw new Error(`Missing shared inventory: ${entry}`);
    }
  }
  return descriptor;
}

export function readCandidateDescriptor(descriptorPath, expectedDescriptorSha256) {
  if (!digest.test(expectedDescriptorSha256)) throw new Error('Pinned descriptor SHA256 is required');
  assertRealPath(descriptorPath);
  if (fs.statSync(descriptorPath).size > LIMITS.descriptorBytes) throw new Error('Candidate descriptor size exceeds limit');
  const bytes = fs.readFileSync(descriptorPath);
  if (sha256(bytes) !== expectedDescriptorSha256) throw new Error('Candidate descriptor digest mismatch');
  return validateCandidateDescriptor(JSON.parse(bytes.toString('utf8')));
}

// Only regular ustar entries are emitted/accepted. No native extractor ever
// sees unchecked headers, including PAX/GNU extensions or hard/symbolic links.
export function encodeCandidateArchive(entries) {
  assertInventory(entries.map(({ path: file, bytes }) => ({ path: file, bytes: bytes.length, sha256: sha256(bytes) })));
  const chunks = [];
  for (const { path: file, bytes } of entries) {
    const header = Buffer.alloc(512);
    let name = file; let prefix = '';
    if (Buffer.byteLength(name) > 100) {
      const splits = [...file.matchAll(/\//gu)].map((match) => match.index).reverse();
      const split = splits.find((index) => Buffer.byteLength(file.slice(0, index)) <= 155 && Buffer.byteLength(file.slice(index + 1)) <= 100);
      if (split === undefined) throw new Error(`Candidate path does not fit ustar: ${file}`);
      prefix = file.slice(0, split); name = file.slice(split + 1);
    }
    header.write(name, 0, 100, 'utf8');
    const octal = (value, offset, length) => header.write(`${value.toString(8).padStart(length - 1, '0')}\0`, offset, length, 'ascii');
    octal(0o644, 100, 8); octal(0, 108, 8); octal(0, 116, 8); octal(bytes.length, 124, 12); octal(0, 136, 12);
    header.fill(32, 148, 156); header[156] = 48;
    header.write('ustar\0', 257, 6, 'ascii'); header.write('00', 263, 2, 'ascii'); header.write(prefix, 345, 155, 'utf8');
    header.write(`${[...header].reduce((sum, byte) => sum + byte, 0).toString(8).padStart(6, '0')}\0 `, 148, 8, 'ascii');
    chunks.push(header, bytes, Buffer.alloc((512 - bytes.length % 512) % 512));
  }
  chunks.push(Buffer.alloc(1024));
  return Buffer.concat(chunks);
}

export function verifyCandidateArchive(archive, descriptor) {
  validateCandidateDescriptor(descriptor);
  if (archive.length > LIMITS.archiveBytes || archive.length % 512 || sha256(archive) !== descriptor.artifact.sha256) throw new Error('Candidate archive digest/size mismatch');
  const entries = [];
  let offset = 0;
  while (offset + 512 <= archive.length) {
    const header = archive.subarray(offset, offset + 512);
    if (header.every((byte) => byte === 0)) {
      if (archive.length - offset < 1024 || !archive.subarray(offset).every((byte) => byte === 0)) throw new Error('Invalid candidate tar trailer');
      break;
    }
    const field = (start, length) => {
      const raw = header.subarray(start, start + length);
      const nul = raw.indexOf(0);
      if (nul >= 0 && raw.subarray(nul).some((byte) => byte !== 0)) throw new Error('Invalid candidate tar field');
      const value = raw.subarray(0, nul < 0 ? raw.length : nul);
      const decoded = new TextDecoder('utf-8', { fatal: true }).decode(value);
      return decoded;
    };
    const numeric = (start, length) => {
      const value = header.subarray(start, start + length).toString('ascii').replace(/[\0 ]+$/u, '');
      if (!/^[0-7]+$/u.test(value)) throw new Error('Unsupported candidate tar number');
      return parseInt(value, 8);
    };
    const checksum = numeric(148, 8);
    const sum = [...header].reduce((total, byte, index) => total + (index >= 148 && index < 156 ? 32 : byte), 0);
    if (checksum !== sum || field(257, 6) !== 'ustar' || field(263, 2) !== '00' || ![0, 48].includes(header[156]) || field(157, 100)) throw new Error('Unsupported candidate tar header/type/link');
    const prefix = field(345, 155);
    const file = assertCandidatePath(`${prefix ? `${prefix}/` : ''}${field(0, 100)}`);
    const size = numeric(124, 12);
    const end = offset + 512 + size;
    const next = end + (512 - size % 512) % 512;
    if (size > LIMITS.fileBytes || next > archive.length || archive.subarray(end, next).some((byte) => byte !== 0)) throw new Error('Invalid candidate tar content/padding');
    entries.push({ path: file, bytes: archive.subarray(offset + 512, end) });
    if (entries.length > LIMITS.files) throw new Error('Candidate archive entry count exceeds limit');
    offset = next;
  }
  if (offset + 1024 > archive.length) throw new Error('Missing candidate tar trailer');
  const files = entries.map((entry) => ({ path: entry.path, bytes: entry.bytes.length, sha256: sha256(entry.bytes) }));
  assertInventory(files);
  if (JSON.stringify(files) !== JSON.stringify(descriptor.files)) throw new Error('Candidate archive inventory mismatch');
  assertPublicReviewPackaging(entries);
  return entries;
}

export function verifyCandidateExecutable({ descriptorPath, expectedDescriptorSha256, buildReceiptPath, expectedBuildReceiptSha256, executablePath, expectedExecutableSha256 }) {
  const descriptor = readCandidateDescriptor(descriptorPath, expectedDescriptorSha256);
  if (!digest.test(expectedBuildReceiptSha256) || !digest.test(expectedExecutableSha256)) throw new Error('Pinned build/executable digests are required');
  assertRealPath(buildReceiptPath); assertRealPath(executablePath);
  if (fs.statSync(buildReceiptPath).size > LIMITS.descriptorBytes || fs.statSync(executablePath).size > LIMITS.archiveBytes) throw new Error('Candidate build/executable size exceeds limit');
  const bytes = fs.readFileSync(buildReceiptPath);
  if (bytes.length > LIMITS.descriptorBytes || sha256(bytes) !== expectedBuildReceiptSha256) throw new Error('Candidate build receipt digest mismatch');
  const receipt = JSON.parse(bytes.toString('utf8'));
  if (Object.keys(receipt).sort().join(',') !== 'aggregateSha256,candidateIdentity,capabilities,descriptorSha256,evidenceIds,executable,kind,reviewIds,schemaVersion' || receipt.schemaVersion !== 1 || receipt.kind !== 'pantheon-candidate-build' || receipt.candidateIdentity !== descriptor.candidateIdentity || receipt.descriptorSha256 !== expectedDescriptorSha256 || receipt.aggregateSha256 !== descriptor.aggregateSha256 || JSON.stringify(receipt.capabilities) !== JSON.stringify(descriptor.capabilities)) throw new Error('Candidate build identity/capability mismatch');
  ids(receipt.evidenceIds, 'build evidence IDs'); ids(receipt.reviewIds, 'build review IDs');
  const executable = fs.readFileSync(executablePath);
  if (receipt.executable?.sha256 !== expectedExecutableSha256 || sha256(executable) !== expectedExecutableSha256 || receipt.executable.bytes !== executable.length) throw new Error('Candidate executable hash/size mismatch');
  return receipt;
}
