import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

const DEFAULT_THRESHOLD = 3;
const DEFAULT_MINIMUM_BLOCK_LINES = 21;

const DEFAULT_CONFIG = {
  includeRoots: ['backend', 'frontend/src', 'frontend/scripts', 'scripts', 'tests', '.github/workflows'],
  supportedExtensions: new Set(['.go', '.ts', '.tsx', '.js', '.mjs', '.cjs', '.yml', '.yaml']),
  excludedDirectoryFragments: [
    '/node_modules/',
    '/dist/',
    '/.git/',
    '/.codegraph/',
    '/scripts/harness/',
    '/frontend/src/modules/lowcode/generated/',
    '/frontend/src/modules/business/',
    '/schema/generated/',
    '/tests/fixtures/',
  ],
  excludedFiles: new Set([
    'backend/modules/business/generated_registry.go',
    'backend/modules/system/iam/menu/generated_component_registry.go',
  ]),
  generatedFilePattern: /\.generated\./i,
  minimumBlockLines: DEFAULT_MINIMUM_BLOCK_LINES,
};

function normalizePath(value) {
  return String(value).replaceAll('\\', '/');
}

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

function shouldExclude(relativePath, config) {
  const normalized = `/${normalizePath(relativePath)}`;
  if (config.excludedFiles.has(normalized.slice(1))) {
    return true;
  }
  if (config.generatedFilePattern.test(path.basename(normalized))) {
    return true;
  }
  return config.excludedDirectoryFragments.some((fragment) => normalized.includes(fragment));
}

function walkFiles(rootDir, relativeDir, config, files) {
  const absoluteDir = path.join(rootDir, relativeDir);
  if (!fs.existsSync(absoluteDir)) {
    return;
  }
  const stat = fs.statSync(absoluteDir);
  if (stat.isFile()) {
    const extension = path.extname(relativeDir);
    const normalized = normalizePath(relativeDir);
    if (!config.supportedExtensions.has(extension) || shouldExclude(normalized, config)) {
      return;
    }
    files.push(normalized);
    return;
  }

  for (const entry of fs.readdirSync(absoluteDir, { withFileTypes: true })) {
    const childRelative = normalizePath(path.join(relativeDir, entry.name));
    if (shouldExclude(childRelative, config)) {
      continue;
    }
    if (entry.isDirectory()) {
      walkFiles(rootDir, childRelative, config, files);
      continue;
    }
    if (!config.supportedExtensions.has(path.extname(entry.name))) {
      continue;
    }
    files.push(childRelative);
  }
}

function normalizeLine(line) {
  return line.trim().replace(/\s+/g, ' ');
}

function isIgnorableLine(line) {
  const normalized = normalizeLine(line);
  if (normalized === '') {
    return true;
  }
  return /^(\/\/|\/\*|\*\/|\*|#)/.test(normalized);
}

function extractRelevantLines(content) {
  return content
    .split(/\r?\n/)
    .map((line, index) => ({ lineNumber: index + 1, text: normalizeLine(line) }))
    .filter((line) => !isIgnorableLine(line.text));
}

export function analyzeDuplication(rootDir = process.cwd(), overrides = {}) {
  const config = {
    ...DEFAULT_CONFIG,
    ...overrides,
    supportedExtensions: overrides.supportedExtensions ?? DEFAULT_CONFIG.supportedExtensions,
    excludedDirectoryFragments:
      overrides.excludedDirectoryFragments ?? DEFAULT_CONFIG.excludedDirectoryFragments,
    excludedFiles: overrides.excludedFiles ?? DEFAULT_CONFIG.excludedFiles,
    generatedFilePattern: overrides.generatedFilePattern ?? DEFAULT_CONFIG.generatedFilePattern,
  };

  const files = [];
  for (const relativeRoot of config.includeRoots) {
    walkFiles(rootDir, relativeRoot, config, files);
  }

  const uniqueFiles = [...new Set(files)].sort((left, right) => left.localeCompare(right));
  const fileReports = uniqueFiles.map((relativePath) => {
    const absolutePath = path.join(rootDir, relativePath);
    const content = fs.readFileSync(absolutePath, 'utf8');
    const relevantLines = extractRelevantLines(content);
    return {
      path: relativePath,
      relevantLines,
      totalLines: relevantLines.length,
    };
  });

  const windowOccurrences = new Map();
  for (let fileIndex = 0; fileIndex < fileReports.length; fileIndex += 1) {
    const report = fileReports[fileIndex];
    if (report.relevantLines.length < config.minimumBlockLines) {
      continue;
    }
    for (let start = 0; start <= report.relevantLines.length - config.minimumBlockLines; start += 1) {
      const block = report.relevantLines
        .slice(start, start + config.minimumBlockLines)
        .map((line) => line.text)
        .join('\n');
      const occurrences = windowOccurrences.get(block) ?? [];
      occurrences.push({ fileIndex, start });
      windowOccurrences.set(block, occurrences);
    }
  }

  const duplicatedLineSets = new Map();
  const duplicates = [];

  for (const [fingerprint, occurrences] of windowOccurrences.entries()) {
    if (occurrences.length < 2) {
      continue;
    }
    const duplicate = {
      blockLines: config.minimumBlockLines,
      sample: fingerprint.split('\n').slice(0, 2).join(' | '),
      occurrences: occurrences.map(({ fileIndex, start }) => {
        const report = fileReports[fileIndex];
        const matchedLines = report.relevantLines.slice(start, start + config.minimumBlockLines);
        const lineNumbers = matchedLines.map((line) => line.lineNumber);
        let duplicated = duplicatedLineSets.get(report.path);
        if (!duplicated) {
          duplicated = new Set();
          duplicatedLineSets.set(report.path, duplicated);
        }
        for (const lineNumber of lineNumbers) {
          duplicated.add(lineNumber);
        }
        return {
          file: report.path,
          startLine: lineNumbers[0],
          endLine: lineNumbers[lineNumbers.length - 1],
        };
      }),
    };
    duplicates.push(duplicate);
  }

  const totalLines = fileReports.reduce((sum, report) => sum + report.totalLines, 0);
  const duplicatedLines = [...duplicatedLineSets.values()].reduce((sum, set) => sum + set.size, 0);
  const percentage = totalLines === 0 ? 0 : Number(((duplicatedLines / totalLines) * 100).toFixed(2));

  duplicates.sort((left, right) => right.occurrences.length - left.occurrences.length);

  return {
    files: fileReports.map(({ path: file, totalLines: fileLines }) => ({ path: file, totalLines: fileLines })),
    totalLines,
    duplicatedLines,
    percentage,
    duplicates,
    minimumBlockLines: config.minimumBlockLines,
  };
}

export function evaluateDuplication(report, threshold = DEFAULT_THRESHOLD) {
  return {
    ok: report.percentage <= threshold,
    threshold,
  };
}

export function formatDuplicationReport(report, threshold = DEFAULT_THRESHOLD) {
  const evaluation = evaluateDuplication(report, threshold);
  const lines = [
    `Repository duplication: ${report.percentage.toFixed(2)}% (${report.duplicatedLines}/${report.totalLines} normalized lines)`,
    `Threshold: <= ${threshold.toFixed(2)}%`,
    `Scanned files: ${report.files.length}`,
    `Minimum block size: ${report.minimumBlockLines} lines`,
  ];

  if (report.duplicates.length > 0) {
    lines.push('');
    lines.push('Top duplicate blocks:');
    for (const duplicate of report.duplicates.slice(0, 10)) {
      lines.push(`- ${duplicate.sample}`);
      for (const occurrence of duplicate.occurrences.slice(0, 5)) {
        lines.push(`  ${occurrence.file}:${occurrence.startLine}-${occurrence.endLine}`);
      }
    }
  }

  if (!evaluation.ok) {
    lines.push('');
    lines.push('Result: FAIL');
  } else {
    lines.push('');
    lines.push('Result: PASS');
  }

  return lines.join('\n');
}

function main() {
  const rootDir = path.resolve(readArg('--root', process.cwd()));
  const threshold = Number(readArg('--threshold', String(DEFAULT_THRESHOLD)));
  const minimumBlockLines = Number(readArg('--min-lines', String(DEFAULT_MINIMUM_BLOCK_LINES)));
  const json = process.argv.includes('--json');

  const report = analyzeDuplication(rootDir, { minimumBlockLines });
  const evaluation = evaluateDuplication(report, threshold);

  if (json) {
    console.log(JSON.stringify({ ...report, ...evaluation }, null, 2));
  } else {
    console.log(formatDuplicationReport(report, threshold));
  }

  if (!evaluation.ok) {
    process.exit(1);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  main();
}
