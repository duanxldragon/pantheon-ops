import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const scriptPath = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../frontend/scripts/check-search-toolbar-contract.mjs',
);

const VALID_CSS = [
  '.search-toolbar .search-toolbar__keyword.arco-input-group-wrapper {',
  '  min-height: 0;',
  '  border: none;',
  '  background: transparent;',
  '}',
  '.search-toolbar .search-toolbar__keyword .arco-input-inner-wrapper {',
  '  height: var(--shell-filter-control-min-height);',
  '}',
  '',
].join('\n');

const VALID_PAGE = [
  "import { SearchToolbar } from '../../components';",
  'const Page = () => (',
  '  <SearchToolbar',
  "    keyword={query.keyword ?? ''}",
  "    keywordPlaceholder={t('system.demo.search.placeholder')}",
  '    onKeywordChange={(keyword) => search({ keyword })}',
  '    onClearAll={reset}',
  '  />',
  ');',
  'export default Page;',
  '',
].join('\n');

function runGate(fixture) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-search-toolbar-'));
  try {
    const modulesRoot = path.join(repoRoot, 'src', 'modules', 'system', 'demo');
    fs.mkdirSync(modulesRoot, { recursive: true });
    fs.writeFileSync(path.join(repoRoot, 'src', 'index.css'), fixture.css ?? VALID_CSS, 'utf8');
    fs.writeFileSync(path.join(modulesRoot, 'DemoList.tsx'), fixture.page ?? VALID_PAGE, 'utf8');
    try {
      const stdout = execFileSync(process.execPath, [scriptPath, '--root', repoRoot], {
        encoding: 'utf8',
      });
      return { code: 0, output: stdout };
    } catch (error) {
      return { code: error.status, output: `${error.stdout ?? ''}${error.stderr ?? ''}` };
    }
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

test('passes a compliant SearchToolbar page and css', () => {
  const result = runGate({});
  assert.equal(result.code, 0, result.output);
  assert.match(result.output, /0 finding\(s\)/);
});

test('flags FilterPanel reintroduction in modules', () => {
  const result = runGate({
    page: VALID_PAGE.replace(
      "import { SearchToolbar } from '../../components';",
      "import { SearchToolbar, FilterPanel } from '../../components';",
    ),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /no-filter-panel/);
});

test('flags inline keywordPlaceholder string', () => {
  const result = runGate({
    page: VALID_PAGE.replace(
      "keywordPlaceholder={t('system.demo.search.placeholder')}",
      'keywordPlaceholder="按名称搜索"',
    ),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /keyword-placeholder-i18n/);
});

test('flags onKeywordChange without controlled keyword', () => {
  const result = runGate({
    page: VALID_PAGE.replace("    keyword={query.keyword ?? ''}\n", ''),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /controlled-keyword/);
});

test('flags missing onClearAll', () => {
  const result = runGate({
    page: VALID_PAGE.replace('    onClearAll={reset}\n', ''),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /clear-all-required/);
});

test('flags legacy action-item class', () => {
  const result = runGate({
    page: VALID_PAGE.replace(
      'export default Page;',
      'export const legacy = "filter-panel__action-item";\nexport default Page;',
    ),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /no-action-item/);
});

test('flags toolbar control selectors re-declaring border or background', () => {
  const result = runGate({
    css: `${VALID_CSS}.search-toolbar .search-toolbar__inline .arco-select-view {\n  border: 2px solid red;\n  background: pink;\n}\n`,
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /inherit-border/);
  assert.match(result.output, /inherit-background/);
});

test('flags missing group-wrapper border reset', () => {
  const result = runGate({
    css: VALID_CSS.replace('  border: none;\n', ''),
  });
  assert.equal(result.code, 1);
  assert.match(result.output, /group-wrapper-layout-only/);
});

test('escape hatch comment suppresses a rule', () => {
  const result = runGate({
    page: VALID_PAGE.replace(
      'export default Page;',
      '// legacy bridge, tracked in PR: FilterPanel (search-toolbar-allow: no-filter-panel)\nexport default Page;',
    ),
  });
  assert.equal(result.code, 0, result.output);
});
