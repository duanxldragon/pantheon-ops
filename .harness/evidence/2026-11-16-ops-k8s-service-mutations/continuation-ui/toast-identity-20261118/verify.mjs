import { createHash } from 'node:crypto';
import { execFile } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { promisify } from 'node:util';

const run = promisify(execFile);
const root = process.cwd(), frontend = resolve(root, 'frontend');
const sources = [
  'frontend/src/modules/business/k8s/api.ts',
  'frontend/src/modules/business/k8s/service/ServiceList.tsx',
  'frontend/src/modules/business/k8s/service/ServiceEditor.tsx',
  'frontend/src/modules/business/k8s/service/serviceDraft.ts',
  'frontend/src/modules/business/k8s/service/service-editor.css',
  'frontend/tests/smoke/business/k8s/k8s-pages.spec.ts',
  'docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md',
  ...['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'].flatMap((locale) => [
    `frontend/src/modules/business/v12locales/${locale}.json`,
    `frontend/src/i18n/resources/business/${locale}.ts`,
  ]),
];
const hashes = () => sources.map((path) => ({ path, sha256: createHash('sha256').update(readFileSync(resolve(root, path))).digest('hex') }));
const before = hashes();
const checks = [
  ['eslint', process.execPath, ['node_modules/eslint/bin/eslint.js', 'src/modules/business/k8s/service/ServiceList.tsx', 'src/modules/business/k8s/service/ServiceEditor.tsx', 'src/modules/business/k8s/api.ts', 'tests/smoke/business/k8s/k8s-pages.spec.ts'], frontend],
  ['type-check', process.execPath, ['node_modules/typescript/bin/tsc', '-b'], frontend],
  ['shell-visual-contract', process.execPath, ['scripts/check-shell-visual-contract.mjs'], frontend],
  ['playwright-discovery', process.execPath, ['node_modules/@playwright/test/cli.js', 'test', 'tests/smoke/business/k8s/k8s-pages.spec.ts', '--config', 'playwright.config.ts', '--list'], frontend],
  ['diff-check', 'git', ['diff', '--check', '--', ...sources], root],
];
const commands = await Promise.all(checks.map(async ([name, executable, args, cwd]) => {
  const startedUTC = new Date().toISOString();
  let exitCode = 0, stdout = '', stderr = '';
  try { ({ stdout, stderr } = await run(executable, args, { cwd, windowsHide: true, timeout: 120000, maxBuffer: 4 * 1024 * 1024 })); }
  catch (error) { exitCode = typeof error.code === 'number' ? error.code : 1; stdout = error.stdout ?? ''; stderr = error.stderr ?? error.message; }
  writeFileSync(new URL(`./${name}.log`, import.meta.url), stdout + stderr);
  return { name, executable, args, cwd, startedUTC, finishedUTC: new Date().toISOString(), exitCode, stdout, stderr };
}));
const after = hashes();
const sourcesStable = JSON.stringify(before) === JSON.stringify(after);
const result = { taskId: '2026-11-16-ops-k8s-service-mutations', owner: 'c04_k8s_service_ui_executor', scope: 'C04-owned modal feedback and draft removal semantics; no shared Base or generated resource changes', generatedUTC: new Date().toISOString(), commands, before, after, sourcesStable, testsExecuted: false, newRenderedEvidenceInspected: false };
writeFileSync(new URL('./verification.json', import.meta.url), `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify({ checks: commands.map(({ name, exitCode }) => ({ name, exitCode })), sourceFiles: before.length, sourcesStable }));
process.exitCode = commands.some((row) => row.exitCode !== 0) || !sourcesStable ? 1 : 0;
