import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';

const root = process.cwd();
const foundationLockPath = path.join(root, 'foundation-release.lock.json');
const foundationLock = fs.existsSync(foundationLockPath)
  ? JSON.parse(fs.readFileSync(foundationLockPath, 'utf8'))
  : null;

const requiredFiles = [
  'AGENTS.md',
  'foundation-release.lock.json',
  'docs/README.md',
  'docs/README.en.md',
  'docs/PROJECT_INHERITANCE.md',
  'docs/PROJECT_INHERITANCE.en.md',
];

const requiredMarkers = {
  'AGENTS.md': [
    'pantheon-base',
    'business/*',
    '先判断是否应在 `pantheon-base` 修复，再同步到 ops',
  ],
  'foundation-release.lock.json': [
    '"consumerMode": "foundation-release-consumer"',
    '"frontend/src/store"',
  ],
  'docs/README.md': [
    'PROJECT_INHERITANCE.md',
    'TASK_PACKET_OPS_TEMPLATE.md',
  ],
  'docs/README.en.md': [
    'PROJECT_INHERITANCE.md',
    'TASK_PACKET_OPS_TEMPLATE.md',
  ],
  'docs/PROJECT_INHERITANCE.md': [
    'Base repository：当前继承源是 `../pantheon-base`',
    'Base version：当前锁定到',
    'business/cmdb',
    'business/deploy',
    '如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`',
    '这次共享改动对应的 base commit 是什么',
    '共享路径哪些已同步，哪些故意未同步',
    '是否分别验证了 base 和 ops 的最小启动、build 或 smoke',
    '`foundation-release.lock.json`',
    '`npm run check:base-sync:workspace`',
  ],
  'docs/PROJECT_INHERITANCE.en.md': [
    'Base repository: `../pantheon-base`',
    'Base version:',
    'business/cmdb',
    'business/deploy',
    'update `pantheon-base` first and then upgrade `pantheon-ops`',
    'which base commit introduced the shared change',
    'which shared paths were synced and which were intentionally left out',
    'whether base and ops each received their minimum validation pass',
    '`foundation-release.lock.json`',
    '`npm run check:base-sync:workspace`',
  ],
};

const findings = [];
const warnings = [];

for (const relativePath of requiredFiles) {
  const absolutePath = path.join(root, relativePath);
  if (!fs.existsSync(absolutePath)) {
    findings.push(`${relativePath}: required inheritance file is missing`);
    continue;
  }

  const content = fs.readFileSync(absolutePath, 'utf8');
  for (const marker of requiredMarkers[relativePath] ?? []) {
    if (!content.includes(marker)) {
      findings.push(`${relativePath}: missing required marker: ${marker}`);
    }
  }

  if (relativePath === 'foundation-release.lock.json') {
    if (!foundationLock?.releaseVersion) {
      findings.push(`${relativePath}: releaseVersion is missing`);
    }
    if (!foundationLock?.baseCommit || !/^[0-9a-f]{40}$/iu.test(foundationLock.baseCommit)) {
      findings.push(`${relativePath}: baseCommit must be a 40-char git commit`);
    }
    if (foundationLock?.releaseLine !== 'release/0.8') {
      findings.push(`${relativePath}: releaseLine must remain release/0.8`);
    }
  }

  if (relativePath === 'docs/PROJECT_INHERITANCE.md' && foundationLock) {
    const releaseMarker = `Base version：当前锁定到 \`${foundationLock.releaseVersion}\`（\`${foundationLock.baseCommit}\`）`;
    if (!content.includes(releaseMarker)) {
      findings.push(`${relativePath}: base version marker does not match foundation-release.lock.json`);
    }
  }

  if (relativePath === 'docs/PROJECT_INHERITANCE.en.md' && foundationLock) {
    const releaseMarker = `Base version: \`${foundationLock.releaseVersion}\` (\`${foundationLock.baseCommit}\`)`;
    if (!content.includes(releaseMarker)) {
      findings.push(`${relativePath}: base version marker does not match foundation-release.lock.json`);
    }
  }
}

// ── Lock staleness check (non-blocking advisory) ──────────────────────

function resolveBaseRepoRoot() {
  if (!foundationLock) return null;

  if (process.env.PANTHEON_BASE_REPO_ROOT) {
    const resolved = path.resolve(process.env.PANTHEON_BASE_REPO_ROOT);
    if (fs.existsSync(resolved)) return resolved;
  }

  const baseRelative = path.resolve(root, foundationLock.baseRepo ?? '../pantheon-base');
  if (fs.existsSync(baseRelative)) return baseRelative;

  return null;
}

function git(args, cwd) {
  const result = spawnSync('git', ['-C', cwd, ...args], { encoding: 'utf8' });
  if (result.status !== 0) return null;
  return result.stdout.trim();
}

function checkLockStaleness() {
  if (!foundationLock) return;

  const baseRepoRoot = resolveBaseRepoRoot();
  if (!baseRepoRoot) {
    // base repo is not checked out — staleness check is deferred to
    // the weekly .github/workflows/inheritance-drift-detection.yml
    return;
  }

  // Verify the locked commit still exists in the base repo
  const lockedCommit = foundationLock.baseCommit;
  const commitExists = git(['rev-parse', '--verify', `${lockedCommit}^{commit}`], baseRepoRoot);
  if (!commitExists) {
    warnings.push(
      `Locked base commit ${lockedCommit} no longer exists in pantheon-base. ` +
      `The lock file may reference a force-pushed or garbage-collected commit. ` +
      `Re-run upgrade:foundation:local-apply with a valid release version.`,
    );
    return;
  }

  // Compare lock commit against base HEAD: how many commits ahead is HEAD?
  const revList = git(['rev-list', '--count', `${lockedCommit}..HEAD`], baseRepoRoot);
  if (revList === null) return;

  const commitsAhead = parseInt(revList, 10);
  const threshold = parseInt(process.env.FOUNDATION_LOCK_STALENESS_THRESHOLD ?? '10', 10);
  if (commitsAhead > threshold) {
    warnings.push(
      `Locked base commit (${lockedCommit.slice(0, 8)}) is ${commitsAhead} commits behind ` +
      `pantheon-base HEAD (threshold: ${threshold}). Consider running:\n` +
      `  npm run check:base-sync:workspace    (preview drift)\n` +
      `  npm run upgrade:foundation:local-plan -- --release-version <version>\n` +
      `Or override the threshold: FOUNDATION_LOCK_STALENESS_THRESHOLD=20\n` +
      `A scheduled CI workflow (.github/workflows/inheritance-drift-detection.yml) ` +
      `runs weekly to detect this automatically.`,
    );
  }
}

checkLockStaleness();

// ── Report ────────────────────────────────────────────────────────────

if (findings.length > 0) {
  console.error('Pantheon Ops inheritance contract check failed');
  for (const finding of findings) {
    console.error(`- ${finding}`);
  }
}

if (warnings.length > 0) {
  console.warn('Pantheon Ops inheritance contract advisories:');
  for (const warning of warnings) {
    console.warn(`- ${warning}`);
  }
}

if (findings.length === 0) {
  if (warnings.length === 0) {
    console.log('OK pantheon-ops inheritance contract markers are present');
  } else {
    console.log('OK pantheon-ops inheritance contract markers are present (with advisories)');
  }
}

process.exit(findings.length > 0 ? 1 : 0);
