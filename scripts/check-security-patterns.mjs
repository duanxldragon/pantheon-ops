#!/usr/bin/env node

/**
 * 安全模式检查脚本
 * 扫描代码中的危险 API 使用，用于代码审查自动化门禁
 *
 * 检查项：
 * 1. Go 后端：Raw() / Exec() SQL 拼接风险
 * 2. React 前端：dangerouslySetInnerHTML XSS 风险
 * 3. Token 存储：localStorage 存储敏感 token
 * 4. 硬编码密钥：代码中的密码/密钥/凭证
 */

import fs from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';
import { scanTree } from './harness/scan-tree.mjs';
import { fingerprintOf, loadFingerprints, reviewByDate, saveFingerprints } from './harness/pattern-baseline.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = resolve(__dirname, '..');

// 颜色输出
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

const BASELINE_PATH = resolve(projectRoot, 'config', 'security-pattern-baseline.json');
const writeBaseline = process.argv.includes('--write-baseline');
const baseline = writeBaseline ? null : loadFingerprints(BASELINE_PATH);
const collectedFingerprints = [];
let baselineHits = 0;

// Scans with a plain Node walk (scripts/harness/scan-tree.mjs) instead of
// ripgrep: no external binary, no CI install step, and JS regexes support the
// look-ahead the session-fixation rule needs. A scan root that cannot be read
// is a failure, not a clean result.
function runPattern(pattern, paths, description, { extensions = null, excludeSuffixes = null } = {}) {
  const { matches, missing } = scanTree({ root: projectRoot, dirs: paths, pattern, extensions, excludeSuffixes });

  if (missing.length > 0) {
    log('error', `Pattern check could not run: ${description} (unreadable: ${missing.join(', ')})`);
    process.exit(2);
  }

  const fresh = [];
  for (const match of matches) {
    const fingerprint = fingerprintOf(description, match);
    if (writeBaseline) {
      collectedFingerprints.push(fingerprint);
    }
    if (baseline && baseline.has(fingerprint)) {
      baselineHits += 1;
      continue;
    }
    fresh.push(match);
  }

  if (fresh.length === 0) {
    return [];
  }

  const lines = fresh.map((match) => `${match.file}:${match.line}:${match.text}`);
  return [{
    description,
    count: lines.length,
    lines: lines.slice(0, 10), // 只显示前 10 条
    hasMore: lines.length > 10,
  }];
}

const GO_EXTENSIONS = new Set(['.go']);

// Test files build fixtures, not the production query surface; counting them
// would drown the 46 real hits in 310 fixture hits and make the baseline
// meaningless.
const GO_PROD = { extensions: GO_EXTENSIONS, excludeSuffixes: ['_test.go'] };

// baselineReviewBy reports the baseline's own review date so a stale acceptance
// list is visible in the green output, not only in the file.
function baselineReviewBy() {
  if (!fs.existsSync(BASELINE_PATH)) {
    return 'n/a';
  }
  try {
    return JSON.parse(fs.readFileSync(BASELINE_PATH, 'utf8')).reviewBy ?? 'n/a';
  } catch {
    return 'unreadable';
  }
}

function checkGoRawSQL() {
  log('info', 'Checking Go Raw/Exec SQL injection risks...');

  const patterns = [
    {
      pattern: /Raw\(/,
      desc: 'GORM Raw() usage (potential SQL injection if not parameterized)',
    },
    {
      pattern: /Exec\(/,
      desc: 'GORM Exec() usage (potential SQL injection if not parameterized)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runPattern(pattern, ['backend/'], desc, GO_PROD);
    findings.push(...result);
  }

  return findings;
}

function checkXSSRisks() {
  log('info', 'Checking XSS risks (dangerouslySetInnerHTML)...');

  return runPattern(
    /dangerouslySetInnerHTML/,
    ['frontend/src/'],
    'dangerouslySetInnerHTML usage (XSS risk if not sanitized)'
  );
}

function checkTokenStorage() {
  log('info', 'Checking insecure token storage...');

  return runPattern(
    /localStorage\.(setItem|getItem).*["'].*token/,
    ['frontend/src/'],
    'localStorage token storage (XSS can steal tokens, prefer httpOnly cookies)',
  );
}

function checkHardcodedSecrets() {
  log('info', 'Checking hardcoded secrets...');

  const patterns = [
    {
      pattern: /(password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}["']/,
      desc: 'Hardcoded password in code',
    },
    {
      pattern: /(api[_-]?key|apikey|access[_-]?key)\s*[:=]\s*["'][^"']{16,}["']/,
      desc: 'Hardcoded API key in code',
    },
    {
      pattern: /(secret|token)\s*[:=]\s*["'][a-zA-Z0-9+/=]{32,}["']/,
      desc: 'Hardcoded secret/token in code',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runPattern(
      pattern,
      ['backend/', 'frontend/src/'],
      desc
    );
    findings.push(...result);
  }

  return findings;
}

function checkSessionFixation() {
  log('info', 'Checking session fixation risks...');

  return runPattern(
    /SetCookie.*session.*(?!Secure|HttpOnly)/,
    ['backend/'],
    'Session cookie without Secure/HttpOnly flags',
    GO_PROD
  );
}

function main() {
  console.log(`\n${colors.blue}=== Pantheon Security Pattern Check ===${colors.reset}\n`);

  const allFindings = [
    ...checkGoRawSQL(),
    ...checkXSSRisks(),
    ...checkTokenStorage(),
    ...checkHardcodedSecrets(),
    ...checkSessionFixation(),
  ];

  if (writeBaseline) {
    const written = saveFingerprints(BASELINE_PATH, collectedFingerprints, reviewByDate());
    log('success', `baseline written: ${written} fingerprint(s) -> config/security-pattern-baseline.json`);
    console.log('');
    process.exit(0);
  }

  console.log('');

  if (allFindings.length === 0) {
    log(
      'success',
      baselineHits > 0
        ? `No new security pattern violations (${baselineHits} accepted by baseline, review by ${baselineReviewBy()})`
        : 'No security pattern violations found'
    );
    console.log('');
    process.exit(0);
  }

  // 输出 findings
  let errorCount = 0;
  let warnCount = 0;

  for (const finding of allFindings) {
    const isError = finding.description.includes('SQL injection') ||
                    finding.description.includes('Hardcoded');

    if (isError) {
      log('error', `${finding.description} (${finding.count} occurrences)`);
      errorCount += finding.count;
    } else {
      log('warn', `${finding.description} (${finding.count} occurrences)`);
      warnCount += finding.count;
    }

    // 显示前几条
    for (const line of finding.lines) {
      console.log(`  ${colors.yellow}${line}${colors.reset}`);
    }

    if (finding.hasMore) {
      console.log(`  ${colors.blue}... and ${finding.count - 10} more${colors.reset}`);
    }

    console.log('');
  }

  // 总结
  console.log(`${colors.blue}Summary:${colors.reset}`);
  console.log(`  Errors: ${errorCount}`);
  console.log(`  Warnings: ${warnCount}`);
  if (baselineHits > 0) {
    console.log(`  Accepted by baseline: ${baselineHits}`);
  }
  console.log('');

  // 错误级别问题阻断构建
  if (errorCount > 0) {
    log('error', 'Security pattern check failed');
    console.log('');
    process.exit(1);
  }

  // 警告级别问题不阻断，但提示
  if (warnCount > 0) {
    log('warn', 'Security pattern check passed with warnings');
    console.log('');
    process.exit(0);
  }
}

main();
