#!/usr/bin/env node
/**
 * check-coverage.mjs — Go 测试覆盖率阈值门禁
 *
 * 解析 `go tool cover -func` 输出，校验总覆盖率与各包覆盖率是否达到阈值。
 *
 * 用法:
 *   node scripts/harness/check-coverage.mjs <coverage.txt> --threshold <number>
 *   node scripts/harness/check-coverage.mjs <coverage-summary.json> --format json --threshold <number>
 *
 * 参数:
 *   coverage.txt   `go tool cover -func=coverage.out` 的输出文件
 *   --format F     输入格式：go（默认，cover -func 文本）或 json（istanbul/vitest coverage-summary.json）
 *   --threshold N  覆盖率阈值（百分比，0-100）。总覆盖率低于该值则 exit 1。
 *
 * 退出码:
 *   0  总覆盖率 >= threshold
 *   1  参数错误、文件不可读、或总覆盖率 < threshold
 *
 * 说明:
 *   输入每行格式: `path/file.go:LINE:\tFUNC\tCOVER%`，最后一行为
 *   `total:\t(statements)\tCOVER%`。包覆盖率按文件路径目录分组求平均，仅用于报告。
 */

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-coverage.mjs <coverage.txt> --threshold <number>

Arguments:
  coverage.txt      Output of 'go tool cover -func=coverage.out'
  --threshold N     Coverage threshold percentage (0-100)

Example:
  node scripts/harness/check-coverage.mjs backend/coverage.txt --threshold 11`);
}

function parseArgs(argv) {
  const options = { file: null, threshold: null, format: 'go', help: false };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--threshold') {
      const value = argv[++i];
      if (value === undefined || Number.isNaN(Number(value))) {
        throw new Error('--threshold requires a numeric value');
      }
      options.threshold = Number(value);
    } else if (arg === '--format') {
      const value = argv[++i];
      if (value !== 'go' && value !== 'json') {
        throw new Error("--format must be 'go' or 'json'");
      }
      options.format = value;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg.startsWith('--')) {
      throw new Error(`Unknown argument: ${arg}`);
    } else if (options.file === null) {
      options.file = arg;
    } else {
      throw new Error(`Unexpected positional argument: ${arg}`);
    }
  }

  return options;
}

/** Parse a single cover -func line into { file, func, cover } or null. */
function parseLine(line) {
  // Format: path/file.go:LINE:\tFUNC\tCOVER%
  const match = line.match(/^(.+?:\d+:)\t(.+?)\t([\d.]+)%\s*$/);
  if (!match) {
    return null;
  }
  return { file: match[1], func: match[2], cover: Number(match[3]) };
}

/** Parse the total line: `total:\t(statements)\tCOVER%` → percentage number. */
function parseTotal(line) {
  const match = line.match(/^total:\s*\(statements\)\s*([\d.]+)%\s*$/);
  return match ? Number(match[1]) : null;
}

/** Derive a package key from a cover -func file path (strip :LINE:, take dir). */
function packageOf(fileRef) {
  const withoutLine = fileRef.replace(/:\d+:$/, '');
  const dir = path.posix.dirname(withoutLine.replaceAll('\\', '/'));
  return dir === '.' ? '(root)' : dir;
}

/** Check an istanbul/vitest coverage-summary.json ({ total: { lines: { pct }, ... } }). */
function checkJsonSummary(content, options) {
  let summary;
  try {
    summary = JSON.parse(content);
  } catch (error) {
    console.error(`Error: cannot parse '${options.file}' as JSON: ${error.message}`);
    return 1;
  }

  const total = summary?.total;
  const pct = total?.lines?.pct;
  if (typeof pct !== 'number') {
    console.error(`Error: no 'total.lines.pct' found in '${options.file}'. Is it a coverage-summary.json?`);
    return 1;
  }

  console.log(`Coverage report (threshold: ${options.threshold}%)`);
  for (const metric of ['lines', 'statements', 'functions', 'branches']) {
    const value = total?.[metric]?.pct;
    if (typeof value === 'number') {
      console.log(`  ${metric}: ${value.toFixed(1)}%`);
    }
  }
  console.log('');

  if (pct < options.threshold) {
    console.error(`FAIL: total line coverage ${pct.toFixed(1)}% is below threshold ${options.threshold}%`);
    return 1;
  }

  console.log(`OK: total line coverage ${pct.toFixed(1)}% meets threshold ${options.threshold}%`);
  return 0;
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(`Error: ${error.message}`);
    return 1;
  }

  if (options.help) {
    printHelp();
    return 0;
  }

  if (options.file === null) {
    console.error('Error: coverage file path is required (see --help)');
    return 1;
  }
  if (options.threshold === null) {
    console.error('Error: --threshold is required (see --help)');
    return 1;
  }

  let content;
  try {
    content = fs.readFileSync(options.file, 'utf8');
  } catch (error) {
    console.error(`Error: cannot read coverage file '${options.file}': ${error.message}`);
    return 1;
  }

  if (options.format === 'json') {
    return checkJsonSummary(content, options);
  }

  const lines = content.split(/\r?\n/).filter((line) => line.trim().length > 0);

  let total = null;
  const packageBuckets = new Map(); // pkg -> { sum, count }

  for (const line of lines) {
    const totalValue = parseTotal(line);
    if (totalValue !== null) {
      total = totalValue;
      continue;
    }

    const entry = parseLine(line);
    if (entry === null) {
      continue;
    }

    const pkg = packageOf(entry.file);
    if (!packageBuckets.has(pkg)) {
      packageBuckets.set(pkg, { sum: 0, count: 0 });
    }
    const bucket = packageBuckets.get(pkg);
    bucket.sum += entry.cover;
    bucket.count += 1;
  }

  if (total === null) {
    console.error(`Error: no 'total:' line found in '${options.file}'. Is it 'go tool cover -func' output?`);
    return 1;
  }

  // Per-package report (sorted by coverage ascending so weakest show first).
  const packageRows = [...packageBuckets.entries()]
    .map(([pkg, { sum, count }]) => ({ pkg, avg: sum / count, files: count }))
    .sort((a, b) => a.avg - b.avg);

  console.log(`Coverage report (threshold: ${options.threshold}%)`);
  console.log(`  total: ${total.toFixed(1)}%`);
  console.log('');
  console.log('  Per-package averages (ascending):');
  for (const row of packageRows) {
    const marker = row.avg < options.threshold ? '  (below threshold)' : '';
    console.log(`    ${row.avg.toFixed(1).padStart(6)}%  ${row.pkg} (${row.files} funcs)${marker}`);
  }
  console.log('');

  if (total < options.threshold) {
    console.error(
      `FAIL: total coverage ${total.toFixed(1)}% is below threshold ${options.threshold}%`,
    );
    return 1;
  }

  console.log(`OK: total coverage ${total.toFixed(1)}% meets threshold ${options.threshold}%`);
  return 0;
}

process.exitCode = main();
