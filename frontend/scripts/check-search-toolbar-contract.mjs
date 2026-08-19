/**
 * SearchToolbar 交互契约门禁（DESIGN.md §7.2 列表页筛选区规范）。
 *
 * 2026-07 全部列表页已从 FilterPanel 表单栅格（显式搜索/重置按钮）迁移到
 * SearchToolbar（防抖关键词 + 即时下拉 + 弹层低频筛选 + 条件清空）。本门禁
 * 把该契约固化为机械检查，防止两类回退：
 *
 * A. 结构回退 —— 列表页重新引入旧筛选形态：
 *    - src/modules 内禁止 import FilterPanel（组件保留给 lowcode 模板过渡，
 *      业务页不得再使用）
 *    - 禁止 filter-panel__action-item（旧搜索/重置按钮容器）
 *    - SearchToolbar 的 keywordPlaceholder 必须走 t() i18n，不得内联字符串
 *    - 使用 SearchToolbar 且传 onKeywordChange 时必须同时传受控 keyword
 *    - 使用 SearchToolbar 必须提供 onClearAll（清空入口是契约的一部分）
 *
 * B. 样式回退 —— 工具栏控件脱离全局控件规则（2026-07-16 双边框/交互态
 *    覆盖两次事故的教训）：
 *    - index.css 中 .search-toolbar 作用域的 input/select/picker 选择器
 *      不得重声明 border/background（必须继承全局 .arco-* 控件规则）
 *    - .search-toolbar__keyword 的外层 group wrapper 必须保持 border: none
 *
 * Escape hatch：同行追加 `search-toolbar-allow: <rule-id>` 注释并在 PR body
 * 说明原因。
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const rootFlagIndex = process.argv.indexOf('--root');
// --root 供 tmpdir fixture 测试使用；默认锚定脚本所在的 frontend 目录。
const frontendRoot =
  rootFlagIndex >= 0 ? path.resolve(process.argv[rootFlagIndex + 1]) : path.resolve(scriptDir, '..');
const modulesRoot = path.join(frontendRoot, 'src', 'modules');
const indexCssPath = path.join(frontendRoot, 'src', 'index.css');

// lowcode 模板/生成器输出属于生成物契约，由 generator-quality-contract 测试覆盖。
const EXEMPT_SEGMENTS = [`${path.sep}lowcode${path.sep}templates${path.sep}`];

function walk(dir) {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(full));
    else out.push(full);
  }
  return out;
}

const findings = [];

function report(file, line, rule, contract) {
  findings.push({
    file: path.relative(frontendRoot, file).replaceAll(path.sep, '/'),
    line,
    rule,
    contract,
  });
}

function allowed(line, ruleId) {
  return line.includes(`search-toolbar-allow: ${ruleId}`);
}

// ---------------------------------------------------------------------------
// A. 结构契约（src/modules 下 .tsx/.ts）
// ---------------------------------------------------------------------------

const moduleFiles = walk(modulesRoot).filter(
  (file) =>
    ['.tsx', '.ts'].includes(path.extname(file)) &&
    !EXEMPT_SEGMENTS.some((segment) => file.includes(segment)),
);

for (const file of moduleFiles) {
  const source = fs.readFileSync(file, 'utf8');
  const lines = source.split(/\r?\n/);

  lines.forEach((line, index) => {
    if (/\bFilterPanel\b/.test(line) && !allowed(line, 'no-filter-panel')) {
      report(
        file,
        index + 1,
        'no-filter-panel',
        'List pages must use SearchToolbar; FilterPanel form grids are retired (DESIGN.md §7.2)',
      );
    }
    if (/filter-panel__action-item/.test(line) && !allowed(line, 'no-action-item')) {
      report(
        file,
        index + 1,
        'no-action-item',
        'Explicit search/reset button rows are retired; filters trigger queries directly',
      );
    }
    const placeholderMatch = line.match(/keywordPlaceholder\s*=\s*(?:\{\s*)?(['"`])/);
    if (placeholderMatch && !allowed(line, 'keyword-placeholder-i18n')) {
      report(
        file,
        index + 1,
        'keyword-placeholder-i18n',
        'keywordPlaceholder must be an i18n call t(...), not an inline string literal',
      );
    }
  });

  // 组件级断言：逐个 <SearchToolbar …/> 使用点检查必备 props。
  // 组件必有嵌套 JSX（inlineFilters 等），不能用非贪婪正则找 `/>`；
  // 按代码风格约定：闭合 `/>` 单独成行且与开标签同缩进。
  for (let index = 0; index < lines.length; index += 1) {
    const openMatch = lines[index].match(/^(\s*)<SearchToolbar\b/);
    if (!openMatch) continue;
    const indent = openMatch[1];
    let closeIndex = index;
    for (let cursor = index; cursor < lines.length; cursor += 1) {
      const trimmed = lines[cursor].trim();
      if (
        (cursor === index && /\/>\s*$/.test(lines[cursor])) ||
        (cursor > index && lines[cursor].startsWith(`${indent}/>`) && trimmed === '/>')
      ) {
        closeIndex = cursor;
        break;
      }
    }
    const props = lines.slice(index, closeIndex + 1).join('\n');
    const lineNumber = index + 1;
    const headLine = lines[index];
    if (
      props.includes('onKeywordChange') &&
      !/\bkeyword\s*=/.test(props) &&
      !allowed(headLine, 'controlled-keyword')
    ) {
      report(
        file,
        lineNumber,
        'controlled-keyword',
        'SearchToolbar with onKeywordChange must also pass controlled keyword= (clear-all must reset the input)',
      );
    }
    if (!props.includes('onClearAll') && !allowed(headLine, 'clear-all-required')) {
      report(
        file,
        lineNumber,
        'clear-all-required',
        'SearchToolbar must wire onClearAll — the conditional clear entry is part of the contract',
      );
    }
    index = closeIndex;
  }
}

// ---------------------------------------------------------------------------
// B. 样式契约（src/index.css 的 .search-toolbar 作用域）
// ---------------------------------------------------------------------------

const cssSource = fs.readFileSync(indexCssPath, 'utf8');

// 提取每个含 .search-toolbar 的规则块（selector -> body），容忍多选择器。
const rulePattern = /([^{}]+)\{([^{}]*)\}/g;
let cssRule;
let sawGroupWrapperReset = false;
while ((cssRule = rulePattern.exec(cssSource)) !== null) {
  const selector = cssRule[1].trim();
  const body = cssRule[2];
  if (!selector.includes('.search-toolbar')) continue;
  const lineNumber = cssSource.slice(0, cssRule.index).split(/\r?\n/).length;

  const targetsInputControl =
    /\.search-toolbar[^,{]*\.(arco-input-inner-wrapper|arco-select-view|arco-tree-select-view|arco-picker)\b/.test(
      selector,
    ) && !selector.includes('__popover');
  if (targetsInputControl) {
    if (/(?:^|;)\s*border\s*:/.test(body) && !body.includes('search-toolbar-allow: inherit-border')) {
      report(
        indexCssPath,
        lineNumber,
        'inherit-border',
        'Toolbar input/select/picker selectors must inherit border from global .arco-* control rules (do not re-declare)',
      );
    }
    if (
      /(?:^|;)\s*background\s*:/.test(body) &&
      !body.includes('search-toolbar-allow: inherit-background')
    ) {
      report(
        indexCssPath,
        lineNumber,
        'inherit-background',
        'Toolbar input/select/picker selectors must inherit background from global control rules',
      );
    }
  }

  if (
    selector.includes('.search-toolbar__keyword.arco-input-group-wrapper') &&
    /border\s*:\s*none/.test(body)
  ) {
    sawGroupWrapperReset = true;
  }
}

if (!sawGroupWrapperReset) {
  report(
    indexCssPath,
    1,
    'group-wrapper-layout-only',
    'The .search-toolbar__keyword.arco-input-group-wrapper rule must keep border: none (outer wrapper is layout-only; prevents double border)',
  );
}

// ---------------------------------------------------------------------------

if (findings.length === 0) {
  console.log(
    `SearchToolbar contract check: 0 finding(s) across ${moduleFiles.length} module file(s) + index.css`,
  );
  process.exitCode = 0;
} else {
  console.error(`SearchToolbar contract check: ${findings.length} finding(s)`);
  for (const finding of findings) {
    console.error(`finding: ${finding.file}:${finding.line} [${finding.rule}]`);
    console.error(`  contract: ${finding.contract}`);
  }
  console.error(
    '\nFix the violation or, when a rule genuinely does not apply, append a `search-toolbar-allow: <rule-id>` comment and justify it in the PR body.',
  );
  process.exitCode = 1;
}
