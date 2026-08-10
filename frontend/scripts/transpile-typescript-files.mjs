import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';

const rootDir = dirname(fileURLToPath(import.meta.url));
const frontendDir = join(rootDir, '..');

const compilerOptions = {
  target: ts.ScriptTarget.ES2023,
  module: ts.ModuleKind.CommonJS,
  esModuleInterop: true,
};

function transpileFile(sourcePath, outputPath) {
  const output = ts.transpileModule(ts.sys.readFile(sourcePath) ?? '', {
    fileName: sourcePath,
    compilerOptions,
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

export function prepareTranspiledWorkspace(tempDirName, files) {
  const tempDir = join(frontendDir, 'node_modules', '.tmp', tempDirName);
  rmSync(tempDir, { recursive: true, force: true });
  mkdirSync(tempDir, { recursive: true });
  writeFileSync(join(tempDir, 'package.json'), '{"type":"commonjs"}\n');

  for (const file of files) {
    const sourcePath = join(frontendDir, file);
    const outputPath = join(tempDir, file.replace(/\.ts$/, '.js'));
    transpileFile(sourcePath, outputPath);
  }

  return { frontendDir, tempDir };
}
