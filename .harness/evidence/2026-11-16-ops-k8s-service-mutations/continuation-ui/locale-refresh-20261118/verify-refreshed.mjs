import { createHash } from 'node:crypto';
import { execFile } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { promisify } from 'node:util';

const run = promisify(execFile);
const root = process.cwd();
const frontend = resolve(root, 'frontend');
const receipt = JSON.parse(readFileSync(new URL('./copy-receipt.json', import.meta.url), 'utf8'));
if (receipt.status !== 'passed' || receipt.copies.length !== 5) throw new Error('No complete copy receipt');
const sourceHash = (path) => createHash('sha256').update(readFileSync(resolve(root, path))).digest('hex');
const paths = [...receipt.canonicalBefore.map((row) => row.path), ...receipt.copies.map((row) => row.path)];
const sourceHashes = () => paths.map((path) => ({ path, sha256: sourceHash(path) }));
const before = sourceHashes();
const checks = [
  ['generated-eslint', process.execPath, ['node_modules/eslint/bin/eslint.js', ...receipt.copies.map((row) => row.path.slice('frontend/'.length))], frontend],
  ['type-check', process.execPath, ['node_modules/typescript/bin/tsc', '-b'], frontend],
  ['generated-diff-check', 'git', ['diff', '--check', '--', ...receipt.copies.map((row) => row.path)], root],
];
const commands = await Promise.all(checks.map(async ([name, executable, args, cwd]) => {
  const startedUTC = new Date().toISOString();
  let exitCode = 0, stdout = '', stderr = '';
  try { ({ stdout, stderr } = await run(executable, args, { cwd, windowsHide: true, timeout: 120000, maxBuffer: 4 * 1024 * 1024 })); }
  catch (error) { exitCode = typeof error.code === 'number' ? error.code : 1; stdout = error.stdout ?? ''; stderr = error.stderr ?? error.message; }
  writeFileSync(new URL(`./${name}.log`, import.meta.url), stdout + stderr);
  return { name, executable, args, cwd, startedUTC, finishedUTC: new Date().toISOString(), exitCode, stdout, stderr };
}));
const canonical = JSON.parse(readFileSync(new URL('../canonical-zh-en.json', import.meta.url), 'utf8'));
const translations = [];
for (const locale of ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR']) {
  const resource = readFileSync(resolve(root, `frontend/src/i18n/resources/business/${locale}.ts`), 'utf8');
  const object = resource.match(/^const \w+ = (\{[\s\S]*\});\n\nexport default \w+;\n$/)?.[1];
  if (!object) throw new Error(`Unexpected generated resource syntax: ${locale}`);
  const generated = JSON.parse(object);
  const source = JSON.parse(readFileSync(resolve(root, `frontend/src/modules/business/v12locales/${locale}.json`), 'utf8'));
  for (const row of canonical) {
    if (generated[row.key] !== source[row.key]) throw new Error(`Generated translation mismatch: ${locale}:${row.key}`);
    translations.push({ locale, key: row.key, matchedCanonical: true });
  }
}
const after = sourceHashes();
const sourcesStable = JSON.stringify(before) === JSON.stringify(after);
const boundToCopyReceipt = receipt.copies.every((row) => sourceHash(row.path) === row.afterSHA256);
const result = { taskId: receipt.taskId, scope: receipt.scope, generatedUTC: new Date().toISOString(), commands, before, after, sourcesStable, boundToCopyReceipt, generatedTranslationComparisons: translations.length, translations, testsExecuted: false, renderedEvidenceInspected: false };
writeFileSync(new URL('./verification.json', import.meta.url), `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify({ commands: commands.map(({ name, exitCode }) => ({ name, exitCode })), sourcesStable, boundToCopyReceipt, generatedTranslationComparisons: translations.length }));
process.exitCode = commands.some((row) => row.exitCode !== 0) || !sourcesStable || !boundToCopyReceipt ? 1 : 0;
