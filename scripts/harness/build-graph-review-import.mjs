#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();

function printHelp() {
  console.log(`Usage:
  node scripts/harness/build-graph-review-import.mjs --source <file> [--write <file>] [--root <path>]
  node scripts/harness/build-graph-review-import.mjs --codegraph-path <repo> [--codegraph-bin <path>] [--sync] [--live-callers <symbol>] [--live-callees <symbol>] [--live-impact <symbol>] [--live-context <task>] [--write <file>]

Default behavior:
  Read a saved CodeGraph-style JSON payload and print a normalized graph-review import JSON to stdout.

Examples:
  node scripts/harness/build-graph-review-import.mjs --source trace.json
  node scripts/harness/build-graph-review-import.mjs --source trace.json --write graph-review.json
  node scripts/harness/build-graph-review-import.mjs --codegraph-path D:\\repo\\pantheon-base --live-callers Authenticate --live-callees Authenticate
  node scripts/harness/build-graph-review-import.mjs --codegraph-path D:\\repo\\pantheon-base --codegraph-bin C:\\tools\\codegraph.exe --live-impact AuthService`);
}

function parseArgs(argv) {
  const options = {
    help: false,
    root: DEFAULT_ROOT,
    source: null,
    write: null,
    sync: false,
    codegraphPath: null,
    codegraphBin: process.env.CODEGRAPH_BIN ?? null,
    liveCallers: null,
    liveCallees: null,
    liveImpact: null,
    liveContext: null,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--sync') {
      options.sync = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else if (arg === '--codegraph-path') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--codegraph-path requires a path');
      }
      options.codegraphPath = value;
      index += 1;
    } else if (arg === '--codegraph-bin') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--codegraph-bin requires a path');
      }
      options.codegraphBin = value;
      index += 1;
    } else if (arg === '--live-callers') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--live-callers requires a symbol');
      }
      options.liveCallers = value;
      index += 1;
    } else if (arg === '--live-callees') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--live-callees requires a symbol');
      }
      options.liveCallees = value;
      index += 1;
    } else if (arg === '--live-impact') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--live-impact requires a symbol');
      }
      options.liveImpact = value;
      index += 1;
    } else if (arg === '--live-context') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--live-context requires a task');
      }
      options.liveContext = value;
      index += 1;
    } else if (arg === '--source') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--source requires a path');
      }
      options.source = value;
      index += 1;
    } else if (arg === '--write') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--write requires a path');
      }
      options.write = value;
      index += 1;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  const hasLiveMode =
    options.codegraphPath &&
    (options.liveCallers || options.liveCallees || options.liveImpact || options.liveContext);

  if (!options.help && !options.source && !hasLiveMode) {
    throw new Error('Either --source or live CodeGraph options are required');
  }

  return options;
}

function resolveFile(root, candidate) {
  return path.isAbsolute(candidate) ? candidate : path.join(root, candidate);
}

function readJsonObject(filePath) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`Source file not found: ${filePath}`);
  }
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(`Expected JSON object in ${filePath}`);
  }
  return parsed;
}

function normalizeChecks(value) {
  const items = Array.isArray(value) ? value : typeof value === 'string' ? [value] : [];
  return items
    .map((entry) => String(entry).trim())
    .filter(Boolean)
    .sort((left, right) => left.localeCompare(right));
}

function normalizeFindings(value) {
  const items = Array.isArray(value) ? value : typeof value === 'string' ? [value] : [];
  return items
    .map((entry) => {
      if (typeof entry === 'string') {
        return entry.trim();
      }
      if (entry && typeof entry === 'object') {
        for (const key of ['message', 'reason', 'summary', 'finding']) {
          if (typeof entry[key] === 'string' && entry[key].trim() !== '') {
            return entry[key].trim();
          }
        }
      }
      return '';
    })
    .filter(Boolean);
}

function extractNodeLabel(node) {
  if (typeof node === 'string') {
    return node.trim();
  }
  if (!node || typeof node !== 'object') {
    return '';
  }

  for (const key of ['symbol', 'name', 'label', 'displayName', 'title', 'id']) {
    if (typeof node[key] === 'string' && node[key].trim() !== '') {
      return node[key].trim();
    }
  }

  return '';
}

function normalizeChain(chain) {
  if (!Array.isArray(chain)) {
    return '';
  }
  const labels = chain.map((entry) => extractNodeLabel(entry)).filter(Boolean);
  return labels.length > 0 ? labels.join(' -> ') : '';
}

function normalizeAffectedSubgraph(payload) {
  if (Array.isArray(payload.affectedSubgraph) || typeof payload.affectedSubgraph === 'string') {
    const values = Array.isArray(payload.affectedSubgraph) ? payload.affectedSubgraph : [payload.affectedSubgraph];
    return values
      .map((entry) => String(entry).trim())
      .filter(Boolean)
      .sort((left, right) => left.localeCompare(right));
  }

  if (Array.isArray(payload.trace)) {
    const chain = normalizeChain(payload.trace);
    return chain ? [chain] : [];
  }

  if (Array.isArray(payload.path)) {
    const chain = normalizeChain(payload.path);
    return chain ? [chain] : [];
  }

  if (Array.isArray(payload.paths)) {
    return payload.paths
      .map((entry) => normalizeChain(entry))
      .filter(Boolean)
      .sort((left, right) => left.localeCompare(right));
  }

  return [];
}

function buildImport(payload) {
  return {
    usedCodeGraph: typeof payload.usedCodeGraph === 'boolean' ? payload.usedCodeGraph : true,
    affectedSubgraph: normalizeAffectedSubgraph(payload),
    checks: normalizeChecks(payload.checks),
    findings: normalizeFindings(payload.findings),
    notes: typeof payload.notes === 'string' && payload.notes.trim() !== '' ? payload.notes.trim() : 'derived from saved codegraph output',
  };
}

function resolveCodegraphBin(options) {
  if (!options.codegraphBin) {
    return 'codegraph';
  }
  return resolveFile(options.root, options.codegraphBin);
}

function getWindowsPowerShellBin() {
  const systemRoot = process.env.SystemRoot;
  if (systemRoot) {
    return path.join(systemRoot, 'System32', 'WindowsPowerShell', 'v1.0', 'powershell.exe');
  }
  return 'powershell.exe';
}

function quotePowerShellLiteral(value) {
  return `'${String(value).replace(/'/g, "''")}'`;
}

function isNodeScript(command) {
  return /\.(?:cjs|mjs|js)$/iu.test(command);
}

function unsupportedWindowsWrapperError(command) {
  return new Error(
    `Windows .cmd/.bat wrappers are not supported for codegraph execution: ${command}. Use a direct executable, a .ps1 wrapper, or a .js/.mjs script instead.`,
  );
}

function resolveWindowsShellCommand(command) {
  try {
    const output = execFileSync(
      getWindowsPowerShellBin(),
      [
        '-NoProfile',
        '-Command',
        `$resolved = Get-Command ${quotePowerShellLiteral(command)} -ErrorAction Stop; if ($resolved.Path) { $resolved.Path } else { $resolved.Source }`,
      ],
      { encoding: 'utf8' },
    ).trim();
    return output || null;
  } catch {
    return null;
  }
}

function runProgram(command, args) {
  if (isNodeScript(command)) {
    return execFileSync(process.execPath, [command, ...args], {
      encoding: 'utf8',
    });
  }

  if (process.platform === 'win32' && /\.(cmd|bat)$/iu.test(command)) {
    throw unsupportedWindowsWrapperError(command);
  }

  if (process.platform === 'win32' && /\.ps1$/iu.test(command)) {
    return execFileSync(
      getWindowsPowerShellBin(),
      ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', command, ...args],
      { encoding: 'utf8' },
    );
  }

  try {
    return execFileSync(command, args, {
      encoding: 'utf8',
    });
  } catch (error) {
    if (
      process.platform === 'win32' &&
      (error?.code === 'ENOENT' || error?.code === 'EINVAL') &&
      !path.isAbsolute(command)
    ) {
      const resolved = resolveWindowsShellCommand(command);
      if (resolved && resolved !== command) {
        if (/\.(cmd|bat)$/iu.test(resolved)) {
          throw unsupportedWindowsWrapperError(resolved);
        }
        return runProgram(resolved, args);
      }
    }
    throw error;
  }
}

function runCodegraphJson(options, args) {
  const output = runProgram(resolveCodegraphBin(options), args);
  return JSON.parse(output);
}

function uniqueSorted(values) {
  return sortStrings(Array.from(new Set(values.filter(Boolean))));
}

function buildImportFromLive(options) {
  const repoPath = resolveFile(options.root, options.codegraphPath);
  const codegraphBin = resolveCodegraphBin(options);
  if (options.sync) {
    runProgram(codegraphBin, ['sync', repoPath]);
  }

  const affectedSubgraph = [];
  const checks = new Set();
  const notes = [];

  if (options.liveCallers) {
    const payload = runCodegraphJson(options, ['callers', '-p', repoPath, '-j', options.liveCallers]);
    for (const caller of payload.callers ?? []) {
      if (typeof caller.name === 'string' && caller.name.trim() !== '') {
        affectedSubgraph.push(`${caller.name.trim()} -> ${payload.symbol}`);
      }
    }
    checks.add('call-depth');
    notes.push(`live callers for ${options.liveCallers}`);
  }

  if (options.liveCallees) {
    const payload = runCodegraphJson(options, ['callees', '-p', repoPath, '-j', options.liveCallees]);
    for (const callee of payload.callees ?? []) {
      if (typeof callee.name === 'string' && callee.name.trim() !== '') {
        affectedSubgraph.push(`${payload.symbol} -> ${callee.name.trim()}`);
      }
    }
    checks.add('call-depth');
    notes.push(`live callees for ${options.liveCallees}`);
  }

  if (options.liveImpact) {
    const payload = runCodegraphJson(options, ['impact', '-p', repoPath, '-j', options.liveImpact]);
    for (const entry of payload.affected ?? []) {
      if (typeof entry.name === 'string' && entry.name.trim() !== '') {
        affectedSubgraph.push(`${payload.symbol} -> ${entry.name.trim()}`);
      }
    }
    checks.add('hub');
    notes.push(`live impact for ${options.liveImpact}`);
  }

  let findings = [];
  if (options.liveContext) {
    const payload = runCodegraphJson(options, ['context', '-p', repoPath, '--format', 'json', options.liveContext]);
    for (const entry of payload.entryPoints ?? []) {
      if (typeof entry.name === 'string' && entry.name.trim() !== '') {
        affectedSubgraph.push(`context:${options.liveContext} -> ${entry.name.trim()}`);
      }
    }
    if (typeof payload.summary === 'string' && payload.summary.trim() !== '') {
      findings.push(payload.summary.trim());
    }
    notes.push(`live context for ${options.liveContext}`);
  }

  findings = uniqueSorted(findings);

  return {
    usedCodeGraph: true,
    affectedSubgraph: uniqueSorted(affectedSubgraph),
    checks: uniqueSorted(Array.from(checks)),
    findings,
    notes: notes.length > 0 ? notes.join('; ') : 'derived from live codegraph output',
  };
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

  let result;
  try {
    if (options.source) {
      const payload = readJsonObject(resolveFile(options.root, options.source));
      result = buildImport(payload);
    } else {
      result = buildImportFromLive(options);
    }
    if (options.write) {
      const outputPath = resolveFile(options.root, options.write);
      fs.mkdirSync(path.dirname(outputPath), { recursive: true });
      fs.writeFileSync(outputPath, `${JSON.stringify(result, null, 2)}\n`);
    }
  } catch (error) {
    console.error(error.message);
    return 1;
  }

  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  return 0;
}

process.exitCode = main();
