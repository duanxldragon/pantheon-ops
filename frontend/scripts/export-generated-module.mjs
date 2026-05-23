import { mkdirSync, rmSync, writeFileSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import ts from 'typescript';

const rootDir = dirname(fileURLToPath(import.meta.url));
const frontendDir = join(rootDir, '..');
const tempDir = join(frontendDir, 'node_modules', '.tmp', 'generator-server-export');
const schemaPath = process.argv[2];

if (!schemaPath) {
  throw new Error('schema path required');
}

const files = [
  'src/modules/system/generator/schema.ts',
  'src/modules/system/generator/type-mapping.ts',
  'src/modules/system/generator/backend-generator.ts',
  'src/modules/system/generator/frontend-generator.ts',
  'src/modules/system/generator/exporter.ts',
];

rmSync(tempDir, { recursive: true, force: true });
mkdirSync(tempDir, { recursive: true });
writeFileSync(join(tempDir, 'package.json'), '{"type":"commonjs"}\n');

for (const file of files) {
  const sourcePath = join(frontendDir, file);
  const outputPath = join(tempDir, file.replace(/\.ts$/, '.js'));
  const output = ts.transpileModule(ts.sys.readFile(sourcePath) ?? '', {
    fileName: sourcePath,
    compilerOptions: {
      target: ts.ScriptTarget.ES2023,
      module: ts.ModuleKind.CommonJS,
      esModuleInterop: true,
      importsNotUsedAsValues: ts.ImportsNotUsedAsValues.Remove,
    },
    reportDiagnostics: true,
  });

  if (output.diagnostics?.length) {
    const message = ts.formatDiagnosticsWithColorAndContext(output.diagnostics, {
      getCanonicalFileName: (name) => name,
      getCurrentDirectory: () => frontendDir,
      getNewLine: () => '\n',
    });
    throw new Error(message);
  }

  mkdirSync(dirname(outputPath), { recursive: true });
  writeFileSync(outputPath, output.outputText);
}

const { ModuleExporter } = await import(
  pathToFileURL(join(tempDir, 'src', 'modules', 'system', 'generator', 'exporter.js'))
);

const schema = JSON.parse(readFileSync(schemaPath, 'utf8'));
const exporter = new ModuleExporter(schema);
process.stdout.write(JSON.stringify(exporter.generateAll()));
