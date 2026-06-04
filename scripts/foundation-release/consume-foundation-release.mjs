import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  backendMergedJsonPaths,
  collectFiles,
  detectBackendModuleNameFromTree,
  ensureDir,
  isBackendOverlayPath,
  isFrontendOverlayPath,
  mergeBuiltinLocaleResources,
  readGoModuleName,
  rewriteBackendModuleReferences,
} from './shared-foundation-rules.mjs';

const DEFAULT_OPS_ROOT = process.cwd();

function parseArgs(argv) {
  const options = {
    opsRoot: DEFAULT_OPS_ROOT,
    applySharedBackend: false,
    applySharedFrontend: false,
    updateInheritanceDocs: false,
    check: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const value = argv[index + 1];

    if (arg === '--ops-root') {
      if (!value) throw new Error('--ops-root requires a path');
      options.opsRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--manifest') {
      if (!value) throw new Error('--manifest requires a path');
      options.manifestPath = path.resolve(value);
      index += 1;
    } else if (arg === '--bundle') {
      if (!value) throw new Error('--bundle requires a path');
      options.bundleRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--apply-shared-backend') {
      options.applySharedBackend = true;
    } else if (arg === '--apply-shared-frontend') {
      options.applySharedFrontend = true;
    } else if (arg === '--update-inheritance-docs') {
      options.updateInheritanceDocs = true;
    } else if (arg === '--check') {
      options.check = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function validateOptions(options) {
  if (options.help) {
    return;
  }
  if (!options.manifestPath) {
    throw new Error('manifest is required');
  }
  if (!options.bundleRoot) {
    throw new Error('bundle is required');
  }
}

function readManifest(manifestPath) {
  if (!fs.existsSync(manifestPath)) {
    throw new Error(`manifest not found: ${manifestPath}`);
  }

  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.sourceRepo !== 'pantheon-base') {
    throw new Error('manifest sourceRepo must be pantheon-base');
  }
  if (manifest.consumerMode !== 'foundation-release-consumer') {
    throw new Error('manifest consumerMode must be foundation-release-consumer');
  }
  return manifest;
}

function replaceLine(content, patterns, nextLine) {
  for (const pattern of patterns) {
    if (pattern.test(content)) {
      return content.replace(pattern, nextLine);
    }
  }
  return `${content.trimEnd()}\n${nextLine}\n`;
}

function updateInheritanceDoc(filePath, manifest, language) {
  let content = fs.readFileSync(filePath, 'utf8');

  if (language === 'zh') {
    content = replaceLine(content, [/^- Base branch：.*$/m, /^- Base release line：.*$/m], `- Base release line：当前跟随 \`${manifest.releaseLine}\``);
    content = replaceLine(content, [/^- Base version：.*$/m], `- Base version：当前锁定到 \`${manifest.releaseVersion}\`（\`${manifest.baseCommit}\`）`);
    content = replaceLine(content, [/^- Inheritance mode：.*$/m], '- Inheritance mode：`foundation-release-consumer`');
  } else {
    content = replaceLine(content, [/^- Base branch:.*$/m, /^- Base release line:.*$/m], `- Base release line: \`${manifest.releaseLine}\``);
    content = replaceLine(content, [/^- Base version:.*$/m], `- Base version: \`${manifest.releaseVersion}\` (\`${manifest.baseCommit}\`)`);
    content = replaceLine(content, [/^- Inheritance mode:.*$/m], '- Inheritance mode: `foundation-release-consumer`');
  }

  fs.writeFileSync(filePath, content, 'utf8');
}

function readUtf8(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function writeUtf8(filePath, content) {
  ensureDir(filePath);
  fs.writeFileSync(filePath, content, 'utf8');
}

function resolveBackendModuleNames(bundleRoot, opsRoot, manifest) {
  const sharedBackendRoot = path.join(bundleRoot, 'bundle', 'shared-backend', 'backend');
  const baseModuleName = manifest.baseGoModule
    || manifest.baseModule
    || detectBackendModuleNameFromTree(sharedBackendRoot);
  const opsModuleName = readGoModuleName(opsRoot);
  return { baseModuleName, opsModuleName };
}

function applySharedBackendBundle(bundleRoot, opsRoot, manifest) {
  const sourceRoot = path.join(bundleRoot, 'bundle', 'shared-backend', 'backend');
  if (!fs.existsSync(sourceRoot)) {
    return;
  }

  const targetRoot = path.join(opsRoot, 'backend');
  const { baseModuleName, opsModuleName } = resolveBackendModuleNames(bundleRoot, opsRoot, manifest);

  for (const relativePath of collectFiles(sourceRoot)) {
    if (isBackendOverlayPath(relativePath)) {
      continue;
    }

    const sourcePath = path.join(sourceRoot, relativePath);
    const targetPath = path.join(targetRoot, relativePath);
    const source = readUtf8(sourcePath);
    let nextSource = rewriteBackendModuleReferences(source, baseModuleName, opsModuleName);

    if (backendMergedJsonPaths.has(relativePath) && fs.existsSync(targetPath)) {
      nextSource = mergeBuiltinLocaleResources(nextSource, readUtf8(targetPath));
    }

    writeUtf8(targetPath, nextSource);
  }
}

function applySharedFrontendBundle(bundleRoot, opsRoot) {
  const sourceRoot = path.join(bundleRoot, 'bundle', 'shared-frontend', 'frontend', 'src');
  if (!fs.existsSync(sourceRoot)) {
    return;
  }

  const targetRoot = path.join(opsRoot, 'frontend', 'src');
  for (const relativePath of collectFiles(sourceRoot)) {
    if (isFrontendOverlayPath(relativePath)) {
      continue;
    }

    const sourcePath = path.join(sourceRoot, relativePath);
    const targetPath = path.join(targetRoot, relativePath);
    writeUtf8(targetPath, readUtf8(sourcePath));
  }
}

function runCheckScript(opsRoot, scriptName) {
  const scriptPath = path.join(opsRoot, 'scripts', scriptName);
  if (!fs.existsSync(scriptPath)) {
    throw new Error(`required check script is missing: ${scriptName}`);
  }

  const result = spawnSync(process.execPath, [scriptPath], {
    cwd: opsRoot,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${scriptName} failed`);
  }

  return result.stdout.trim();
}

function runNodeScript(opsRoot, scriptRelativePath) {
  const scriptPath = path.join(opsRoot, scriptRelativePath);
  if (!fs.existsSync(scriptPath)) {
    throw new Error(`required check script is missing: ${scriptRelativePath}`);
  }

  const result = spawnSync(process.execPath, [scriptPath, '--check'], {
    cwd: opsRoot,
    encoding: 'utf8',
  });

  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || `${scriptRelativePath} failed`);
  }

  return result.stdout.trim();
}

export function consumeFoundationRelease(options) {
  validateOptions(options);

  const manifest = readManifest(options.manifestPath);
  const summary = [
    `Target foundation release: ${manifest.releaseVersion}`,
    `Release line: ${manifest.releaseLine}`,
  ];

  if (options.updateInheritanceDocs) {
    updateInheritanceDoc(path.join(options.opsRoot, 'docs', 'PROJECT_INHERITANCE.md'), manifest, 'zh');
    updateInheritanceDoc(path.join(options.opsRoot, 'docs', 'PROJECT_INHERITANCE.en.md'), manifest, 'en');
    summary.push('Updated inheritance docs');
  }

  if (options.applySharedBackend) {
    applySharedBackendBundle(options.bundleRoot, options.opsRoot, manifest);
    summary.push('Applied shared-backend bundle');
  }

  if (options.applySharedFrontend) {
    applySharedFrontendBundle(options.bundleRoot, options.opsRoot);
    summary.push('Applied shared-frontend bundle');
  }

  if (options.check) {
    summary.push('Running check-inheritance-contract.mjs');
    runCheckScript(options.opsRoot, 'check-inheritance-contract.mjs');
    summary.push('Running check-base-backend-sync.mjs');
    runCheckScript(options.opsRoot, 'check-base-backend-sync.mjs');
    summary.push('Running frontend/scripts/sync-base-shared.mjs --check');
    runNodeScript(options.opsRoot, path.join('frontend', 'scripts', 'sync-base-shared.mjs'));
    summary.push('Running frontend/scripts/check-menu-contract.mjs --check');
    runNodeScript(options.opsRoot, path.join('frontend', 'scripts', 'check-menu-contract.mjs'));
  }

  return {
    manifest,
    summary,
  };
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/consume-foundation-release.mjs --manifest <path> --bundle <path> [options]

Options:
  --ops-root <path>
  --apply-shared-backend
  --apply-shared-frontend
  --update-inheritance-docs
  --check`);
}

function main() {
  let options;

  try {
    options = parseArgs(process.argv.slice(2));
    if (options.help) {
      printHelp();
      return 0;
    }

    const result = consumeFoundationRelease(options);
    console.log(result.summary.join('\n'));
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  process.exitCode = main();
}
