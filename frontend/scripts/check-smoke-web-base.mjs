import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');

const targetRoots = [
  'package.json',
  'playwright.config.ts',
  'playwright.full-system.config.ts',
  'playwright.auto-recycle.config.ts',
  'playwright.many-to-many.config.ts',
  'playwright.master-detail.config.ts',
  'scripts',
  'tests/smoke',
  'tmp-icon-audit.cjs',
];

const forbiddenUrlPattern = /http:\/\/127\.0\.0\.1:517[3-9]/;
const forbiddenProxyTargetPattern = /--proxy-target\s+https?:\/\/127\.0\.0\.1:\d+/;
const requiredToken = 'PANTHEON_WEB_BASE_URL';
const ignoredFileSuffixes = new Set([
  'scripts/check-smoke-web-base.mjs',
]);

function fail(message) {
  throw new Error(message);
}

function listFiles(relativePath) {
  const absolutePath = path.resolve(frontendRoot, relativePath);
  if (!fs.existsSync(absolutePath)) {
    return [];
  }
  const stat = fs.statSync(absolutePath);
  if (stat.isFile()) {
    return [relativePath.replaceAll('\\', '/')];
  }
  const results = [];
  for (const entry of fs.readdirSync(absolutePath, { withFileTypes: true })) {
    const childRelativePath = path.posix.join(relativePath.replaceAll('\\', '/'), entry.name);
    if (entry.isDirectory()) {
      results.push(...listFiles(childRelativePath));
      continue;
    }
    results.push(childRelativePath);
  }
  return results;
}

function shouldInspectFile(relativePath) {
  const normalizedPath = relativePath.replaceAll('\\', '/');
  if (ignoredFileSuffixes.has(normalizedPath)) {
    return false;
  }
  return normalizedPath.endsWith('.ts')
    || normalizedPath.endsWith('.tsx')
    || normalizedPath.endsWith('.js')
    || normalizedPath.endsWith('.mjs')
    || normalizedPath.endsWith('.cjs')
    || normalizedPath.endsWith('.json');
}

function assertPackageScripts() {
  const packageJsonPath = path.resolve(frontendRoot, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
  const scripts = packageJson.scripts ?? {};
  const findings = [];

  for (const [name, command] of Object.entries(scripts)) {
    if (!name.startsWith('test:smoke:')) {
      continue;
    }
    if (typeof command !== 'string') {
      findings.push(`${name} must be a string script command.`);
      continue;
    }
    if (!command.includes('tests/smoke/')) {
      continue;
    }
    if (command.includes('playwright.api.config.ts')) {
      continue;
    }
    if (!command.includes('scripts/run-smoke-suite.mjs')) {
      findings.push(
        `${name} must run browser smoke through scripts/run-smoke-suite.mjs so the started Vite server and Playwright baseURL stay synchronized.`,
      );
    }
    if (forbiddenProxyTargetPattern.test(command)) {
      findings.push(
        `${name} must not hard-code --proxy-target http://127.0.0.1:* in package.json. Set PANTHEON_API_PROXY_TARGET in the environment so smoke can follow the active backend instance.`,
      );
    }
  }

  if (findings.length > 0) {
    fail(`Smoke script-entry check failed.\n${findings.join('\n')}`);
  }
}

function main() {
  const files = targetRoots.flatMap((relativePath) => listFiles(relativePath)).filter(shouldInspectFile);
  const findings = [];

  assertPackageScripts();

  for (const relativePath of files) {
    const source = fs.readFileSync(path.resolve(frontendRoot, relativePath), 'utf8');
    const lines = source.split(/\r?\n/);
    lines.forEach((line, index) => {
      if (!forbiddenUrlPattern.test(line)) {
        return;
      }
      if (line.includes(requiredToken)) {
        return;
      }
      findings.push(`${relativePath}:${index + 1} contains hard-coded frontend origin ${line.trim()}`);
    });
  }

  if (findings.length > 0) {
    fail(
      `Smoke web-base check failed.\n`
      + `Use ${requiredToken} for browser smoke scripts, Playwright configs, and QA helpers instead of hard-coded 127.0.0.1:517x origins.\n`
      + findings.join('\n'),
    );
  }

  console.log(`smoke web-base check passed for ${files.length} files`);
}

main();
