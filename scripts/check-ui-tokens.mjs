#!/usr/bin/env node

/**
 * UI Token 一致性检查脚本
 * 确保所有 UI 代码使用 Pantheon 设计 token，而非 Arco 原始 token
 *
 * 检查项：
 * 1. Arco 原始 token 使用（--color-text-1、--color-border-2 等）
 * 2. 禁止模式：radial-gradient、大面积渐变、非标准字重
 * 3. 裸调用 Modal.confirm（应使用平台封装）
 * 4. 硬编码颜色值（#hex、rgb()）
 */

import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import { scanTree } from './harness/scan-tree.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = resolve(__dirname, '..');

const colors = {
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  green: '\x1b[32m',
  blue: '\x1b[34m',
  reset: '\x1b[0m',
};

function log(level, message) {
  const prefix = {
    error: `${colors.red}✗${colors.reset}`,
    warn: `${colors.yellow}⚠${colors.reset}`,
    info: `${colors.blue}ℹ${colors.reset}`,
    success: `${colors.green}✓${colors.reset}`,
  };
  console.log(`${prefix[level]} ${message}`);
}

// Scans with a plain Node walk (scripts/harness/scan-tree.mjs) instead of
// ripgrep: no external binary, no CI install step, and JS regexes support the
// look-ahead the gradient rule needs. A scan root that cannot be read is a
// failure, not a clean result.
function runPattern(pattern, paths, description, extensions = null) {
  const { matches, missing } = scanTree({ root: projectRoot, dirs: paths, pattern, extensions });

  if (missing.length > 0) {
    log('error', `Pattern check could not run: ${description} (unreadable: ${missing.join(', ')})`);
    process.exit(2);
  }

  if (matches.length === 0) {
    return [];
  }

  const lines = matches.map((match) => `${match.file}:${match.line}:${match.text}`);
  return [{
    description,
    count: lines.length,
    lines: lines.slice(0, 8),
    hasMore: lines.length > 8,
  }];
}

// Replaces rg's `-t style -t typescript -t tsx` type filters.
const UI_STYLE_EXTENSIONS = new Set(['.css', '.scss', '.less', '.ts', '.tsx']);
const TS_EXTENSIONS = new Set(['.ts', '.tsx']);

function checkArcoRawTokens() {
  log('info', 'Checking Arco raw token usage...');

  // Arco Design 原始 token 模式
  const patterns = [
    /--color-text-[0-9]/,
    /--color-border-[0-9]/,
    /--color-fill-[0-9]/,
    /--color-bg-[0-9]/,
  ];

  const findings = [];

  for (const pattern of patterns) {
    const result = runPattern(
      pattern,
      ['frontend/src/'],
      `Arco raw token ${pattern.source} (should use Pantheon token)`,
      UI_STYLE_EXTENSIONS
    );
    findings.push(...result);
  }

  return findings;
}

function checkForbiddenPatterns() {
  log('info', 'Checking UI forbidden patterns...');

  const patterns = [
    {
      pattern: /radial-gradient/,
      desc: 'radial-gradient usage (forbidden decoration pattern)',
    },
    {
      pattern: /font-weight:\s*(650|620)/,
      desc: 'Non-standard font-weight (should use 400/500/600/700)',
    },
    {
      pattern: /linear-gradient.*background.*(?!button)/,
      desc: 'Large-area gradient background (forbidden except buttons)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runPattern(
      pattern,
      ['frontend/src/'],
      desc,
      UI_STYLE_EXTENSIONS
    );
    findings.push(...result);
  }

  return findings;
}

function checkModalDirectUsage() {
  log('info', 'Checking Modal.confirm direct usage...');

  return runPattern(
    /Modal\.(confirm|success|error|info|warning)/,
    ['frontend/src/modules/', 'frontend/src/components/'],
    'Modal.confirm direct usage (should use platform wrapper)',
    TS_EXTENSIONS
  );
}

function checkHardcodedColors() {
  log('info', 'Checking hardcoded color values...');

  const patterns = [
    {
      pattern: /color:\s*#[0-9a-fA-F]{3,6}/,
      desc: 'Hardcoded hex color (should use design token)',
    },
    {
      pattern: /background:\s*#[0-9a-fA-F]{3,6}/,
      desc: 'Hardcoded hex background (should use design token)',
    },
    {
      pattern: /rgb\(/,
      desc: 'Hardcoded rgb() color (should use design token)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runPattern(
      pattern,
      ['frontend/src/modules/', 'frontend/src/components/'],
      desc,
      UI_STYLE_EXTENSIONS
    );
    findings.push(...result);
  }

  return findings;
}

function checkInlineStyles() {
  log('info', 'Checking inline style usage...');

  return runPattern(
    /style=\{\{.*color.*\}\}/,
    ['frontend/src/modules/'],
    'Inline style with color (prefer CSS class with token)',
    TS_EXTENSIONS
  );
}

function main() {
  console.log(`\n${colors.blue}=== Pantheon UI Token Consistency Check ===${colors.reset}\n`);

  const allFindings = [
    ...checkArcoRawTokens(),
    ...checkForbiddenPatterns(),
    ...checkModalDirectUsage(),
    ...checkHardcodedColors(),
    ...checkInlineStyles(),
  ];

  console.log('');

  if (allFindings.length === 0) {
    log('success', 'No UI token violations found');
    console.log('');
    process.exit(0);
  }

  // 输出 findings
  let errorCount = 0;
  let warnCount = 0;

  for (const finding of allFindings) {
    // Arco raw token 和禁止模式是错误级
    const isError = finding.description.includes('Arco raw token') ||
                    finding.description.includes('forbidden') ||
                    finding.description.includes('Non-standard font-weight');

    if (isError) {
      log('error', `${finding.description} (${finding.count} occurrences)`);
      errorCount += finding.count;
    } else {
      log('warn', `${finding.description} (${finding.count} occurrences)`);
      warnCount += finding.count;
    }

    for (const line of finding.lines) {
      console.log(`  ${colors.yellow}${line}${colors.reset}`);
    }

    if (finding.hasMore) {
      console.log(`  ${colors.blue}... and ${finding.count - 8} more${colors.reset}`);
    }

    console.log('');
  }

  // 总结
  console.log(`${colors.blue}Summary:${colors.reset}`);
  console.log(`  Errors: ${errorCount}`);
  console.log(`  Warnings: ${warnCount}`);
  console.log('');

  // Fix guidance
  if (errorCount > 0) {
    console.log(`${colors.yellow}Fix:${colors.reset}`);
    console.log('  - Replace Arco tokens with Pantheon tokens (--p-text-*, --p-bg-*, etc.)');
    console.log('  - Remove forbidden patterns (radial-gradient, non-standard font-weight)');
    console.log('  - Use CSS classes with design tokens instead of inline styles');
    console.log('');
  }

  if (errorCount > 0) {
    log('error', 'UI token consistency check failed');
    console.log('');
    process.exit(1);
  }

  if (warnCount > 0) {
    log('warn', 'UI token consistency check passed with warnings');
    console.log('');
    process.exit(0);
  }
}

main();
