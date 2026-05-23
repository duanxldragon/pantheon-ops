import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');
const admissionPath = path.resolve(frontendRoot, 'config/system-page-admission.json');

const governancePatterns = ['GovernanceInsightDrawer', 'GovernanceRailToggleButton'];
const heroPatterns = ['system-list__hero', 'system-page-hero'];
const validNarratives = [
  'task-first-list',
  'config-overview',
  'config-group',
  'governance-workbench',
  'audit-console',
  'module-governance',
  'module-onboarding',
];

function fail(message) {
  throw new Error(message);
}

function readAdmissionConfig() {
  return JSON.parse(fs.readFileSync(admissionPath, 'utf8'));
}

function readSourceFile(relativePath) {
  return fs.readFileSync(path.resolve(frontendRoot, relativePath), 'utf8');
}

function readProjectFile(relativePath) {
  const absolutePath = path.resolve(frontendRoot, relativePath);
  if (!fs.existsSync(absolutePath)) {
    fail(`Referenced file does not exist: ${relativePath}`);
  }
  return fs.readFileSync(absolutePath, 'utf8');
}

function assertUniquePaths(entries) {
  const seen = new Set();
  for (const entry of entries) {
    if (seen.has(entry.path)) {
      fail(`Duplicate admission path: ${entry.path}`);
    }
    seen.add(entry.path);
  }
}

function assertForbiddenPatterns(entry, source) {
  if (entry.governanceDrawer === 'forbidden') {
    for (const pattern of governancePatterns) {
      if (source.includes(pattern)) {
        fail(`${entry.path} forbids governance drawer but ${entry.sourceFile} still contains ${pattern}`);
      }
    }
  }
  if (entry.hero === 'forbidden') {
    for (const pattern of heroPatterns) {
      if (source.includes(pattern)) {
        fail(
          `${entry.path} forbids hero but ${entry.sourceFile} still contains ${pattern}; migrate to GovernanceSummaryBar instead`,
        );
      }
    }
  }
}

function assertAllowedPatterns(entry, source) {
  if (entry.governanceDrawer !== 'allowed') {
    return;
  }
  if (!entry.governanceButtonText || !entry.governanceDrawerTitle) {
    fail(`${entry.path} allows governance drawer but is missing button or drawer title metadata`);
  }
  for (const pattern of governancePatterns) {
    if (!source.includes(pattern)) {
      fail(`${entry.path} allows governance drawer but ${entry.sourceFile} is missing ${pattern}`);
    }
  }
}

function assertNarrativeContract(entry) {
  if (!validNarratives.includes(entry.narrative)) {
    fail(`${entry.path} has unsupported narrative: ${entry.narrative}`);
  }
  if (entry.narrative === 'config-overview' || entry.narrative === 'config-group') {
    if (!Array.isArray(entry.identitySelectors) || entry.identitySelectors.length === 0) {
      fail(
        `${entry.path} must declare identitySelectors so route-level UI semantics can be guarded with smoke coverage`,
      );
    }
    if (!Array.isArray(entry.requiredSmokeAssertions) || entry.requiredSmokeAssertions.length === 0) {
      fail(
        `${entry.path} must declare requiredSmokeAssertions so code and tests stay synchronized in the same change`,
      );
    }
  }
  if (entry.narrative === 'module-governance' || entry.narrative === 'module-onboarding') {
    if (entry.governanceDrawer !== 'forbidden') {
      fail(`${entry.path} is a high-sensitivity module page and must keep governanceDrawer forbidden`);
    }
    if (entry.hero !== 'forbidden') {
      fail(`${entry.path} is a high-sensitivity module page and must keep hero forbidden`);
    }
  }
}

function assertIdentitySelectors(entry, source) {
  if (!Array.isArray(entry.identitySelectors)) {
    return;
  }
  for (const selector of entry.identitySelectors) {
    if (typeof selector !== 'string' || selector.trim().length === 0) {
      fail(`${entry.path} has invalid identity selector metadata: ${JSON.stringify(selector)}`);
    }
    const token = selector.replace(/^[.#]/, '');
    if (!source.includes(token)) {
      fail(
        `${entry.path} expects identity selector ${selector} but ${entry.sourceFile} no longer contains ${token}; update the page and matching smoke tests together`,
      );
    }
  }
}

function assertRequiredSmokeAssertions(entry) {
  if (!Array.isArray(entry.requiredSmokeAssertions)) {
    return;
  }
  for (const assertion of entry.requiredSmokeAssertions) {
    if (!assertion || typeof assertion.file !== 'string' || !Array.isArray(assertion.tokens)) {
      fail(`${entry.path} has invalid requiredSmokeAssertions metadata: ${JSON.stringify(assertion)}`);
    }
    if (assertion.tokens.length === 0) {
      fail(`${entry.path} must provide at least one token for ${assertion.file}`);
    }
    const smokeSource = readProjectFile(assertion.file);
    for (const token of assertion.tokens) {
      if (typeof token !== 'string' || token.trim().length === 0) {
        fail(`${entry.path} has invalid smoke assertion token in ${assertion.file}`);
      }
      if (!smokeSource.includes(token)) {
        fail(
          `${entry.path} is missing required smoke assertion token ${JSON.stringify(token)} in ${assertion.file}; update tests/scripts in the same change as the route`,
        );
      }
    }
  }
}

function main() {
  const entries = readAdmissionConfig();
  assertUniquePaths(entries);

  for (const entry of entries) {
    if (!entry.path || !entry.title || !entry.sourceFile || !entry.narrative) {
      fail(`Admission entry is missing required fields: ${JSON.stringify(entry)}`);
    }
    if (!['allowed', 'forbidden'].includes(entry.governanceDrawer)) {
      fail(`${entry.path} has invalid governanceDrawer value: ${entry.governanceDrawer}`);
    }
    if (!['allowed', 'forbidden'].includes(entry.hero)) {
      fail(`${entry.path} has invalid hero value: ${entry.hero}`);
    }

    assertNarrativeContract(entry);
    const source = readSourceFile(entry.sourceFile);
    assertForbiddenPatterns(entry, source);
    assertAllowedPatterns(entry, source);
    assertIdentitySelectors(entry, source);
    assertRequiredSmokeAssertions(entry);
  }

  console.log(`system page admission check passed for ${entries.length} entries`);
}

main();
