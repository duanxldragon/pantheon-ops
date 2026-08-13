#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const REPOSITORIES = ['pantheon-base', 'pantheon-ops'];
const DTO_LIKE_SUFFIXES = [
  'DTO',
  'Dto',
  'Resp',
  'Response',
  'Result',
  'View',
  'Option',
  'Tree',
  'Summary',
  'Overview',
  'Profile',
  'Workbench',
  'Policy',
  'Event',
  'Pack',
];
const SAFE_RETURN_TYPES = new Set([
  'nil',
  'error',
  'string',
  'bool',
  'int',
  'int64',
  'uint',
  'uint64',
  'float64',
  'any',
  'interface{}',
  'gin.H',
  'map[string]interface{}',
  'map[string]any',
]);

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-backend-dto-contract.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report suspicious backend handler DTO contract warnings and exit 0.
  Use --strict to exit 1 only when findings exist. Conservative DTO risks are warnings.

Examples:
  node scripts/harness/check-backend-dto-contract.mjs
  node scripts/harness/check-backend-dto-contract.mjs --json
  node scripts/harness/check-backend-dto-contract.mjs --strict`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--root') {
      const root = argv[index + 1];
      if (!root) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(root);
      index += 1;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function walkFiles(rootDir, extensions) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  const files = [];
  const stack = [rootDir];

  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });

    for (const entry of entries) {
      const entryPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        if (entry.name === 'vendor') {
          continue;
        }
        stack.push(entryPath);
      } else if (extensions.some((extension) => entry.name.endsWith(extension))) {
        files.push(entryPath);
      }
    }
  }

  return sortStrings(files);
}

function toRepoPath(root, filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function normalizeType(typeName) {
  return typeName
    .trim()
    .replace(/^\*/, '')
    .replace(/^\[\]/, '')
    .replace(/^map\[[^\]]+\]/, '')
    .replace(/^chan\s+/, '')
    .replace(/^<-chan\s+/, '')
    .trim();
}

function isDtoLike(typeName) {
  const normalized = normalizeType(typeName);
  if (SAFE_RETURN_TYPES.has(normalized)) {
    return true;
  }
  if (/^(?:\[\])?(?:string|bool|int|int64|uint|uint64|float64)$/.test(typeName.trim())) {
    return true;
  }
  return DTO_LIKE_SUFFIXES.some((suffix) => normalized.endsWith(suffix));
}

function splitTopLevelCsv(text) {
  const parts = [];
  let current = '';
  let depth = 0;

  for (const char of text) {
    if (char === '[' || char === '(' || char === '{') {
      depth += 1;
    } else if (char === ']' || char === ')' || char === '}') {
      depth -= 1;
    }

    if (char === ',' && depth === 0) {
      parts.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }

  if (current.trim()) {
    parts.push(current.trim());
  }

  return parts;
}

function parseReturnTypes(returnSignature) {
  const trimmed = returnSignature.trim();
  if (!trimmed) {
    return [];
  }
  if (trimmed.startsWith('(') && trimmed.endsWith(')')) {
    return splitTopLevelCsv(trimmed.slice(1, -1));
  }
  return [trimmed];
}

function parseServiceMethods(files) {
  const methods = new Map();
  const methodPattern = /func\s*\(\s*\w+\s+\*?(\w*Service)\s*\)\s*(\w+)\s*\([^)]*\)\s*([^{\n]+)/g;

  for (const filePath of files) {
    if (!filePath.endsWith('.go') || filePath.endsWith('_test.go')) {
      continue;
    }
    const content = fs.readFileSync(filePath, 'utf8');
    let match;
    while ((match = methodPattern.exec(content)) !== null) {
      const receiverType = match[1];
      const methodName = match[2];
      const returnTypes = parseReturnTypes(match[3]).filter((typeName) => normalizeType(typeName) !== 'error');
      methods.set(`${receiverType}.${methodName}`, {
        receiverType,
        methodName,
        returnTypes,
        file: filePath,
      });
    }
  }

  return methods;
}

function findServiceFields(handlerBody) {
  const fields = new Map();
  const fieldPattern = /(\w+)\s+\*?(\w*Service)\b/g;
  let match;
  while ((match = fieldPattern.exec(handlerBody)) !== null) {
    fields.set(match[1], match[2]);
  }
  return fields;
}

function findHandlerServiceFields(files) {
  const handlerServices = new Map();
  const typePattern = /type\s+(\w*Handler)\s+struct\s*\{([\s\S]*?)\}/g;

  for (const filePath of files) {
    const content = fs.readFileSync(filePath, 'utf8');
    let match;
    while ((match = typePattern.exec(content)) !== null) {
      handlerServices.set(match[1], findServiceFields(match[2]));
    }
  }

  return handlerServices;
}

function lineNumberAt(content, index) {
  return content.slice(0, index).split(/\r?\n/).length;
}

function scanHandlerFile(root, filePath, handlerServices, serviceMethods) {
  const content = fs.readFileSync(filePath, 'utf8');
  const findings = [];
  const warnings = [];
  const handlerPattern = /func\s*\(\s*(\w+)\s+\*?(\w*Handler)\s*\)\s*(\w+)\s*\([^)]*\)\s*\{([\s\S]*?)\n\}/g;
  let handlerMatch;

  while ((handlerMatch = handlerPattern.exec(content)) !== null) {
    const receiverName = handlerMatch[1];
    const handlerType = handlerMatch[2];
    const handlerName = handlerMatch[3];
    const body = handlerMatch[4];
    const serviceFields = handlerServices.get(handlerType) || new Map();

    for (const [fieldName, serviceType] of serviceFields.entries()) {
      const serviceCallPattern = new RegExp(
        `(?:([A-Za-z_][A-Za-z0-9_]*)\\s*,\\s*err\\s*:=|([A-Za-z_][A-Za-z0-9_]*)\\s*:=)\\s*${receiverName}\\.${fieldName}\\.(\\w+)\\s*\\(`,
        'g',
      );
      let serviceCallMatch;
      while ((serviceCallMatch = serviceCallPattern.exec(body)) !== null) {
        const resultVar = serviceCallMatch[1] || serviceCallMatch[2];
        const methodName = serviceCallMatch[3];
        const successPattern = new RegExp(`common\\.Success\\s*\\(\\s*\\w+\\s*,\\s*${resultVar}\\s*\\)`);
        if (!successPattern.test(body)) {
          continue;
        }

        const method = serviceMethods.get(`${serviceType}.${methodName}`);
        if (!method) {
          warnings.push({
            file: toRepoPath(root, filePath),
            line: lineNumberAt(content, handlerMatch.index + serviceCallMatch.index),
            handler: `${handlerType}.${handlerName}`,
            service: `${serviceType}.${methodName}`,
            reason: 'handler returns service result but service signature could not be inspected',
          });
          continue;
        }

        const riskyReturnTypes = method.returnTypes.filter((typeName) => !isDtoLike(typeName));
        if (riskyReturnTypes.length > 0) {
          warnings.push({
            file: toRepoPath(root, filePath),
            line: lineNumberAt(content, handlerMatch.index + serviceCallMatch.index),
            handler: `${handlerType}.${handlerName}`,
            service: `${serviceType}.${methodName}`,
            returnTypes: riskyReturnTypes,
            reason: 'service return type is not DTO-like; confirm handler does not expose an unstable internal model',
          });
        }
      }
    }
  }

  return { findings, warnings };
}

function scanRepository(root, repoName) {
  const repoRoot = path.join(root, repoName);
  const backendRoot = path.join(repoRoot, 'backend');
  const findings = [];
  const warnings = [];
  const scanWarnings = [];

  if (!fs.existsSync(repoRoot)) {
    scanWarnings.push(`Repository root not found: ${repoName}`);
    return { repo: repoName, findings, warnings, scanWarnings };
  }
  if (!fs.existsSync(backendRoot)) {
    scanWarnings.push(`Backend root not found: ${toRepoPath(root, backendRoot)}`);
    return { repo: repoName, findings, warnings, scanWarnings };
  }

  const files = walkFiles(backendRoot, ['.go']).filter((filePath) => !filePath.endsWith('_test.go'));
  const serviceMethods = parseServiceMethods(files);
  const handlerServices = findHandlerServiceFields(files.filter((filePath) => filePath.endsWith('_handler.go')));

  for (const filePath of files.filter((file) => file.endsWith('_handler.go'))) {
    const result = scanHandlerFile(root, filePath, handlerServices, serviceMethods);
    findings.push(...result.findings);
    warnings.push(...result.warnings);
  }

  return { repo: repoName, findings, warnings, scanWarnings };
}

function printTextReport(results, strict) {
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);
  const mode = strict ? 'strict' : 'report-only';

  console.log(
    `Backend DTO contract check (${mode}): ${findingCount} finding(s), ${warningCount} warning(s), ${scanWarningCount} scan warning(s)`,
  );

  for (const result of results) {
    console.log(`\n${result.repo}`);
    if (result.findings.length === 0 && result.warnings.length === 0 && result.scanWarnings.length === 0) {
      console.log('  no findings');
    }

    for (const finding of result.findings) {
      console.log(`  finding: ${finding.file}:${finding.line}`);
      console.log(`    reason: ${finding.reason}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning.file}:${warning.line}`);
      console.log(`    handler: ${warning.handler}`);
      console.log(`    service: ${warning.service}`);
      if (warning.returnTypes) {
        console.log(`    returnTypes: ${warning.returnTypes.join(', ')}`);
      }
      console.log(`    reason: ${warning.reason}`);
    }

    for (const warning of result.scanWarnings) {
      console.log(`  scan warning: ${warning}`);
    }
  }
}

function main() {
  let options;

  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }

  if (options.help) {
    printHelp();
    return 0;
  }

  const results = REPOSITORIES.map((repoName) => scanRepository(options.root, repoName));
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          root: options.root,
          findingCount,
          warningCount,
          scanWarningCount,
          results,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(results, options.strict);
  }

  return options.strict && findingCount > 0 ? 1 : 0;
}

process.exitCode = main();
