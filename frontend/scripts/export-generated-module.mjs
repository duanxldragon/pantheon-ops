import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

import { prepareTranspiledWorkspace } from './transpile-typescript-files.mjs';
const schemaPath = process.argv[2];

if (!schemaPath) {
  throw new Error('schema path required');
}

const files = [
  'src/modules/lowcode/generator/schema.ts',
  'src/modules/lowcode/generator/typeMapping.ts',
  'src/modules/lowcode/generator/backendGenerator.ts',
  'src/modules/lowcode/generator/frontendGenerator.ts',
  'src/modules/lowcode/generator/exporter.ts',
];

const { tempDir } = prepareTranspiledWorkspace(`generator-server-export-${process.pid}`, files);

const { ModuleExporter } = await import(
  pathToFileURL(join(tempDir, 'src', 'modules', 'lowcode', 'generator', 'exporter.js'))
);

// ModuleExporter performs the same admission check used by the wizard/scaffold.
const schema = JSON.parse(readFileSync(schemaPath, 'utf8').replace(/^\uFEFF/, ''));
const exporter = new ModuleExporter(schema);
process.stdout.write(JSON.stringify(exporter.generateAll()));
