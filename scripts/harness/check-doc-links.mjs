#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { resolvePantheonHarnessRoot } from './upstream-root.mjs';

const DEFAULT_ROOT = process.cwd();

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
  console.log('Usage: node scripts/harness/check-doc-links.mjs [--json] [--strict] [--root <path>]');
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

function extractLinks(source) {
  const links = [];
  const regex = /\[[^\]]+\]\(([^)]+)\)/g;
  let match;
  while ((match = regex.exec(source)) !== null) {
    const href = match[1].trim();
    if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('#')) continue;
    links.push(href);
  }
  return links;
}

function resolvePantheonHarnessTarget(methodRoot, href) {
  const withoutAnchor = href.replace(/#.*$/u, '').replaceAll('\\', '/');
  const match = withoutAnchor.match(/(?:^|\/)pantheon-harness\/(.+)$/u);
  if (!match) {
    return null;
  }
  return path.join(methodRoot, match[1]);
}

function scan(root) {
  const findings = [];
  const methodRoot = resolvePantheonHarnessRoot(root);
  for (const dir of ['docs', '.agents', 'scripts/harness', 'config']) {
    for (const filePath of walkMarkdownFiles(path.join(root, dir))) {
      const source = fs.readFileSync(filePath, 'utf8');
      for (const href of extractLinks(source)) {
        const target = path.resolve(path.dirname(filePath), href);
        const externalTarget = resolvePantheonHarnessTarget(methodRoot, href);
        const targetExists =
          fs.existsSync(target) ||
          fs.existsSync(target.replace(/#.*$/, '')) ||
          (externalTarget !== null && fs.existsSync(externalTarget));
        if (!targetExists) {
          findings.push({
            file: path.relative(root, filePath).replaceAll(path.sep, '/'),
            reason: `missing internal link target: ${href}`,
          });
        }
      }
    }
  }
  return findings;
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
  const findings = scan(options.root);
  if (options.json) console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', findingCount: findings.length, findings }, null, 2));
  else {
    console.log(`Docs link check (${options.strict ? 'strict' : 'report-only'}): ${findings.length} finding(s)`);
    for (const finding of findings) console.log(`finding: ${finding.file}\n  reason: ${finding.reason}`);
  }
  return options.strict && findings.length > 0 ? 1 : 0;
}

process.exitCode = main();
