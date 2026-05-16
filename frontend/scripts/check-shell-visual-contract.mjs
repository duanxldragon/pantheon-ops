import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const currentFilePath = fileURLToPath(import.meta.url);
const frontendRoot = path.resolve(path.dirname(currentFilePath), '..');
const layoutCssPath = path.join(frontendRoot, 'src', 'core', 'layout', 'index.css');
const globalCssPath = path.join(frontendRoot, 'src', 'index.css');
const listPageCssPath = path.join(frontendRoot, 'src', 'core', 'styles', 'list-page.css');
const userListPath = path.join(frontendRoot, 'src', 'modules', 'system', 'user', 'UserList.tsx');
const source = fs.readFileSync(layoutCssPath, 'utf8');
const globalSource = fs.readFileSync(globalCssPath, 'utf8');
const listPageSource = fs.readFileSync(listPageCssPath, 'utf8');
const userListSource = fs.readFileSync(userListPath, 'utf8');
const dictTypeTabSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'dict', 'DictTypeTab.tsx'),
  'utf8',
);
const dictItemTabSource = fs.readFileSync(
  path.join(frontendRoot, 'src', 'modules', 'system', 'dict', 'DictItemTab.tsx'),
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
    new RegExp(`(?:^|\\n)${escapedSelector}\\s*\\{([\\s\\S]*?)\\n\\}`, 'i'),
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
  const pattern = new RegExp(`${property}\\s*:\\s*${expectedValue}\\s*;`, 'i');
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

const findings = [];

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
  '.system-list__table-card .arco-card-body',
  findings,
);
if (systemTableCardBlock) {
  if (!hasDeclaration(systemTableCardBlock, 'padding', 'var\\(--shell-table-card-padding\\)')) {
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

const listHeaderActionsBlock = requireBlock(listPageSource, '.list-header-actions', findings);
if (
  listHeaderActionsBlock &&
  !hasDeclaration(listHeaderActionsBlock, 'gap', 'var\\(--shell-list-actions-gap\\)')
) {
  findings.push('.list-header-actions must use --shell-list-actions-gap.');
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
if (
  appDialogInputNumberControlBlock &&
  !hasDeclaration(appDialogInputNumberControlBlock, 'border', '1px solid var\\(--panel-border-strong\\)')
) {
  findings.push('.app-dialog InputNumber outer control must render one shared border.');
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

const nestedInputResetBlock = requireBlock(
  globalSource,
  '.arco-input-inner-wrapper .arco-input',
  findings,
);
if (nestedInputResetBlock) {
  for (const [property, expectedValue] of [
    ['min-height', 'auto'],
    ['border', '0'],
    ['background', 'transparent'],
    ['box-shadow', 'none'],
    ['outline', '0'],
  ]) {
    if (!hasDeclaration(nestedInputResetBlock, property, expectedValue)) {
      findings.push(`Nested Arco input controls must reset ${property}: ${expectedValue}.`);
    }
  }
}

const inputNumberInnerBlock = requireBlock(
  globalSource,
  '.arco-input-number .arco-input-inner-wrapper',
  findings,
);
if (inputNumberInnerBlock) {
  if (!hasDeclaration(inputNumberInnerBlock, 'border', '0')) {
    findings.push(
      'InputNumber inner wrapper must not draw a second border; the outer .arco-input-number owns it.',
    );
  }
  if (!hasDeclaration(inputNumberInnerBlock, 'box-shadow', 'none')) {
    findings.push('InputNumber inner wrapper must not draw a second focus ring.');
  }
}

for (const selectorList of globalSource.match(/[^{}]*\.arco-input-number\s+\.arco-input-inner-wrapper[^{}]*\{[^{}]*\}/g) ?? []) {
  const body = selectorList.slice(selectorList.indexOf('{') + 1, selectorList.lastIndexOf('}'));
  const nonNoneBoxShadow = Array.from(body.matchAll(/\bbox-shadow\s*:\s*([^;]+);/gi)).some(
    (match) => match[1].trim().toLowerCase() !== 'none',
  );
  if (
    /\bborder-color\s*:/i.test(body) ||
    nonNoneBoxShadow
  ) {
    findings.push(
      'InputNumber inner wrapper must not be part of the bordered control group; the outer .arco-input-number owns the border.',
    );
  }
}

const nestedInputFocusBlock = requireBlock(
  globalSource,
  '.arco-input-inner-wrapper .arco-input:focus',
  findings,
);
if (nestedInputFocusBlock) {
  if (!hasDeclaration(nestedInputFocusBlock, 'border', '0')) {
    findings.push('Nested Arco input focus must keep border: 0.');
  }
  if (!hasDeclaration(nestedInputFocusBlock, 'box-shadow', 'none')) {
    findings.push('Nested Arco input focus must not draw a second focus ring.');
  }
  if (!hasDeclaration(nestedInputFocusBlock, 'outline', '0')) {
    findings.push('Nested Arco input focus must not draw a second outline.');
  }
}

if (/(?:^|\n)\s*\.arco-input:focus\s*,/i.test(globalSource)) {
  findings.push(
    'Bare .arco-input:focus must not own the global focus ring; the outer input wrapper owns it.',
  );
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
