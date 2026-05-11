import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import ts from 'typescript';

const rootDir = dirname(fileURLToPath(import.meta.url));
const frontendDir = join(rootDir, '..');
const testDir = join(frontendDir, 'node_modules', '.tmp', 'generator-quality-contract-test');

const files = [
  'src/modules/generator/schema.ts',
  'src/modules/generator/type-mapping.ts',
  'src/modules/generator/backend-generator.ts',
  'src/modules/generator/frontend-generator.ts',
  'src/modules/generator/exporter.ts',
  'tests/generator/generator-quality-contract.test.ts',
];

rmSync(testDir, { recursive: true, force: true });
mkdirSync(testDir, { recursive: true });
writeFileSync(join(testDir, 'package.json'), '{"type":"commonjs"}\n');

for (const file of files) {
  const sourcePath = join(frontendDir, file);
  const outputPath = join(testDir, file.replace(/\.ts$/, '.js'));
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

await import(pathToFileURL(join(testDir, 'tests', 'generator', 'generator-quality-contract.test.js')));
