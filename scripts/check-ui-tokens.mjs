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

import { execSync } from 'child_process';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

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

function runRipgrep(pattern, paths, description, extraFlags = '') {
  const findings = [];

  try {
    const cmd = `rg -n --no-heading --color=never ${extraFlags} '${pattern}' ${paths.join(' ')}`;
    const output = execSync(cmd, {
      cwd: projectRoot,
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe']
    });

    if (output.trim()) {
      const lines = output.trim().split('\n');
      findings.push({
        description,
        count: lines.length,
        lines: lines.slice(0, 8),
        hasMore: lines.length > 8,
      });
    }
  } catch (error) {
    if (error.status !== 1) {
      log('warn', `Pattern check failed: ${description}`);
    }
  }

  return findings;
}

function checkArcoRawTokens() {
  log('info', 'Checking Arco raw token usage...');

  // Arco Design 原始 token 模式
  const patterns = [
    '--color-text-[0-9]',
    '--color-border-[0-9]',
    '--color-fill-[0-9]',
    '--color-bg-[0-9]',
  ];

  const findings = [];

  for (const pattern of patterns) {
    const result = runRipgrep(
      pattern,
      ['frontend/src/'],
      `Arco raw token ${pattern} (should use Pantheon token)`,
      '--type-add "style:*.{css,scss,less}" -t style -t typescript -t tsx'
    );
    findings.push(...result);
  }

  return findings;
}

function checkForbiddenPatterns() {
  log('info', 'Checking UI forbidden patterns...');

  const patterns = [
    {
      pattern: 'radial-gradient',
      desc: 'radial-gradient usage (forbidden decoration pattern)',
    },
    {
      pattern: 'font-weight:\\s*(650|620)',
      desc: 'Non-standard font-weight (should use 400/500/600/700)',
    },
    {
      pattern: 'linear-gradient.*background.*(?!button)',
      desc: 'Large-area gradient background (forbidden except buttons)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runRipgrep(
      pattern,
      ['frontend/src/'],
      desc,
      '--type-add "style:*.{css,scss,less,tsx,ts}" -t style -t typescript -t tsx'
    );
    findings.push(...result);
  }

  return findings;
}

function checkModalDirectUsage() {
  log('info', 'Checking Modal.confirm direct usage...');

  return runRipgrep(
    'Modal\\.(confirm|success|error|info|warning)',
    ['frontend/src/modules/', 'frontend/src/components/'],
    'Modal.confirm direct usage (should use platform wrapper)',
    '-t typescript -t tsx'
  );
}

function checkHardcodedColors() {
  log('info', 'Checking hardcoded color values...');

  const patterns = [
    {
      pattern: 'color:\\s*#[0-9a-fA-F]{3,6}',
      desc: 'Hardcoded hex color (should use design token)',
    },
    {
      pattern: 'background:\\s*#[0-9a-fA-F]{3,6}',
      desc: 'Hardcoded hex background (should use design token)',
    },
    {
      pattern: 'rgb\\(',
      desc: 'Hardcoded rgb() color (should use design token)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runRipgrep(
      pattern,
      ['frontend/src/modules/', 'frontend/src/components/'],
      desc,
      '--type-add "style:*.{css,scss,less,tsx,ts}" -t style -t typescript -t tsx'
    );
    findings.push(...result);
  }

  return findings;
}

function checkInlineStyles() {
  log('info', 'Checking inline style usage...');

  return runRipgrep(
    'style={{.*color.*}}',
    ['frontend/src/modules/'],
    'Inline style with color (prefer CSS class with token)',
    '-t typescript -t tsx'
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
