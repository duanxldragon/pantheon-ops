import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

import { prepareTranspiledWorkspace } from './transpile-typescript-files.mjs';

const files = [
  'src/modules/lowcode/generator/schema.ts',
  'src/modules/lowcode/generator/typeMapping.ts',
  'src/modules/lowcode/generator/backendGenerator.ts',
  'src/modules/lowcode/generator/frontendGenerator.ts',
  'src/modules/lowcode/generator/exporter.ts',
  'tests/generator/generator-quality-contract.test.ts',
];

const { tempDir } = prepareTranspiledWorkspace('generator-quality-contract-test', files);
await import(pathToFileURL(join(tempDir, 'tests', 'generator', 'generator-quality-contract.test.js')));
