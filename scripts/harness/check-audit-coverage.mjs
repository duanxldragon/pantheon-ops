#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const REPOSITORIES = ['pantheon-base', 'pantheon-ops'];
const WRITE_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH']);
const ROUTE_CALL_PATTERN = /\.(POST|PUT|DELETE|PATCH)\s*\(/g;
const HANDLER_REF_PATTERN = /([a-zA-Z_][a-zA-Z0-9_]*)\.([A-Z][a-zA-Z0-9_]*)/g;
const FUNCTION_PATTERN =
  /func\s+\([^)]*\)\s+([A-Z][a-zA-Z0-9_]*)\s*\(\s*c\s+\*gin\.Context\s*\)\s*\{/g;

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-audit-coverage.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report audit findings and warnings, then exit 0. Use --strict to exit 1 when findings exist.

Examples:
  node scripts/harness/check-audit-coverage.mjs
  node scripts/harness/check-audit-coverage.mjs --json
  node scripts/harness/check-audit-coverage.mjs --strict
  node scripts/harness/check-audit-coverage.mjs --root /tmp/fixture`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
  };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--root') {
      const value = argv[++i];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function walkGoFiles(rootDir) {
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
        stack.push(entryPath);
      } else if (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')) {
        files.push(entryPath);
      }
    }
  }

  return sortStrings(files);
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function lineNumberAt(content, index) {
  return content.slice(0, index).split(/\r?\n/).length;
}

function findMatchingBrace(content, openBraceIndex) {
  let depth = 0;
  for (let index = openBraceIndex; index < content.length; index += 1) {
    const char = content[index];
    if (char === '{') {
      depth += 1;
    } else if (char === '}') {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function findMatchingParen(content, openParenIndex) {
  let depth = 0;
  let quote = '';
  let escaped = false;

  for (let index = openParenIndex; index < content.length; index += 1) {
    const char = content[index];

    if (quote) {
      if (escaped) {
        escaped = false;
      } else if (char === '\\') {
        escaped = true;
      } else if (char === quote) {
        quote = '';
      }
      continue;
    }

    if (char === '"' || char === '`') {
      quote = char;
      continue;
    }

    if (char === '(') {
      depth += 1;
    } else if (char === ')') {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }

  return -1;
}

function buildHandlerAuditMap(goFiles, root) {
  const handlers = new Map();

  for (const filePath of goFiles) {
    const content = fs.readFileSync(filePath, 'utf8');
    let match;
    while ((match = FUNCTION_PATTERN.exec(content)) !== null) {
      const functionName = match[1];
      const openBraceIndex = content.indexOf('{', match.index);
      const closeBraceIndex = findMatchingBrace(content, openBraceIndex);
      const body = closeBraceIndex >= 0 ? content.slice(openBraceIndex, closeBraceIndex + 1) : '';
      handlers.set(functionName, {
        file: toRepoPath(filePath, root),
        line: lineNumberAt(content, match.index),
        hasAuditMetadata: /common\.SetAuditMetadata\s*\(/.test(body),
        hasAuditParam: /common\.SetAuditParam\s*\(/.test(body),
        hasAuditResult: /common\.SetAuditResult\s*\(/.test(body),
        hasAuditStatus: /common\.SetAuditStatus\s*\(/.test(body),
      });
    }
  }

  return handlers;
}

function extractHandlerNames(routeArgs) {
  const names = [];
  let match;
  while ((match = HANDLER_REF_PATTERN.exec(routeArgs)) !== null) {
    names.push(match[2]);
  }
  return names;
}

function scanWriteRoutes(goFiles, handlers, root) {
  const routes = [];
  const warnings = [];

  for (const filePath of goFiles) {
    const content = fs.readFileSync(filePath, 'utf8');
    let match;
    while ((match = ROUTE_CALL_PATTERN.exec(content)) !== null) {
      const method = match[1];
      const openParenIndex = content.indexOf('(', match.index);
      const closeParenIndex = findMatchingParen(content, openParenIndex);
      if (closeParenIndex < 0) {
        continue;
      }
      const routeArgs = content.slice(openParenIndex + 1, closeParenIndex);
      const routePath = routeArgs.match(/^\s*"([^"]*)"/)?.[1] || '';
      if (!WRITE_METHODS.has(method)) {
        continue;
      }

      const handlerNames = extractHandlerNames(routeArgs);
      const route = {
        file: toRepoPath(filePath, root),
        line: lineNumberAt(content, match.index),
        method,
        path: routePath,
        handlers: handlerNames,
      };
      routes.push(route);

      const finalHandler = handlerNames[handlerNames.length - 1];
      if (!finalHandler) {
        warnings.push({
          file: route.file,
          line: route.line,
          route: `${method} ${routePath}`,
          reason: 'write route handler could not be identified for audit metadata inspection',
        });
        continue;
      }

      const handler = handlers.get(finalHandler);
      if (!handler) {
        warnings.push({
          file: route.file,
          line: route.line,
          route: `${method} ${routePath}`,
          handler: finalHandler,
          reason: 'write route handler function was not found for audit metadata inspection',
        });
        continue;
      }

      if (!handler.hasAuditMetadata) {
        warnings.push({
          file: handler.file,
          line: handler.line,
          route: `${method} ${routePath}`,
          handler: finalHandler,
          reason: 'write handler relies on global operation log defaults; consider common.SetAuditMetadata for semantic audit titles',
        });
      }
    }
  }

  return { routes, warnings };
}

function scanRepository(repoName, root) {
  const repoRoot = path.join(root, repoName);
  const backendRoot = path.join(repoRoot, 'backend');
  const mainPath = path.join(backendRoot, 'cmd', 'server', 'main.go');
  const findings = [];
  const warnings = [];
  const scanWarnings = [];

  if (!fs.existsSync(backendRoot)) {
    scanWarnings.push(`Backend root not found: ${toRepoPath(backendRoot, root)}`);
    return { repo: repoName, findings, warnings, scanWarnings, writeRouteCount: 0 };
  }

  if (!fs.existsSync(mainPath)) {
    findings.push({
      file: toRepoPath(mainPath, root),
      line: 1,
      reason: 'server entrypoint not found; cannot verify OperationLogMiddleware registration',
    });
  } else {
    const mainContent = fs.readFileSync(mainPath, 'utf8');
    if (!/OperationLogMiddleware\s*\(/.test(mainContent)) {
      findings.push({
        file: toRepoPath(mainPath, root),
        line: 1,
        reason: 'OperationLogMiddleware is not registered in server entrypoint',
      });
    }
  }

  const goFiles = walkGoFiles(path.join(backendRoot, 'modules'));
  const handlers = buildHandlerAuditMap(goFiles, root);
  const routeScan = scanWriteRoutes(goFiles, handlers, root);

  warnings.push(...routeScan.warnings);

  return {
    repo: repoName,
    findings,
    warnings,
    scanWarnings,
    writeRouteCount: routeScan.routes.length,
  };
}

function printTextReport(results, strict) {
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);
  const writeRouteCount = results.reduce((count, result) => count + result.writeRouteCount, 0);
  const mode = strict ? 'strict' : 'report-only';

  console.log(
    `Audit coverage check (${mode}): ${findingCount} finding(s), ${warningCount} warning(s), ${scanWarningCount} scan warning(s), ${writeRouteCount} write route(s)`,
  );

  for (const result of results) {
    console.log(`\n${result.repo}`);
    console.log(`  write routes scanned: ${result.writeRouteCount}`);

    if (result.findings.length === 0 && result.warnings.length === 0 && result.scanWarnings.length === 0) {
      console.log('  no findings');
    }

    for (const finding of result.findings) {
      console.log(`  finding: ${finding.file}:${finding.line}`);
      console.log(`    reason: ${finding.reason}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning.file}:${warning.line}`);
      if (warning.route) {
        console.log(`    route: ${warning.route}`);
      }
      if (warning.handler) {
        console.log(`    handler: ${warning.handler}`);
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

  const results = REPOSITORIES.map((repo) => scanRepository(repo, options.root));
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);
  const writeRouteCount = results.reduce((count, result) => count + result.writeRouteCount, 0);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          findingCount,
          warningCount,
          scanWarningCount,
          writeRouteCount,
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
