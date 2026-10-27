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

import { execSync } from 'child_process';
import { existsSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

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

function runRipgrep(pattern, paths, description) {
  const findings = [];

  try {
    const cmd = `rg -n --no-heading --color=never ${pattern} ${paths.join(' ')}`;
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
        lines: lines.slice(0, 10), // 只显示前 10 条
        hasMore: lines.length > 10,
      });
    }
  } catch (error) {
    // Exit code 1 means no matches found, which is good
    if (error.status !== 1) {
      log('warn', `Pattern check failed: ${description}`);
    }
  }

  return findings;
}

function checkGoRawSQL() {
  log('info', 'Checking Go Raw/Exec SQL injection risks...');

  const patterns = [
    {
      pattern: '"Raw\\("',
      desc: 'GORM Raw() usage (potential SQL injection if not parameterized)',
    },
    {
      pattern: '"Exec\\("',
      desc: 'GORM Exec() usage (potential SQL injection if not parameterized)',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runRipgrep(
      pattern,
      ['backend/'],
      desc
    );
    findings.push(...result);
  }

  return findings;
}

function checkXSSRisks() {
  log('info', 'Checking XSS risks (dangerouslySetInnerHTML)...');

  return runRipgrep(
    'dangerouslySetInnerHTML',
    ['frontend/src/'],
    'dangerouslySetInnerHTML usage (XSS risk if not sanitized)'
  );
}

function checkTokenStorage() {
  log('info', 'Checking insecure token storage...');

  return runRipgrep(
    'localStorage\\.(setItem|getItem).*["\'].*token',
    ['frontend/src/'],
    'localStorage token storage (XSS can steal tokens, prefer httpOnly cookies)'
  );
}

function checkHardcodedSecrets() {
  log('info', 'Checking hardcoded secrets...');

  const patterns = [
    {
      pattern: '(password|passwd|pwd)\\s*[:=]\\s*["\'][^"\']{8,}["\']',
      desc: 'Hardcoded password in code',
    },
    {
      pattern: '(api[_-]?key|apikey|access[_-]?key)\\s*[:=]\\s*["\'][^"\']{16,}["\']',
      desc: 'Hardcoded API key in code',
    },
    {
      pattern: '(secret|token)\\s*[:=]\\s*["\'][a-zA-Z0-9+/=]{32,}["\']',
      desc: 'Hardcoded secret/token in code',
    },
  ];

  const findings = [];

  for (const { pattern, desc } of patterns) {
    const result = runRipgrep(
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

  return runRipgrep(
    'SetCookie.*session.*(?!Secure|HttpOnly)',
    ['backend/'],
    'Session cookie without Secure/HttpOnly flags'
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

  console.log('');

  if (allFindings.length === 0) {
    log('success', 'No security pattern violations found');
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
