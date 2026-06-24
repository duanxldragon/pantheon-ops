import { spawnSync } from 'node:child_process';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import { backendOverlayPaths } from './foundation-release/shared-foundation-rules.mjs';

const currentFilePath = fileURLToPath(import.meta.url);
const opsRoot = path.resolve(path.dirname(currentFilePath), '..');

// Overlay files are excluded from byte-level sync checks. But non-overlay
// files may import symbols from overlay files. When base evolves and adds
// new symbols to an overlay file, the overlay stays frozen while the non-overlay
// consumer compiles against the new base version — causing a silent break.
//
// This check compiles the known overlay-sensitive Go packages to verify that
// there are no broken import chains between non-overlay and overlay files.

/**
 * Given an overlay file path (e.g. "internal/scaffold/workspace.go"),
 * return the Go package import path to vet.
 */
function overlayPackage(relativePath) {
  const dir = path.dirname(relativePath);
  return `./${path.join('backend', dir)}/...`;
}

function runGoBuild(pkgPath) {
  const result = spawnSync('go', ['build', pkgPath], {
    cwd: opsRoot,
    encoding: 'utf8',
    timeout: 120_000,
  });
  return { ok: result.status === 0, stderr: result.stderr?.trim() || '' };
}

function overlayConsumerPackages() {
  // Packages known to consume overlay files. Each overlay file may be
  // consumed by multiple non-overlay packages. This list is intentionally
  // explicit — when a new overlay is added to shared-foundation-rules.mjs,
  // its consumers should be recorded here too.
  const consumerMap = {
    'internal/scaffold/workspace.go': [
      './backend/modules/system/dynamicmodule/...',
      './backend/modules/system/generator/...',
      './backend/modules/system/i18n/...',
    ],
    'internal/scaffold/workspace_test.go': [
      './backend/internal/scaffold/...',
      './backend/modules/system/dynamicmodule/...',
    ],
    'modules/system/dynamicmodule/dynamic_module_service_test.go': [
      './backend/modules/system/dynamicmodule/...',
    ],
    'modules/system/generator/generator_service_test.go': [
      './backend/modules/system/generator/...',
    ],
    'modules/system/iam/menu/component_registry.go': [
      './backend/modules/system/iam/menu/...',
      './backend/modules/system/...',
    ],
    'modules/system/iam/menu/generated_component_registry.go': [
      './backend/modules/system/iam/menu/...',
    ],
  };

  const packages = new Set();
  for (const overlay of backendOverlayPaths) {
    const consumers = consumerMap[overlay];
    if (!consumers) continue;
    for (const pkg of consumers) {
      packages.add(pkg);
    }
  }
  return [...packages];
}

function main() {
  const packages = overlayConsumerPackages();
  if (packages.length === 0) {
    console.log('OK no overlay consumer packages to verify');
    return;
  }

  const failures = [];
  for (const pkg of packages) {
    const { ok, stderr } = runGoBuild(pkg);
    if (!ok) {
      failures.push({ pkg, stderr });
    }
  }

  if (failures.length === 0) {
    console.log(`OK overlay consumer packages compile (${packages.length} checked)`);
    return;
  }

  console.error('Overlay compile consistency check failed');
  for (const { pkg, stderr } of failures) {
    console.error(`\nFAIL ${pkg}`);
    console.error(stderr);
  }
  console.error(
    '\nThese failures mean that an overlay file (excluded from sync checks) is\n' +
    'missing symbols that non-overlay files depend on. The overlay likely needs\n' +
    'to be manually merged with the latest base version.',
  );
  process.exit(1);
}

main();
