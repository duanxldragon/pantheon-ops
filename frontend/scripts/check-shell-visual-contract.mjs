import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const layoutCssPath = path.join(frontendRoot, 'src', 'core', 'layout', 'index.css');
const globalCssPath = path.join(frontendRoot, 'src', 'index.css');
const listPageCssPath = path.join(frontendRoot, 'src', 'modules', 'system', 'list-page.css');
const loginCssPath = path.join(frontendRoot, 'src', 'modules', 'auth', 'Login.css');
const modulesRoot = path.join(frontendRoot, 'src', 'modules');
const tableBatchActionBarPath = path.join(
  frontendRoot,
  'src',
  'components',
  'patterns',
  'TableBatchActionBar.tsx',
);
const appModalPath = path.join(frontendRoot, 'src', 'components', 'patterns', 'AppModal.tsx');
const appDrawerPath = path.join(frontendRoot, 'src', 'components', 'patterns', 'AppDrawer.tsx');
const appModalActionsPath = path.join(
  frontendRoot,
  'src',
  'components',
  'patterns',
  'AppModalActions.ts',
);
const formSectionPath = path.join(frontendRoot, 'src', 'components', 'patterns', 'FormSection.tsx');
const submitBarPath = path.join(frontendRoot, 'src', 'components', 'patterns', 'SubmitBar.tsx');
const pageEmptyPath = path.join(frontendRoot, 'src', 'components', 'feedback', 'PageEmpty.tsx');
const pageLoadingPath = path.join(frontendRoot, 'src', 'components', 'feedback', 'PageLoading.tsx');
const pageErrorPath = path.join(frontendRoot, 'src', 'components', 'feedback', 'PageError.tsx');
const userListPath = path.join(frontendRoot, 'src', 'modules', 'system', 'user', 'UserList.tsx');
const source = fs.readFileSync(layoutCssPath, 'utf8');
const globalSource = fs.readFileSync(globalCssPath, 'utf8');
const listPageSource = fs.readFileSync(listPageCssPath, 'utf8');
const loginCssSource = fs.readFileSync(loginCssPath, 'utf8');
const tableBatchActionBarSource = fs.readFileSync(tableBatchActionBarPath, 'utf8');
const appModalSource = fs.readFileSync(appModalPath, 'utf8');
const appDrawerSource = fs.readFileSync(appDrawerPath, 'utf8');
const appModalActionsSource = fs.readFileSync(appModalActionsPath, 'utf8');
const formSectionSource = fs.readFileSync(formSectionPath, 'utf8');
const submitBarSource = fs.readFileSync(submitBarPath, 'utf8');
const pageEmptySource = fs.readFileSync(pageEmptyPath, 'utf8');
const pageLoadingSource = fs.readFileSync(pageLoadingPath, 'utf8');
const pageErrorSource = fs.readFileSync(pageErrorPath, 'utf8');
const userListSource = fs.readFileSync(userListPath, 'utf8');
const dictTypeTabSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'dict', 'DictTypeTab.tsx'),
  'utf8',
);
const dictItemTabSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'dict', 'DictItemTab.tsx'),
  'utf8',
);
const dictPageSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'dict', 'DictPage.tsx'),
  'utf8',
);
const settingGroupPageSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'setting', 'SettingGroupPage.tsx'),
  'utf8',
);

const requiredGlobalTokens = [
  '--shell-table-card-padding',
  '--shell-control-min-height',
  '--shell-filter-body-padding',
  '--shell-filter-control-min-height',
  '--shell-filter-form-item-margin-bottom',
  '--shell-filter-label-padding-bottom',
  '--shell-list-actions-gap',
  '--shell-action-bar-gap',
  '--shell-action-bar-min-height',
  '--shell-table-head-gap',
  '--shell-governance-select-width',
];

function getBlock(cssSource, selector) {
  let start = cssSource.indexOf(selector);
  while (start >= 0) {
    let beforeIndex = start - 1;
    while (beforeIndex >= 0 && /\s/.test(cssSource[beforeIndex])) {
      beforeIndex -= 1;
    }
    const before = beforeIndex < 0 ? '\n' : cssSource[beforeIndex];
    const after = cssSource[start + selector.length] || '';
    if ((/[},]/.test(before) || before === '\n') && (/[\s,{]/.test(after) || after === '')) {
      break;
    }
    start = cssSource.indexOf(selector, start + selector.length);
  }
  if (start < 0) {
    return '';
  }
  const open = cssSource.indexOf('{', start);
  if (open < 0) {
    return '';
  }
  let depth = 0;
  for (let index = open; index < cssSource.length; index += 1) {
    const char = cssSource[index];
    if (char === '{') {
      depth += 1;
    } else if (char === '}') {
      depth -= 1;
      if (depth === 0) {
        return cssSource.slice(open + 1, index);
      }
    }
  }
  return '';
}

function requireBlock(cssSource, selector, findings) {
  const block = getBlock(cssSource, selector);
  if (!block) {
    findings.push(`Missing CSS block: ${selector}`);
  }
  return block;
}

function getStandaloneBlock(cssSource, selector) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = cssSource.match(
    new RegExp(String.raw`(?:^|\n)${escapedSelector}\s*\{([\s\S]*?)\n\}`, 'i'),
  );
  return match?.[1] || '';
}

function requireStandaloneBlock(cssSource, selector, findings) {
  const block = getStandaloneBlock(cssSource, selector);
  if (!block) {
    findings.push(`Missing CSS block: ${selector}`);
  }
  return block;
}

function hasDeclaration(block, property, expectedValue) {
  const pattern = new RegExp(
    String.raw`${property}\s*:\s*${expectedValue}(?:\s*!important)?\s*;`,
    'i',
  );
  return pattern.test(block);
}

function collectBorderLines(block) {
  return block
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => /^(border|box-shadow)\b/i.test(line));
}

function splitSelectorList(selectorText) {
  return selectorText
    .split(',')
    .map((selector) => selector.trim())
    .filter(Boolean);
}

function extractCssRules(cssSource) {
  const rules = [];
  const rulePattern = /([^{}]+)\{([^{}]*)\}/g;
  let match;
  while ((match = rulePattern.exec(cssSource))) {
    const selectorText = match[1].trim();
    if (!selectorText || selectorText.startsWith('@')) {
      continue;
    }
    rules.push({ selectorText, body: match[2] });
  }
  return rules;
}

function selectorTargetsBareArcoInput(selector) {
  return /(?:^|[\s>+~,(])\.arco-input(?=$|[\s:,[{),>+~])/i.test(selector);
}

function selectorTargetsNestedArcoInput(selector) {
  return /(?:\.arco-input-inner-wrapper|\.arco-input-password|\.arco-input-number)\s+\.arco-input(?=$|[\s:,[{),>+~])/i.test(
    selector,
  );
}

function selectorIsContentOnlyArcoInput(selector) {
  return (
    selectorTargetsBareArcoInput(selector) &&
    (/::placeholder/i.test(selector) || selectorTargetsNestedArcoInput(selector))
  );
}

function readFilesRecursive(root, predicate) {
  const files = [];
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const entryPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      files.push(...readFilesRecursive(entryPath, predicate));
    } else if (predicate(entryPath)) {
      files.push(entryPath);
    }
  }
  return files;
}

function extractSelfClosingJsxBlocks(sourceText, tagName) {
  const blocks = [];
  let searchIndex = 0;
  const needle = `<${tagName}`;
  while (searchIndex < sourceText.length) {
    const start = sourceText.indexOf(needle, searchIndex);
    if (start < 0) {
      break;
    }
    const end = sourceText.indexOf('/>', start);
    if (end < 0) {
      break;
    }
    blocks.push(sourceText.slice(start, end + 2));
    searchIndex = end + 2;
  }
  return blocks;
}

function hasPopconfirmNear(sourceText, index) {
  const before = sourceText.slice(Math.max(0, index - 1500), index);
  const after = sourceText.slice(index, Math.min(sourceText.length, index + 1500));
  return /<Popconfirm\b/.test(before) || /<Popconfirm\b/.test(after);
}

function extractButtonBlocks(sourceText) {
  const blocks = [];
  const buttonPattern = /<Button\b[\s\S]*?<\/Button>/g;
  let match;
  while ((match = buttonPattern.exec(sourceText))) {
    blocks.push({ block: match[0], index: match.index });
  }
  return blocks;
}

const findings = [];

const moduleSourceFiles = readFilesRecursive(
  modulesRoot,
  (entryPath) =>
    /\.(?:tsx|ts)$/.test(entryPath) &&
    !entryPath.endsWith('.test.ts') &&
    !entryPath.includes(`${path.sep}modules${path.sep}business${path.sep}`) &&
    !entryPath.includes(`${path.sep}modules${path.sep}generator${path.sep}`),
);

for (const sourcePath of moduleSourceFiles) {
  const moduleSource = fs.readFileSync(sourcePath, 'utf8');
  const relativePath = path.relative(frontendRoot, sourcePath).replaceAll(path.sep, '/');
  if (/<PageHeader\b/.test(moduleSource)) {
    findings.push(`${relativePath} must not render page-level PageHeader inside functional modules.`);
  }

  for (const block of extractSelfClosingJsxBlocks(moduleSource, 'GovernanceSummaryBar')) {
    if (/(IconPlus|IconDownload|ImportCsvButton|common\.add|common\.export|common\.import|common\.refresh)/.test(block)) {
      findings.push(
        `${relativePath} must not put CRUD/import/export/refresh actions inside GovernanceSummaryBar.`,
      );
    }
  }

  if (relativePath.endsWith('.tsx')) {
    for (const { block, index } of extractButtonBlocks(moduleSource)) {
      const rendersDeleteAction =
        /<IconDelete\b/.test(block) ||
        /deleteSelected|batchDelete|common\.delete|\.delete'|\.delete"/.test(block);
      if (!rendersDeleteAction) {
        continue;
      }
      if (!/status\s*=\s*(?:"danger"|\{[\s\S]*?danger[\s\S]*?\})/.test(block)) {
        findings.push(`${relativePath} destructive action buttons must render with danger status.`);
        break;
      }
      if (
        !hasPopconfirmNear(moduleSource, index) &&
        !/Modal|Drawer|SecurityCenter|FieldEditor/.test(relativePath)
      ) {
        findings.push(
          `${relativePath} destructive action buttons must be protected by Popconfirm or modal confirmation.`,
        );
        break;
      }
    }
  }
}

for (const token of requiredGlobalTokens) {
  if (!globalSource.includes(token)) {
    findings.push(`Missing platform UI token: ${token}`);
  }
}

if (!/:focus-visible\s*\{/i.test(globalSource)) {
  findings.push('Global CSS must define a keyboard-visible :focus-visible outline.');
}

if (!/@media\s*\(\s*prefers-reduced-motion\s*:\s*reduce\s*\)/i.test(globalSource)) {
  findings.push('Global CSS must respect prefers-reduced-motion: reduce.');
}

const globalButtonBlock = requireStandaloneBlock(globalSource, '.arco-btn', findings);
if (globalButtonBlock) {
  if (!hasDeclaration(globalButtonBlock, 'min-height', 'var\\(--shell-control-min-height\\)')) {
    findings.push('.arco-btn must use --shell-control-min-height for stable controls.');
  }
  if (!hasDeclaration(globalButtonBlock, 'line-height', '20px')) {
    findings.push('.arco-btn must keep a stable 20px line-height.');
  }
}

const globalIconButtonBlock = requireStandaloneBlock(globalSource, '.arco-btn-icon-only', findings);
if (
  globalIconButtonBlock &&
  !hasDeclaration(globalIconButtonBlock, 'min-width', 'var\\(--shell-control-min-height\\)')
) {
  findings.push('.arco-btn-icon-only must use --shell-control-min-height for stable icon buttons.');
}

const hasStableControlMinHeight =
  /\.arco-input-inner-wrapper,[\s\S]*?\.arco-input-password,[\s\S]*?\.arco-input-number,[\s\S]*?\.arco-input-tag\s*\{[\s\S]*?min-height\s*:\s*var\(--shell-control-min-height\)\s*;/i.test(
    globalSource,
  );
if (!hasStableControlMinHeight) {
  findings.push('Global input/select/picker controls must use --shell-control-min-height.');
}

for (const { selectorText, body } of extractCssRules(globalSource)) {
  const selectors = splitSelectorList(selectorText);
  const hasControlFramingDeclaration =
    /\b(?:border(?:-color)?|box-shadow|background|min-height)\s*:/i.test(body);
  if (!hasControlFramingDeclaration) {
    continue;
  }
  for (const selector of selectors) {
    if (
      selectorTargetsBareArcoInput(selector) &&
      !selectorTargetsNestedArcoInput(selector) &&
      !selectorIsContentOnlyArcoInput(selector)
    ) {
      findings.push(
        `Bare .arco-input must not be part of outer control framing selectors: ${selector}`,
      );
      break;
    }
  }
}

const platformCssSources = [
  ['global CSS', globalSource],
  ['layout CSS', source],
  ['system list-page CSS', listPageSource],
];

for (const [label, cssSource] of platformCssSources) {
  if (/radial-gradient\s*\(/i.test(cssSource)) {
    findings.push(`${label} must not use radial-gradient decoration in the backoffice shell.`);
  }
  if (/linear-gradient\s*\(/i.test(cssSource)) {
    findings.push(
      `${label} must not use broad linear-gradient decoration in the backoffice shell.`,
    );
  }
  if (/font-weight\s*:\s*(?:620|650)\s*;/i.test(cssSource)) {
    findings.push(`${label} must use standard font weights, not 620/650.`);
  }
  if (/var\(--(?:color|arcoblue|green|red|orange|gray)-[^)]+\)/i.test(cssSource)) {
    findings.push(`${label} must route Arco color tokens through Pantheon semantic tokens.`);
  }
}

const headerBlock = requireBlock(source, '.app-shell__header', findings);
if (headerBlock) {
  if (!hasDeclaration(headerBlock, 'height', 'auto')) {
    findings.push(
      '.app-shell__header must reset Arco Layout.Header fixed height with height: auto.',
    );
  }
  if (!hasDeclaration(headerBlock, 'line-height', 'normal')) {
    findings.push('.app-shell__header must reset inherited header line-height to normal.');
  }
}

const headerMetaBlock = requireBlock(source, '.app-shell__header-meta', findings);
if (headerMetaBlock) {
  if (!hasDeclaration(headerMetaBlock, 'align-items', 'center')) {
    findings.push('.app-shell__header-meta must vertically center breadcrumb content.');
  }
  if (/overflow\s*:\s*hidden\s*;/i.test(headerMetaBlock)) {
    findings.push('.app-shell__header-meta must not clip breadcrumb text.');
  }
}

const breadcrumbBlock = requireBlock(source, '.app-shell__header-breadcrumb', findings);
if (breadcrumbBlock) {
  if (!hasDeclaration(breadcrumbBlock, 'line-height', '20px')) {
    findings.push(
      '.app-shell__header-breadcrumb must use line-height: 20px to avoid text clipping.',
    );
  }
  if (/overflow\s*:\s*hidden\s*;/i.test(breadcrumbBlock)) {
    findings.push('.app-shell__header-breadcrumb must not use overflow: hidden.');
  }
}

const breadcrumbItemBlock = requireBlock(
  source,
  '.app-shell__header .arco-breadcrumb-item',
  findings,
);
if (breadcrumbItemBlock) {
  if (!hasDeclaration(breadcrumbItemBlock, 'line-height', '24px')) {
    findings.push('.app-shell breadcrumb items must keep a stable 24px line-height.');
  }
  if (!hasDeclaration(breadcrumbItemBlock, 'overflow', 'visible')) {
    findings.push('.app-shell breadcrumb items must not clip glyph ascenders.');
  }
}

const breadcrumbSeparatorBlock = requireBlock(
  source,
  '.app-shell__header .arco-breadcrumb-item-separator',
  findings,
);
if (breadcrumbSeparatorBlock && !hasDeclaration(breadcrumbSeparatorBlock, 'line-height', '24px')) {
  findings.push('.app-shell breadcrumb separators must align to the same 24px line-height.');
}

const tabSelectors = [
  '.app-shell__tabs',
  '.app-shell__tab',
  '.app-shell__tab:hover',
  '.app-shell__tab--active',
  '.app-shell__tab--drag-over',
  '.app-shell__tab-menu',
];

for (const selector of tabSelectors) {
  const block = requireBlock(source, selector, findings);
  if (!block) {
    continue;
  }
  for (const line of collectBorderLines(block)) {
    if (/color-mix\(/i.test(line)) {
      findings.push(`${selector} uses mixed/gradient-like border styling: ${line}`);
    }
    if (/box-shadow\s*:\s*inset/i.test(line)) {
      findings.push(`${selector} uses inset border styling: ${line}`);
    }
  }
}

const openedTabBlock = requireBlock(source, '.app-shell__tab', findings);
if (openedTabBlock) {
  if (!hasDeclaration(openedTabBlock, 'line-height', '20px')) {
    findings.push('.app-shell__tab must use a stable 20px line-height to avoid text clipping.');
  }
  if (!hasDeclaration(openedTabBlock, 'border', '1px solid transparent')) {
    findings.push('.app-shell__tab must not render a visible per-tab border.');
  }
}

const openedTabLabelBlock = requireBlock(source, '.app-shell__tab-label', findings);
if (openedTabLabelBlock && !hasDeclaration(openedTabLabelBlock, 'line-height', '20px')) {
  findings.push('.app-shell__tab-label must keep a stable 20px line-height.');
}

const batchBarBlock = requireBlock(listPageSource, '.table-batch-action-bar', findings);
if (batchBarBlock) {
  if (!hasDeclaration(batchBarBlock, 'border', '0')) {
    findings.push('.table-batch-action-bar must not render its own border.');
  }
  if (!hasDeclaration(batchBarBlock, 'background', 'transparent')) {
    findings.push('.table-batch-action-bar must keep a transparent toolbar background.');
  }
  if (!hasDeclaration(batchBarBlock, 'box-shadow', 'none')) {
    findings.push('.table-batch-action-bar must not render shadow-like borders.');
  }
  for (const line of collectBorderLines(batchBarBlock)) {
    if (/color-mix\(/i.test(line)) {
      findings.push(`.table-batch-action-bar uses mixed/gradient-like border styling: ${line}`);
    }
    if (/box-shadow\s*:\s*inset/i.test(line)) {
      findings.push(`.table-batch-action-bar uses inset border styling: ${line}`);
    }
  }
}

const batchButtonBlock = requireBlock(
  listPageSource,
  '.table-batch-action-bar .arco-btn-text',
  findings,
);
if (batchButtonBlock) {
  if (!hasDeclaration(batchButtonBlock, 'border', '0')) {
    findings.push('.table-batch-action-bar text buttons must not render borders.');
  }
  if (!hasDeclaration(batchButtonBlock, 'background', 'transparent')) {
    findings.push('.table-batch-action-bar text buttons must keep a transparent background.');
  }
}

const systemTableCardBlock = requireBlock(
  listPageSource,
  '.system-list__table-card > .arco-card-body',
  findings,
);
if (systemTableCardBlock) {
  if (
    !/padding\s*:\s*var\(--shell-table-card-padding\)\s*!important\s*;/i.test(
      systemTableCardBlock,
    )
  ) {
    findings.push(
      '.system-list__table-card must use --shell-table-card-padding so table left/right spacing is consistent.',
    );
  }
}

const filterBodyBlock = requireBlock(globalSource, '.filter-panel__body', findings);
if (filterBodyBlock) {
  if (!hasDeclaration(filterBodyBlock, 'padding', 'var\\(--shell-filter-body-padding\\)')) {
    findings.push('.filter-panel__body must use --shell-filter-body-padding.');
  }
}

const filterFormItemBlock = requireBlock(globalSource, '.filter-panel .arco-form-item', findings);
if (
  filterFormItemBlock &&
  !hasDeclaration(
    filterFormItemBlock,
    'margin-bottom',
    'var\\(--shell-filter-form-item-margin-bottom\\)',
  )
) {
  findings.push('.filter-panel form item spacing must use --shell-filter-form-item-margin-bottom.');
}

const filterControlBlock = requireBlock(
  globalSource,
  '.filter-panel .arco-input-inner-wrapper',
  findings,
);
if (
  filterControlBlock &&
  !/min-height\s*:\s*var\(--shell-filter-control-min-height\)\s*;/i.test(filterControlBlock)
) {
  findings.push('FilterPanel controls must use --shell-filter-control-min-height.');
}

const actionItemButtonBlock = requireBlock(
  globalSource,
  '.filter-panel__action-item .arco-btn',
  findings,
);
if (
  actionItemButtonBlock &&
  !hasDeclaration(actionItemButtonBlock, 'min-height', 'var\\(--shell-filter-control-min-height\\)')
) {
  findings.push('FilterPanel action buttons must align to --shell-filter-control-min-height.');
}

const submitBarBlock = requireBlock(globalSource, '.submit-bar', findings);
if (submitBarBlock) {
  if (!hasDeclaration(submitBarBlock, 'justify-content', 'flex-end')) {
    findings.push('.submit-bar must align form actions to the right.');
  }
  if (!hasDeclaration(submitBarBlock, 'width', '100%')) {
    findings.push('.submit-bar must span the dialog/form footer width.');
  }
}

const pageEmptyLoadingBlock = requireStandaloneBlock(globalSource, '.page-empty', findings);
if (pageEmptyLoadingBlock) {
  if (!hasDeclaration(pageEmptyLoadingBlock, 'min-height', '220px')) {
    findings.push('.page-empty must keep a stable 220px minimum height.');
  }
  if (!hasDeclaration(pageEmptyLoadingBlock, 'padding', '16px 0')) {
    findings.push('.page-empty must keep shared vertical padding.');
  }
}

const pageLoadingHasMinHeight = /(?:^|\n)\.page-loading\s*\{[\s\S]*?min-height\s*:\s*240px\s*;/i.test(
  globalSource,
);
if (!pageLoadingHasMinHeight) {
  requireBlock(globalSource, '.page-loading', findings);
  findings.push('.page-loading must keep a stable 240px minimum height.');
}

const pageEmptyInnerBlock = requireBlock(globalSource, '.page-empty .page-empty__inner', findings);
if (pageEmptyInnerBlock) {
  if (!hasDeclaration(pageEmptyInnerBlock, 'width', 'min\\(100%, 420px\\)')) {
    findings.push('.page-empty inner content must use a constrained readable width.');
  }
  if (!hasDeclaration(pageEmptyInnerBlock, 'border-radius', 'var\\(--radius-md\\)')) {
    findings.push('.page-empty inner content must use radius-md.');
  }
}

const pageResultBlock = requireBlock(globalSource, '.page-result', findings);
if (pageResultBlock) {
  if (!hasDeclaration(pageResultBlock, 'width', 'min\\(100%, 720px\\)')) {
    findings.push('.page-result must use a constrained readable width.');
  }
  if (!hasDeclaration(pageResultBlock, 'border-radius', 'var\\(--radius-md\\)')) {
    findings.push('.page-result must use radius-md.');
  }
}

const listHeaderActionsBlock = requireBlock(listPageSource, '.list-header-actions', findings);
if (
  listHeaderActionsBlock &&
  !hasDeclaration(listHeaderActionsBlock, 'gap', 'var\\(--shell-list-actions-gap\\)')
) {
  findings.push('.list-header-actions must use --shell-list-actions-gap.');
}

const workActionsBlock = requireBlock(listPageSource, '.system-list__work-actions', findings);
if (workActionsBlock) {
  if (!hasDeclaration(workActionsBlock, 'justify-content', 'flex-end')) {
    findings.push('.system-list__work-actions must align work-area actions to the right.');
  }
  if (!hasDeclaration(workActionsBlock, 'gap', 'var\\(--shell-list-actions-gap\\)')) {
    findings.push('.system-list__work-actions must use --shell-list-actions-gap.');
  }
}

if (!/prefixActions\?:\s*React\.ReactNode/.test(tableBatchActionBarSource)) {
  findings.push('TableBatchActionBar must expose prefixActions for list-scope business actions.');
}

if (!/table-batch-action-bar__prefix-actions/.test(tableBatchActionBarSource)) {
  findings.push('TableBatchActionBar must render a prefix actions slot before selection metadata.');
}

if (!/className=\{mergeDialogClassName\('app-dialog'/.test(appModalSource)) {
  findings.push('AppModal must always include the shared app-dialog class.');
}

if (!/maskClosable\s*=\s*false/.test(appModalSource)) {
  findings.push('AppModal must default maskClosable to false for form safety.');
}

if (!/className=\{className \? `app-drawer/.test(appDrawerSource)) {
  findings.push('AppDrawer must always include the shared app-drawer class.');
}

if (!/maskClosable\s*=\s*false/.test(appDrawerSource)) {
  findings.push('AppDrawer must default maskClosable to false for form safety.');
}

for (const actionName of ['Confirm', 'Success', 'Error']) {
  if (
    !new RegExp(String.raw`showAppModal${actionName}[\s\S]*?mergeDialogClassName\('app-dialog'`).test(
      appModalActionsSource,
    )
  ) {
    findings.push(`showAppModal${actionName} must attach the shared app-dialog class.`);
  }
}

if (!/className="form-section"/.test(formSectionSource)) {
  findings.push('FormSection must render the shared form-section class.');
}

if (!/className="submit-bar"/.test(submitBarSource)) {
  findings.push('SubmitBar must render the shared submit-bar class.');
}

if (!/className="page-empty"/.test(pageEmptySource)) {
  findings.push('PageEmpty must render the shared page-empty class.');
}

if (!/className="page-loading"/.test(pageLoadingSource)) {
  findings.push('PageLoading must render the shared page-loading class.');
}

if (!/className="page-result"/.test(pageErrorSource)) {
  findings.push('PageError must render the shared page-result class.');
}

const batchMainBlock = requireBlock(listPageSource, '.table-batch-action-bar__main', findings);
if (batchMainBlock) {
  if (!hasDeclaration(batchMainBlock, 'gap', 'var\\(--shell-action-bar-gap\\)')) {
    findings.push('.table-batch-action-bar__main must use --shell-action-bar-gap.');
  }
  if (!hasDeclaration(batchMainBlock, 'min-height', 'var\\(--shell-action-bar-min-height\\)')) {
    findings.push('.table-batch-action-bar__main must use --shell-action-bar-min-height.');
  }
}

const appDialogControlBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-input-inner-wrapper',
  findings,
);
if (appDialogControlBlock) {
  if (!hasDeclaration(appDialogControlBlock, 'border', '1px solid var\\(--panel-border-strong\\)')) {
    findings.push('.app-dialog controls must render one shared outer border.');
  }
  if (!hasDeclaration(appDialogControlBlock, 'background', '#fff')) {
    findings.push('.app-dialog controls must use a single white control background.');
  }
  if (!hasDeclaration(appDialogControlBlock, 'box-shadow', 'none')) {
    findings.push('.app-dialog controls must not render a second idle shadow layer.');
  }
}

const appDialogInputNumberControlBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-input-number',
  findings,
);
if (appDialogInputNumberControlBlock) {
  if (
    !hasDeclaration(appDialogInputNumberControlBlock, 'border', '1px solid var\\(--panel-border-strong\\)')
  ) {
    findings.push('.app-dialog InputNumber outer control must render one shared border.');
  }
}

const appDrawerControlBlock = requireBlock(
  globalSource,
  '.app-drawer .arco-input-inner-wrapper',
  findings,
);
if (appDrawerControlBlock) {
  if (!hasDeclaration(appDrawerControlBlock, 'border', '1px solid var\\(--panel-border-strong\\)')) {
    findings.push('.app-drawer controls must render one shared outer border.');
  }
  if (!hasDeclaration(appDrawerControlBlock, 'background', '#fff')) {
    findings.push('.app-drawer controls must use a single white control background.');
  }
  if (!hasDeclaration(appDrawerControlBlock, 'box-shadow', 'none')) {
    findings.push('.app-drawer controls must not render a second idle shadow layer.');
  }
}

const appDialogBlock = requireBlock(globalSource, '.app-dialog', findings);
if (appDialogBlock) {
  if (!hasDeclaration(appDialogBlock, 'max-width', 'calc\\(100vw - 32px\\)')) {
    findings.push('.app-dialog must respect desktop viewport width.');
  }
  if (!hasDeclaration(appDialogBlock, 'max-height', 'calc\\(100dvh - 32px\\)')) {
    findings.push('.app-dialog must respect desktop viewport height.');
  }
  if (!hasDeclaration(appDialogBlock, 'overflow', 'hidden')) {
    findings.push('.app-dialog must clip only its own shell, not internal content.');
  }
}

const appDialogAppearBlock = requireBlock(globalSource, '.app-dialog.zoomModal-appear', findings);
if (appDialogAppearBlock) {
  if (!/animation-name\s*:\s*app-dialog-no-scale\s*!important\s*;/i.test(appDialogAppearBlock)) {
    findings.push('.app-dialog must replace zoom animation with the shared no-scale animation.');
  }
  if (!/animation-duration\s*:\s*1ms\s*!important\s*;/i.test(appDialogAppearBlock)) {
    findings.push('.app-dialog animation must be short enough to avoid transient layout scaling.');
  }
  if (!/transform\s*:\s*none\s*!important\s*;/i.test(appDialogAppearBlock)) {
    findings.push('.app-dialog must not use transform scaling during modal entry.');
  }
}

if (!/@keyframes\s+app-dialog-no-scale/i.test(globalSource)) {
  findings.push('.app-dialog must define app-dialog-no-scale keyframes.');
}

const appDialogHeaderBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-modal-header',
  findings,
);
if (appDialogHeaderBlock) {
  if (!/height\s*:\s*64px\s*!important\s*;/i.test(appDialogHeaderBlock)) {
    findings.push('.app-dialog header must keep a stable 64px height.');
  }
  if (!/min-height\s*:\s*64px\s*!important\s*;/i.test(appDialogHeaderBlock)) {
    findings.push('.app-dialog header must keep a stable 64px minimum height.');
  }
  if (!/padding\s*:\s*16px 24px\s*!important\s*;/i.test(appDialogHeaderBlock)) {
    findings.push('.app-dialog header must use shared desktop padding.');
  }
}

const appDialogContentBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-modal-content',
  findings,
);
if (appDialogContentBlock) {
  if (!hasDeclaration(appDialogContentBlock, 'padding', '20px 24px 24px')) {
    findings.push('.app-dialog content must use shared dialog padding.');
  }
  if (!hasDeclaration(appDialogContentBlock, 'overflow-y', 'auto')) {
    findings.push('.app-dialog content must scroll internally.');
  }
  if (!hasDeclaration(appDialogContentBlock, 'overflow-x', 'hidden')) {
    findings.push('.app-dialog content must prevent horizontal overflow.');
  }
}

const appDialogFooterBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-modal-footer',
  findings,
);
if (appDialogFooterBlock && !hasDeclaration(appDialogFooterBlock, 'padding', '16px 24px 20px')) {
  findings.push('.app-dialog footer must use shared dialog padding.');
}

const formSectionBlock = requireBlock(globalSource, '.form-section__title', findings);
if (formSectionBlock) {
  if (!hasDeclaration(formSectionBlock, 'font-size', '14px')) {
    findings.push('.form-section titles must use compact 14px text.');
  }
  if (!hasDeclaration(formSectionBlock, 'font-weight', '600')) {
    findings.push('.form-section titles must use a standard 600 weight.');
  }
}

const dialogStackBlock = requireBlock(globalSource, '.app-dialog .detail-stack', findings);
if (dialogStackBlock && !hasDeclaration(dialogStackBlock, 'gap', '12px')) {
  findings.push('.app-dialog form/detail stacks must keep a 12px rhythm.');
}

const appDialogInnerInputBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-input-inner-wrapper .arco-input',
  findings,
);
if (appDialogInnerInputBlock) {
  if (!hasDeclaration(appDialogInnerInputBlock, 'border', '0')) {
    findings.push('.app-dialog inner inputs must remove their own border.');
  }
  if (!hasDeclaration(appDialogInnerInputBlock, 'background', 'transparent')) {
    findings.push('.app-dialog inner inputs must keep a transparent background.');
  }
  if (!hasDeclaration(appDialogInnerInputBlock, 'box-shadow', 'none')) {
    findings.push('.app-dialog inner inputs must not render their own shadow.');
  }
  if (!hasDeclaration(appDialogInnerInputBlock, 'outline', '0')) {
    findings.push('.app-dialog inner inputs must not render their own focus outline.');
  }
}

const appDialogInputNumberInnerBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-input-number .arco-input-inner-wrapper',
  findings,
);
if (appDialogInputNumberInnerBlock) {
  if (/border\s*:\s*1px/i.test(appDialogInputNumberInnerBlock)) {
    findings.push(
      'App dialog InputNumber must not draw a second border on .arco-input-inner-wrapper.',
    );
  }
  if (!hasDeclaration(appDialogInputNumberInnerBlock, 'border', '0')) {
    findings.push('.app-dialog InputNumber inner wrapper must remove its own border.');
  }
  if (!hasDeclaration(appDialogInputNumberInnerBlock, 'background', 'transparent')) {
    findings.push('.app-dialog InputNumber inner wrapper must keep a transparent background.');
  }
  if (!hasDeclaration(appDialogInputNumberInnerBlock, 'box-shadow', 'none')) {
    findings.push('.app-dialog InputNumber inner wrapper must not render its own shadow.');
  }
  if (!hasDeclaration(appDialogInputNumberInnerBlock, 'outline', '0')) {
    findings.push('.app-dialog InputNumber inner wrapper must not render its own outline.');
  }
}

const appDialogSelectFocusBlock = requireBlock(
  globalSource,
  '.app-dialog .arco-select-open .arco-select-view',
  findings,
);
if (appDialogSelectFocusBlock) {
  if (!hasDeclaration(appDialogSelectFocusBlock, 'border-color', 'var\\(--brand-primary\\)')) {
    findings.push('.app-dialog Select open state must use the active brand border color.');
  }
  if (
    !hasDeclaration(
      appDialogSelectFocusBlock,
      'box-shadow',
      '0 0 0 3px color-mix\\(in srgb, var\\(--brand-primary\\) 14%, transparent\\)',
    )
  ) {
    findings.push('.app-dialog Select open state must use the shared brand focus ring.');
  }
}

const dialogCardTitleSpaceBlock = requireBlock(
  globalSource,
  '.app-dialog .dialog-grid-card .arco-card-header-title .arco-space',
  findings,
);
if (dialogCardTitleSpaceBlock) {
  if (!hasDeclaration(dialogCardTitleSpaceBlock, 'flex-wrap', 'nowrap')) {
    findings.push('.dialog-grid-card headers must keep title rows on a single line.');
  }
  if (!hasDeclaration(dialogCardTitleSpaceBlock, 'width', '100%')) {
    findings.push('.dialog-grid-card headers must reserve full width for title and count tag.');
  }
}

const dialogCardTitleTextBlock = requireBlock(
  globalSource,
  '.app-dialog .dialog-grid-card .arco-card-header-title .arco-typography',
  findings,
);
if (dialogCardTitleTextBlock && !hasDeclaration(dialogCardTitleTextBlock, 'white-space', 'nowrap')) {
  findings.push('.dialog-grid-card title text must not wrap.');
}

const appDrawerInnerInputBlock = requireBlock(
  globalSource,
  '.app-drawer .arco-input-inner-wrapper .arco-input',
  findings,
);
if (appDrawerInnerInputBlock) {
  if (!hasDeclaration(appDrawerInnerInputBlock, 'border', '0')) {
    findings.push('.app-drawer inner inputs must remove their own border.');
  }
  if (!hasDeclaration(appDrawerInnerInputBlock, 'background', 'transparent')) {
    findings.push('.app-drawer inner inputs must keep a transparent background.');
  }
  if (!hasDeclaration(appDrawerInnerInputBlock, 'box-shadow', 'none')) {
    findings.push('.app-drawer inner inputs must not render their own shadow.');
  }
  if (!hasDeclaration(appDrawerInnerInputBlock, 'outline', '0')) {
    findings.push('.app-drawer inner inputs must not render their own focus outline.');
  }
}

const appDrawerInputNumberInnerBlock = requireBlock(
  globalSource,
  '.app-drawer .arco-input-number .arco-input-inner-wrapper',
  findings,
);
if (appDrawerInputNumberInnerBlock) {
  if (/border\s*:\s*1px/i.test(appDrawerInputNumberInnerBlock)) {
    findings.push(
      'App drawer InputNumber must not draw a second border on .arco-input-inner-wrapper.',
    );
  }
  if (!hasDeclaration(appDrawerInputNumberInnerBlock, 'border', '0')) {
    findings.push('.app-drawer InputNumber inner wrapper must remove its own border.');
  }
  if (!hasDeclaration(appDrawerInputNumberInnerBlock, 'background', 'transparent')) {
    findings.push('.app-drawer InputNumber inner wrapper must keep a transparent background.');
  }
  if (!hasDeclaration(appDrawerInputNumberInnerBlock, 'box-shadow', 'none')) {
    findings.push('.app-drawer InputNumber inner wrapper must not render its own shadow.');
  }
  if (!hasDeclaration(appDrawerInputNumberInnerBlock, 'outline', '0')) {
    findings.push('.app-drawer InputNumber inner wrapper must not render its own outline.');
  }
}

const globalNestedInputBlock = requireBlock(
  globalSource,
  '.arco-input-inner-wrapper .arco-input',
  findings,
);
if (globalNestedInputBlock) {
  if (!hasDeclaration(globalNestedInputBlock, 'border', '0')) {
    findings.push('Nested Arco inputs must remove their own border globally.');
  }
  if (!hasDeclaration(globalNestedInputBlock, 'background', 'transparent')) {
    findings.push('Nested Arco inputs must keep a transparent background globally.');
  }
  if (!hasDeclaration(globalNestedInputBlock, 'box-shadow', 'none')) {
    findings.push('Nested Arco inputs must not render their own shadow globally.');
  }
  if (!hasDeclaration(globalNestedInputBlock, 'outline', '0')) {
    findings.push('Nested Arco inputs must not render their own focus outline globally.');
  }
}

const globalInputNumberInnerBlock = requireBlock(
  globalSource,
  '.arco-input-number .arco-input-inner-wrapper',
  findings,
);
if (globalInputNumberInnerBlock) {
  if (!hasDeclaration(globalInputNumberInnerBlock, 'border', '0')) {
    findings.push('InputNumber inner wrapper must remove its own border globally.');
  }
  if (!hasDeclaration(globalInputNumberInnerBlock, 'background', 'transparent')) {
    findings.push('InputNumber inner wrapper must keep a transparent background globally.');
  }
  if (!hasDeclaration(globalInputNumberInnerBlock, 'box-shadow', 'none')) {
    findings.push('InputNumber inner wrapper must not render its own shadow globally.');
  }
  if (!hasDeclaration(globalInputNumberInnerBlock, 'outline', '0')) {
    findings.push('InputNumber inner wrapper must not render its own focus outline globally.');
  }
}

for (const selectorList of globalSource.match(/[^{}]*\.arco-input-number\s+\.arco-input-inner-wrapper[^{}]*\{[^{}]*\}/g) ?? []) {
  const borderedDeclaration = selectorList.match(/\b(border|border-color)\s*:\s*([^;]+);/i);
  if (borderedDeclaration && borderedDeclaration[2].trim() !== '0') {
    findings.push(
      'InputNumber inner wrapper must not be part of the bordered control group; the outer .arco-input-number owns the border.',
    );
    break;
  }
}

const globalNestedInputFocusBlock = requireBlock(
  globalSource,
  '.arco-input-inner-wrapper .arco-input:focus',
  findings,
);
if (globalNestedInputFocusBlock) {
  if (!hasDeclaration(globalNestedInputFocusBlock, 'border', '0')) {
    findings.push('Focused nested Arco inputs must keep border disabled globally.');
  }
  if (!hasDeclaration(globalNestedInputFocusBlock, 'background', 'transparent')) {
    findings.push('Focused nested Arco inputs must keep transparent background globally.');
  }
  if (!hasDeclaration(globalNestedInputFocusBlock, 'box-shadow', 'none')) {
    findings.push('Focused nested Arco inputs must not render their own shadow globally.');
  }
  if (!hasDeclaration(globalNestedInputFocusBlock, 'outline', '0')) {
    findings.push('Focused nested Arco inputs must not render their own outline globally.');
  }
}

if (/(?:^|\n)\s*\.arco-input:focus\s*,/i.test(globalSource)) {
  findings.push(
    'Bare .arco-input:focus must not own the global focus ring; the outer input wrapper owns it.',
  );
}

const loginControlBlock = requireBlock(
  loginCssSource,
  '.auth-login-card .arco-input-inner-wrapper',
  findings,
);
if (loginControlBlock) {
  if (!hasDeclaration(loginControlBlock, 'border', '1px solid var\\(--panel-border-strong\\)')) {
    findings.push('.auth-login-card controls must render one shared outer border.');
  }
  if (!hasDeclaration(loginControlBlock, 'background', '#ffffff')) {
    findings.push('.auth-login-card controls must use a single white control background.');
  }
  if (!hasDeclaration(loginControlBlock, 'box-shadow', 'none')) {
    findings.push('.auth-login-card controls must not render a second idle shadow layer.');
  }
}

const loginInnerInputBlock = requireBlock(
  loginCssSource,
  '.auth-login-card .arco-input-inner-wrapper .arco-input',
  findings,
);
if (loginInnerInputBlock) {
  if (!hasDeclaration(loginInnerInputBlock, 'border', '0')) {
    findings.push('.auth-login-card inner inputs must remove their own border.');
  }
  if (!hasDeclaration(loginInnerInputBlock, 'background', 'transparent')) {
    findings.push('.auth-login-card inner inputs must keep a transparent background.');
  }
  if (!hasDeclaration(loginInnerInputBlock, 'box-shadow', 'none')) {
    findings.push('.auth-login-card inner inputs must not render their own shadow.');
  }
  if (!hasDeclaration(loginInnerInputBlock, 'outline', '0')) {
    findings.push('.auth-login-card inner inputs must not render their own focus outline.');
  }
}

const loginPasswordInnerInputBlock = requireBlock(
  loginCssSource,
  '.auth-login-card .arco-input-password .arco-input',
  findings,
);
if (loginPasswordInnerInputBlock) {
  if (!hasDeclaration(loginPasswordInnerInputBlock, 'border', '0')) {
    findings.push('.auth-login-card password inner inputs must remove their own border.');
  }
  if (!hasDeclaration(loginPasswordInnerInputBlock, 'background', 'transparent')) {
    findings.push('.auth-login-card password inner inputs must keep a transparent background.');
  }
  if (!hasDeclaration(loginPasswordInnerInputBlock, 'box-shadow', 'none')) {
    findings.push('.auth-login-card password inner inputs must not render their own shadow.');
  }
  if (!hasDeclaration(loginPasswordInnerInputBlock, 'outline', '0')) {
    findings.push('.auth-login-card password inner inputs must not render their own focus outline.');
  }
}

const governanceActionsBlock = requireBlock(
  listPageSource,
  '.table-batch-action-bar--governance .table-batch-action-bar__actions',
  findings,
);
if (
  governanceActionsBlock &&
  !hasDeclaration(governanceActionsBlock, 'justify-content', 'flex-end')
) {
  findings.push('Governance action bar secondary actions must align to the right.');
}

const governanceSelectBlock = requireBlock(
  listPageSource,
  '.table-batch-action-bar__select',
  findings,
);
if (
  governanceSelectBlock &&
  !hasDeclaration(governanceSelectBlock, 'width', 'var\\(--shell-governance-select-width\\)')
) {
  findings.push('.table-batch-action-bar__select must use --shell-governance-select-width.');
}

const compressedTableCardOverride =
  /\.(?:i18n-list-page|dept-list-page|post-list-page|setting-page|dict-page)[^{]*\.system-list__table-card\s+\.arco-card-body\s*\{/i;
if (compressedTableCardOverride.test(listPageSource)) {
  findings.push('System pages must not override .system-list__table-card .arco-card-body padding.');
}

const pageSpecificFilterOverride =
  /\.(?:i18n-list-page|dept-list-page|post-list-page|setting-page|dict-page|module-manager-page|permission-list-page|menu-list-page)[^{]*\.filter-panel\s+\.(?:arco-card-body|arco-form-item|arco-form-item-label-col|arco-input|arco-input-inner-wrapper|arco-select-view|arco-tree-select-view|arco-picker)\s*\{/i;
if (pageSpecificFilterOverride.test(listPageSource)) {
  findings.push('System pages must not override FilterPanel spacing or control height.');
}

const pageSpecificActionOverride =
  /\.(?:i18n-list-page|dept-list-page|post-list-page|setting-page|module-manager-page)[^{]*\.list-header-actions__[^{]*\{[^}]*?(?:gap|min-height)\s*:/is;
if (pageSpecificActionOverride.test(listPageSource)) {
  findings.push('System pages must not override ListHeaderActions gap or button height.');
}

if (
  /\.dict-page__actions\s*\{/i.test(globalSource) ||
  /\.dict-page__actions\s*\{/i.test(listPageSource)
) {
  findings.push(
    'Dict page must not override ListHeaderActions alignment through .dict-page__actions.',
  );
}

const pageSpecificBatchOverride =
  /\.(?:i18n-list-page|dept-list-page|post-list-page|setting-page)[^{]*\.table-batch-action-bar\s*\{[^}]*?gap\s*:/is;
if (pageSpecificBatchOverride.test(listPageSource)) {
  findings.push('System pages must not override TableBatchActionBar gap.');
}

if (!/system-list__table-card system-user-list__table-card/.test(userListSource)) {
  findings.push('UserList table card must include the shared system-list__table-card class.');
}

if (!/<AppTable<DictTypeRow>[\s\S]*?className="system-list__table"/.test(dictTypeTabSource)) {
  findings.push('DictTypeTab AppTable must include the shared system-list__table class.');
}

if (!/<AppTable<DictItemRow>[\s\S]*?className="system-list__table"/.test(dictItemTabSource)) {
  findings.push('DictItemTab AppTable must include the shared system-list__table class.');
}

if (!/<GovernanceSummaryBar[\s\S]*?className="dict-page__governance-bar"/.test(dictPageSource)) {
  findings.push('DictPage must render the shared GovernanceSummaryBar above its table card.');
}

if (/<PageHeader/.test(dictPageSource)) {
  findings.push('DictPage must not render a page-level PageHeader title.');
}

if (
  /<Card\s+className="[^"]*dict-page__table-card[^"]*"[\s\S]*?<GovernanceSummaryBar/.test(
    dictPageSource,
  )
) {
  findings.push('DictPage must not nest GovernanceSummaryBar inside the table card.');
}

if (/dict-workbench__context-card/.test(dictTypeTabSource)) {
  findings.push('DictTypeTab must not render a bottom governance/context card.');
}

if (
  !/<GovernanceSummaryBar[\s\S]*?className="setting-page__governance-bar"/.test(
    settingGroupPageSource,
  )
) {
  findings.push('SettingGroupPage must use the shared GovernanceSummaryBar for governance summary.');
}

if (/<PageHeader/.test(settingGroupPageSource)) {
  findings.push('SettingGroupPage must not render a page-level PageHeader title.');
}

if (/setting-page__overview(?:-|_)/.test(settingGroupPageSource)) {
  findings.push('SettingGroupPage must not use legacy setting-page__overview* styles.');
}

if (/setting-page__overview(?:-|_)/.test(globalSource) || /setting-page__overview(?:-|_)/.test(listPageSource)) {
  findings.push('Legacy setting-page__overview* CSS is forbidden; use GovernanceSummaryBar.');
}

if (/\.setting-page\s+\.system-page-hero/i.test(globalSource + listPageSource)) {
  findings.push('Setting pages must not use page-specific system-page-hero overrides.');
}

if (/\.dict-page__governance-bar\s+\.arco-card-body/i.test(globalSource + listPageSource)) {
  findings.push('Dict governance bar must not define card-body overrides.');
}

const tableHeaderRule = globalSource.match(/(?:^|\n)\.arco-table-th\s*\{[\s\S]*?\n\}/)?.[0] || '';
if (!tableHeaderRule) {
  findings.push('Missing CSS block: .arco-table-th');
} else {
  if (!/background\s*:\s*var\(--panel-muted\)\s*;/i.test(tableHeaderRule)) {
    findings.push(
      '.arco-table-th must use neutral panel-muted background, not theme-tinted color-mix.',
    );
  }
  if (/background\s*:[^;]*color-mix\([^;]*var\(--brand-primary\)/i.test(tableHeaderRule)) {
    findings.push('.arco-table-th must not tint table headers with brand-primary color-mix.');
  }
}

const appTableContainerBlock = requireBlock(
  globalSource,
  '.app-table .arco-table-container',
  findings,
);
if (appTableContainerBlock) {
  if (!hasDeclaration(appTableContainerBlock, 'border-radius', 'var\\(--radius-md\\)')) {
    findings.push('.app-table .arco-table-container must use radius-md.');
  }
}

const fixedColumnShadowBlock = requireBlock(
  globalSource,
  '.app-table .arco-table-col-fixed-left-last::after',
  findings,
);
if (
  fixedColumnShadowBlock &&
  !/box-shadow\s*:\s*none\s*!important\s*;/i.test(fixedColumnShadowBlock)
) {
  findings.push(
    'AppTable fixed-column shadow must be disabled to avoid gradient-like table borders.',
  );
}

const globalTabSelectors = [
  '.arco-tabs-header-nav::before',
  '.arco-tabs-header-nav-rounded .arco-tabs-header-title',
  '.arco-tabs-header-nav-rounded .arco-tabs-header-title-active',
];

for (const selector of globalTabSelectors) {
  const block = requireBlock(globalSource, selector, findings);
  if (!block) {
    continue;
  }
  for (const line of collectBorderLines(block)) {
    if (/color-mix\(/i.test(line)) {
      findings.push(`${selector} uses mixed/gradient-like border styling: ${line}`);
    }
    if (/box-shadow\s*:\s*inset/i.test(line)) {
      findings.push(`${selector} uses inset border styling: ${line}`);
    }
  }
}

if (findings.length > 0) {
  console.error('Shell visual contract failed:');
  for (const finding of findings) {
    console.error(`- ${finding}`);
  }
  process.exit(1);
}

console.log('Shell visual contract passed.');
