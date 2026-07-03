import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

import { consumeFoundationRelease } from './consume-foundation-release.mjs';

function parseArgs(argv) {
  const options = {
    releaseVersion: null,
    opsRoot: process.cwd(),
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const value = argv[index + 1];

    if (arg === '--release-version') {
      if (!value) throw new Error('--release-version requires a value');
      options.releaseVersion = value;
      index += 1;
    } else if (arg === '--ops-root') {
      if (!value) throw new Error('--ops-root requires a path');
      options.opsRoot = path.resolve(value);
      index += 1;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function printHelp() {
  console.log(`Usage:
  node scripts/foundation-release/upgrade-foundation-release.mjs --release-version <version> [options]

Options:
  --ops-root <path>
  --release-version <version>`);
}

function main() {
  try {
    const options = parseArgs(process.argv.slice(2));
    if (options.help) {
      printHelp();
      return 0;
    }
    if (!options.releaseVersion) {
      throw new Error('--release-version is required');
    }

    const result = consumeFoundationRelease({
      opsRoot: options.opsRoot,
      releaseVersion: options.releaseVersion,
      applySharedBackend: true,
      applySharedFrontend: true,
      updateInheritanceDocs: true,
      check: true,
    });

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
