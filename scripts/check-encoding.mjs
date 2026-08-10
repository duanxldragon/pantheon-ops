#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { execFileSync } from 'node:child_process';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_EXTENSIONS = ['.md', '.go', '.ts', '.tsx', '.mjs', '.js', '.jsx', '.json', '.yml', '.yaml', '.sql', '.css', '.html'];

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, extensions: DEFAULT_EXTENSIONS };
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
    } else if (arg === '--ext') {
      const value = argv[index + 1];
      if (!value) throw new Error('--ext requires a comma-separated extension list, e.g. .md,.go,.ts');
      options.extensions = value.split(',').map((ext) => (ext.startsWith('.') ? ext : `.${ext}`));
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-encoding.mjs [--json] [--strict] [--root <path>] [--ext <.md,.go,.ts>]');
  console.log('');
  console.log('Scans git-tracked text files for invalid UTF-8 byte sequences.');
  console.log('Catches mojibake corruption introduced by tools that write files back');
  console.log('with a non-UTF-8 codepage (each corrupted spot loses bytes at multi-byte');
  console.log('character boundaries and often eats the following newline).');
}

function listTrackedFiles(root) {
  const output = execFileSync('git', ['-C', root, 'ls-files', '-z'], {
    encoding: 'buffer',
    maxBuffer: 64 * 1024 * 1024,
  });
  return output
    .toString('utf8')
    .split('\0')
    .filter(Boolean);
}

function findInvalidUtf8Positions(buffer, limit) {
  // Manual UTF-8 validation so we can report byte offsets of each violation.
  const positions = [];
  let index = 0;
  while (index < buffer.length) {
    const byte = buffer[index];
    let expected = 0;
    if (byte <= 0x7f) {
      index += 1;
      continue;
    } else if ((byte & 0xe0) === 0xc0 && byte >= 0xc2) expected = 1;
    else if ((byte & 0xf0) === 0xe0) expected = 2;
    else if ((byte & 0xf8) === 0xf0 && byte <= 0xf4) expected = 3;
    else {
      positions.push(index);
      index += 1;
      if (positions.length >= limit) return positions;
      continue;
    }
    let valid = true;
    for (let k = 1; k <= expected; k += 1) {
      const continuation = buffer[index + k];
      if (continuation === undefined || (continuation & 0xc0) !== 0x80) {
        valid = false;
        break;
      }
    }
    if (valid) index += expected + 1;
    else {
      positions.push(index);
      index += 1;
      if (positions.length >= limit) return positions;
    }
  }
  return positions;
}

function lineNumberOfOffset(buffer, offset) {
  let line = 1;
  for (let index = 0; index < offset && index < buffer.length; index += 1) {
    if (buffer[index] === 0x0a) line += 1;
  }
  return line;
}

function scan(root, extensions) {
  const findings = [];
  const extensionSet = new Set(extensions);
  let scanned = 0;
  for (const relative of listTrackedFiles(root)) {
    if (!extensionSet.has(path.extname(relative).toLowerCase())) continue;
    const absolute = path.join(root, relative);
    if (!fs.existsSync(absolute)) continue;
    const buffer = fs.readFileSync(absolute);
    scanned += 1;
    // UTF-8 BOM is tolerated; skip it so the validator starts on real content.
    const body = buffer[0] === 0xef && buffer[1] === 0xbb && buffer[2] === 0xbf ? buffer.subarray(3) : buffer;
    const positions = findInvalidUtf8Positions(body, 5);
    if (positions.length > 0) {
      findings.push({
        file: relative,
        reason: `invalid UTF-8 byte sequence(s) at line(s) ${positions.map((p) => lineNumberOfOffset(body, p)).join(', ')}${positions.length >= 5 ? ' (first 5 shown)' : ''}`,
      });
    }
  }
  return { findings, scanned };
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
  let result;
  try {
    result = scan(options.root, options.extensions);
  } catch (error) {
    console.error(`encoding check failed to enumerate files: ${error.message}`);
    return 1;
  }
  const { findings, scanned } = result;
  if (options.json) {
    console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', scannedCount: scanned, findingCount: findings.length, findings }, null, 2));
  } else {
    console.log(`Encoding check (${options.strict ? 'strict' : 'report-only'}): ${findings.length} finding(s) across ${scanned} scanned file(s)`);
    for (const finding of findings) console.log(`finding: ${finding.file}\n  reason: ${finding.reason}`);
  }
  return options.strict && findings.length > 0 ? 1 : 0;
}

process.exitCode = main();
