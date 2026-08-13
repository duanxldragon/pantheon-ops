#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const DEFAULT_ROOT = process.cwd();
const DOC_ROOTS = ['docs', 'architecture', 'patterns'];
const REQUIRED_FIELDS = ['title', 'doc_type', 'layer', 'status', 'updated_at'];
const DOC_TYPES_REQUIRING_CONTRACTS = new Set(['Design', 'Assessment', 'Remediation', 'Acceptance']);
const ALLOWED_DOC_TYPES = new Set([
  'Contract',
  'Design',
  'Assessment',
  'Remediation',
  'Acceptance',
  'Method',
  'Playbook',
  'Policy',
  'Guide',
  'Reference',
]);
const ALLOWED_STATUSES = new Set(['Draft', 'Active', 'Approved', 'Superseded', 'Archived']);

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') options.json = true;
    else if (arg === '--strict') options.strict = true;
    else if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) throw new Error('--root requires a path');
      options.root = path.resolve(value);
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-doc-frontmatter.mjs [--json] [--strict] [--root <path>]');
}

function walkMarkdownFiles(dirPath) {
  if (!fs.existsSync(dirPath)) return [];
  const files = [];
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const fullPath = path.join(dirPath, entry.name);
    if (entry.isDirectory()) files.push(...walkMarkdownFiles(fullPath));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(fullPath);
  }
  return files;
}

export function parseFrontmatter(source) {
  const lines = source.split(/\r?\n/);
  if (lines[0] !== '---') {
    return { hasFrontmatter: false, data: null, body: source };
  }

  const closingIndex = lines.findIndex((line, index) => index > 0 && line === '---');
  if (closingIndex === -1) {
    throw new Error('frontmatter block is not closed');
  }

  const data = {};
  let currentArrayKey = null;
  for (const line of lines.slice(1, closingIndex)) {
    if (!line.trim()) continue;
    const trimmedLine = line.trimStart();
    if (trimmedLine.startsWith('- ')) {
      if (!currentArrayKey) throw new Error(`array item found before array key: ${line}`);
      data[currentArrayKey].push(trimmedLine.slice(2).trim());
      continue;
    }

    const colonIndex = line.indexOf(':');
    if (colonIndex <= 0) throw new Error(`unsupported frontmatter line: ${line}`);
    const key = line.slice(0, colonIndex).trim();
    if (!/^[A-Za-z0-9_]+$/.test(key)) throw new Error(`unsupported frontmatter key: ${key}`);
    const rawValue = line.slice(colonIndex + 1).trim();
    if (rawValue === '') {
      data[key] = [];
      currentArrayKey = key;
    } else {
      data[key] = rawValue;
      currentArrayKey = null;
    }
  }

  return {
    hasFrontmatter: true,
    data,
    body: lines.slice(closingIndex + 1).join('\n'),
  };
}

function isNonEmptyString(value) {
  return typeof value === 'string' && value.trim().length > 0;
}

function isNonEmptyArray(value) {
  return Array.isArray(value) && value.length > 0 && value.every(isNonEmptyString);
}

function hasLegacyMetadata(source) {
  return /^(Type:|Layer:|Status:|类型：|归属层：|状态：|更新时间：)/m.test(source);
}

function validateFrontmatter(file, data, root) {
  const relativePath = path.relative(root, file).replaceAll(path.sep, '/');
  const errors = [];
  const warnings = [];

  for (const field of REQUIRED_FIELDS) {
    if (!isNonEmptyString(data[field])) {
      errors.push(`frontmatter.${field} must be a non-empty string`);
    }
  }

  if (isNonEmptyString(data.doc_type) && !ALLOWED_DOC_TYPES.has(data.doc_type)) {
    errors.push(`frontmatter.doc_type must be one of: ${Array.from(ALLOWED_DOC_TYPES).join(', ')}`);
  }

  if (isNonEmptyString(data.status) && !ALLOWED_STATUSES.has(data.status)) {
    errors.push(`frontmatter.status must be one of: ${Array.from(ALLOWED_STATUSES).join(', ')}`);
  }

  if (DOC_TYPES_REQUIRING_CONTRACTS.has(data.doc_type) && !isNonEmptyArray(data.linked_contracts)) {
    errors.push('frontmatter.linked_contracts is required for this doc_type');
  }

  if (data.status === 'Superseded' && !isNonEmptyString(data.superseded_by)) {
    errors.push('frontmatter.superseded_by is required for Superseded documents');
  }

  return { file: relativePath, errors, warnings };
}

function validateFile(file, root) {
  const source = fs.readFileSync(file, 'utf8');
  const relativePath = path.relative(root, file).replaceAll(path.sep, '/');
  let parsed;
  try {
    parsed = parseFrontmatter(source);
  } catch (error) {
    return { file: relativePath, errors: [error.message], warnings: [] };
  }

  if (!parsed.hasFrontmatter) {
    return {
      file: relativePath,
      errors: [],
      warnings: hasLegacyMetadata(source) ? ['legacy metadata should migrate to YAML frontmatter'] : [],
    };
  }

  return validateFrontmatter(file, parsed.data, root);
}

function scan(root) {
  return DOC_ROOTS
    .flatMap((docRoot) => walkMarkdownFiles(path.join(root, docRoot)))
    .sort((a, b) => a.localeCompare(b))
    .map((file) => validateFile(file, root))
    .filter((result) => result.errors.length > 0 || result.warnings.length > 0);
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  if (options.help) return printHelp(), 0;

  const results = scan(options.root);
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          findingCount: errorCount,
          warningCount,
          results,
        },
        null,
        2,
      ),
    );
  } else {
    console.log(`Doc frontmatter check (${options.strict ? 'strict' : 'report-only'}): ${errorCount} error(s), ${warningCount} warning(s)`);
    for (const result of results) {
      console.log('');
      console.log(result.errors.length === 0 ? `[WARN] ${result.file}` : `[FAIL] ${result.file}`);
      for (const error of result.errors) console.log(`  error: ${error}`);
      for (const warning of result.warnings) console.log(`  warning: ${warning}`);
    }
  }

  return options.strict && errorCount > 0 ? 1 : 0;
}

process.exitCode = main();
