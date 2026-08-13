// Runs every node:test suite under tests/api/ by transpiling each test file
// plus its transitive relative imports (same mechanism as
// run-request-error-utils-test.mjs, generalized) and executing the compiled
// entry in a child node process. Pure unit tests — no services required.
// Exits non-zero when any suite fails.
import { spawnSync } from 'node:child_process';
import {
  copyFileSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  existsSync,
  writeFileSync,
} from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import ts from 'typescript';

import { prepareTranspiledWorkspace } from './transpile-typescript-files.mjs';

const scriptsDir = dirname(fileURLToPath(import.meta.url));
const frontendDir = resolve(scriptsDir, '..');
const apiTestsDir = join(frontendDir, 'tests', 'api');

const importPattern = /(?:from|import\()\s*['"](\.[^'"]+)['"]/g;

function resolveModuleFile(baseDir, specifier) {
  const raw = resolve(baseDir, specifier);
  const candidates = [
    raw,
    `${raw}.ts`,
    `${raw}.tsx`,
    // ESM 风格的 .js 后缀引用实际指向 .ts 源
    raw.replace(/\.js$/, '.ts'),
    join(raw, 'index.ts'),
    join(raw, 'index.tsx'),
  ];
  return candidates.find(
    (candidate) => existsSync(candidate) && /\.(tsx?|mjs|js)$/.test(candidate),
  );
}

function collectRelativeClosure(entryAbsPath, collected) {
  if (collected.has(entryAbsPath)) {
    return;
  }
  collected.add(entryAbsPath);
  const source = readFileSync(entryAbsPath, 'utf8');
  for (const match of source.matchAll(importPattern)) {
    const resolved = resolveModuleFile(dirname(entryAbsPath), match[1]);
    if (!resolved) {
      throw new Error(`cannot resolve import '${match[1]}' from ${entryAbsPath}`);
    }
    collectRelativeClosure(resolved, collected);
  }
}

function addEsmExtensions(source, sourcePath) {
  return source.replace(
    /((?:from|import\()\s*['"])(\.[^'"]+?)(['"])/g,
    (whole, prefix, specifier, suffix) => {
      if (/\.(js|mjs|json)$/.test(specifier)) {
        return whole;
      }
      const resolved = resolveModuleFile(dirname(sourcePath), specifier);
      if (!resolved) {
        throw new Error(`cannot resolve import '${specifier}' from ${sourcePath}`);
      }
      const emittedPath = resolved.replace(/\.tsx?$/, '.js');
      const mapped = relative(dirname(sourcePath), emittedPath).replaceAll('\\', '/');
      return `${prefix}${mapped.startsWith('.') ? mapped : `./${mapped}`}${suffix}`;
    },
  );
}

function prepareEsmWorkspace(tempDirName, files) {
  const tempDir = join(frontendDir, 'node_modules', '.tmp', tempDirName);
  mkdirSync(tempDir, { recursive: true });
  writeFileSync(join(tempDir, 'package.json'), '{"type":"module"}\n');
  for (const file of files) {
    const sourcePath = join(frontendDir, file);
    // Vite 语义 define：与 vitest 开发环境一致，DEV 视为 true。
    const source = readFileSync(sourcePath, 'utf8').replace(
      /import\.meta\.env\?\.DEV|import\.meta\.env\.DEV/g,
      'true',
    );
    const output = ts.transpileModule(source, {
      fileName: file,
      compilerOptions: { target: ts.ScriptTarget.ES2023, module: ts.ModuleKind.ESNext },
    });
    const outputPath = join(tempDir, file.replace(/\.tsx?$/, '.js'));
    mkdirSync(dirname(outputPath), { recursive: true });
    writeFileSync(outputPath, addEsmExtensions(output.outputText, sourcePath));
  }
  return tempDir;
}

function discoverApiTestFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const entryPath = join(directory, entry.name);
      if (entry.isDirectory()) {
        return discoverApiTestFiles(entryPath).map((file) => join(entry.name, file));
      }
      return entry.isFile() && entry.name.endsWith('.test.ts') ? [entry.name] : [];
    })
    .sort((a, b) => a.localeCompare(b));
}

const testFiles = discoverApiTestFiles(apiTestsDir);

if (testFiles.length === 0) {
  console.error('no tests/api/*.test.ts files found');
  process.exit(1);
}

const failedSuites = [];
for (const testFile of testFiles) {
  const entry = join(apiTestsDir, testFile);
  const closure = new Set();
  collectRelativeClosure(entry, closure);
  const relativeAll = [...closure].map((absPath) =>
    absPath.slice(frontendDir.length + 1).replaceAll('\\', '/'),
  );
  const transpileFiles = relativeAll.filter((file) => /\.tsx?$/.test(file));
  // .mjs/.js 助手原样拷贝：转译出的 CJS 通过 Node>=22 的 require(esm) 加载它们。
  const copyFiles = relativeAll.filter((file) => /\.(mjs|js)$/.test(file));
  // 闭包内含 import.meta（Vite 语义）的套件必须走 ESM 转译，CJS 无法承载。
  const needsEsm = transpileFiles.some((file) =>
    readFileSync(join(frontendDir, file), 'utf8').includes('import.meta'),
  );
  const workspaceName = `api-unit-${testFile.replace(/\.test\.ts$/, '').replace(/[\\/]/g, '-')}`;
  const tempDir = needsEsm
    ? prepareEsmWorkspace(workspaceName, transpileFiles)
    : prepareTranspiledWorkspace(workspaceName, transpileFiles).tempDir;
  for (const file of copyFiles) {
    const target = join(tempDir, file);
    mkdirSync(dirname(target), { recursive: true });
    copyFileSync(join(frontendDir, file), target);
  }
  // 显式 .ts/.tsx 后缀的相对 import 在转译产物中改指 .js
  for (const file of transpileFiles) {
    const compiledPath = join(tempDir, file.replace(/\.tsx?$/, '.js'));
    const content = readFileSync(compiledPath, 'utf8');
    const patched = content.replace(
      /((?:require\(|from )(['"])\.[^'"]+?)\.tsx?(\2\)?)/g,
      '$1.js$3',
    );
    if (patched !== content) {
      writeFileSync(compiledPath, patched);
    }
  }
  const compiled = join(tempDir, 'tests', 'api', testFile.replace(/\.ts$/, '.js'));
  console.log(`\n== tests/api/${testFile} ==`);
  const result = spawnSync(process.execPath, [compiled], { stdio: 'inherit' });
  if (result.status !== 0) {
    failedSuites.push(testFile);
  }
}

if (failedSuites.length > 0) {
  console.error(`\nFAILED suites: ${failedSuites.join(', ')}`);
  process.exit(1);
}
console.log(`\napi unit suites: ${testFiles.length}/${testFiles.length} passed`);
