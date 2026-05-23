import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');
const packageJsonPath = path.join(frontendRoot, 'package.json');
const smokeReadmePath = path.join(frontendRoot, 'tests', 'smoke', 'README.md');
const smokeRoot = path.join(frontendRoot, 'tests', 'smoke');

function fail(message) {
  throw new Error(message);
}

function walk(relativePath) {
  const absolutePath = path.join(frontendRoot, relativePath);
  const entries = fs.readdirSync(absolutePath, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const childRelativePath = path.posix.join(relativePath.replaceAll('\\', '/'), entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(childRelativePath));
      continue;
    }
    files.push(childRelativePath);
  }
  return files;
}

function collectSpecFiles() {
  return walk('tests/smoke')
    .filter((relativePath) => relativePath.endsWith('.spec.ts'))
    .filter((relativePath) => !relativePath.includes('/helpers/'))
    .sort();
}

function collectLeafSmokeScripts(scripts) {
  return Object.entries(scripts)
    .filter(([name]) => name.startsWith('test:smoke:'))
    .filter(([name]) => name !== 'test:smoke:all')
    .filter(([, command]) => typeof command === 'string' && command.includes('tests/smoke/'))
    .sort((left, right) => left[0].localeCompare(right[0]));
}

function collectSpecPathsFromCommand(command) {
  return command
    .split(/\s+/)
    .filter((token) => token.startsWith('tests/smoke/') && token.endsWith('.spec.ts'));
}

function main() {
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
  const scripts = packageJson.scripts ?? {};
  const smokeReadme = fs.readFileSync(smokeReadmePath, 'utf8');
  const specFiles = collectSpecFiles();
  const leafSmokeScripts = collectLeafSmokeScripts(scripts);
  const findings = [];

  const scriptToSpecs = new Map();
  const referencedSpecs = new Set();

  for (const [name, command] of leafSmokeScripts) {
    const specs = collectSpecPathsFromCommand(command);
    if (specs.length === 0) {
      findings.push(`${name} must reference at least one tests/smoke/*.spec.ts file.`);
      continue;
    }
    for (const spec of specs) {
      const absoluteSpecPath = path.join(frontendRoot, spec);
      if (!fs.existsSync(absoluteSpecPath)) {
        findings.push(`${name} references missing spec ${spec}.`);
        continue;
      }
      referencedSpecs.add(spec);
    }
    scriptToSpecs.set(name, specs);
  }

  for (const spec of specFiles) {
    if (!referencedSpecs.has(spec)) {
      findings.push(`Orphan smoke spec without script coverage: ${spec}.`);
    }
    if (!smokeReadme.includes(`\`${spec.replace('tests/smoke/', '')}\``)) {
      findings.push(`Smoke README coverage matrix is missing ${spec}.`);
    }
  }

  for (const [name, specs] of scriptToSpecs.entries()) {
    if (!smokeReadme.includes(`\`${name}\``)) {
      findings.push(`Smoke README must mention script ${name}.`);
    }
    for (const spec of specs) {
      const relativeSpec = spec.replace('tests/smoke/', '');
      if (!smokeReadme.includes(`\`${relativeSpec}\``)) {
        findings.push(`Smoke README must map ${name} to ${relativeSpec}.`);
      }
    }
  }

  const smokeAll = String(scripts['test:smoke:all'] || '');
  for (const requiredScript of ['test:smoke:platform', 'test:smoke:system', 'test:smoke:business']) {
    if (!smokeAll.includes(requiredScript)) {
      findings.push(`test:smoke:all must include ${requiredScript}.`);
    }
  }

  if (findings.length > 0) {
    fail(
      `Smoke coverage contract failed.\n`
      + `When routes, modules, smoke specs, or script entrypoints change, update the matching smoke command and tests/smoke/README.md in the same patch.\n`
      + findings.join('\n'),
    );
  }

  console.log(`smoke coverage contract passed for ${specFiles.length} smoke specs and ${leafSmokeScripts.length} script entries`);
}

main();
