import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs, { existsSync, mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import { checkDirty, cleanup } from './cleanup-generated-modules.mjs';

function createFixtureRepo() {
  const repoRoot = mkdtempSync(path.join(os.tmpdir(), 'pantheon-generated-modules-'));
  const generatedPaths = {
    backendBusinessDir: path.join(repoRoot, 'backend', 'modules', 'business'),
    frontendBusinessDir: path.join(repoRoot, 'frontend', 'src', 'modules', 'business'),
    schemaBusinessDir: path.join(repoRoot, 'schema', 'generated', 'business'),
    featureLedger: path.join(repoRoot, 'schema', 'generated', 'feature-ledger.json'),
    i18nDir: path.join(repoRoot, 'frontend', 'src', 'i18n', 'resources', 'generated'),
  };
  const registryFiles = {
    backendRegistry: path.join(repoRoot, 'backend', 'modules', 'business', 'generated_registry.go'),
    backendMenuRegistry: path.join(
      repoRoot,
      'backend',
      'modules',
      'system',
      'iam',
      'menu',
      'generated_component_registry.go',
    ),
    frontendBusinessRegistry: path.join(repoRoot, 'frontend', 'src', 'modules', 'generated', 'business.ts'),
    frontendComponentRegistry: path.join(
      repoRoot,
      'frontend',
      'src',
      'core',
      'router',
      'generatedComponentRegistry.ts',
    ),
  };

  for (const dir of [
    generatedPaths.backendBusinessDir,
    generatedPaths.frontendBusinessDir,
    generatedPaths.schemaBusinessDir,
    generatedPaths.i18nDir,
    path.dirname(registryFiles.backendMenuRegistry),
    path.dirname(registryFiles.frontendBusinessRegistry),
    path.dirname(registryFiles.frontendComponentRegistry),
  ]) {
    mkdirSync(dir, { recursive: true });
  }

  return { repoRoot, generatedPaths, registryFiles };
}

test('checkDirty returns no issues for clean generated artifacts', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  try {
    writeFileSync(registryFiles.backendRegistry, 'package business\n', 'utf8');
    writeFileSync(registryFiles.backendMenuRegistry, 'package iam\n', 'utf8');
    writeFileSync(registryFiles.frontendBusinessRegistry, 'export const generatedBusinessModules = [];\n', 'utf8');
    writeFileSync(registryFiles.frontendComponentRegistry, 'export const generatedComponentRegistry = {};\n', 'utf8');
    writeFileSync(path.join(generatedPaths.i18nDir, 'zh-CN.ts'), 'const generatedzhCNFallback = {};\n', 'utf8');

    const dirty = checkDirty(generatedPaths, registryFiles, repoRoot, new Set());
    assert.deepEqual(dirty, []);
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('checkDirty detects generated module leftovers by prefix and content markers', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  try {
    mkdirSync(path.join(generatedPaths.backendBusinessDir, 'mdqa-order'), { recursive: true });
    writeFileSync(path.join(generatedPaths.schemaBusinessDir, 'mdqa-order.json'), '{}\n', 'utf8');
    writeFileSync(
      registryFiles.backendRegistry,
      'import (\n\t"pantheon-base/modules/business/mdqaorder"\n)\n',
      'utf8',
    );
    writeFileSync(
      registryFiles.frontendBusinessRegistry,
      "import './business/mdqaorder';\nexport const generatedBusinessModules = [];\n",
      'utf8',
    );

    const dirty = checkDirty(generatedPaths, registryFiles, repoRoot, new Set());

    assert.ok(dirty.some((item) => item.includes('backend generated_registry.go')));
    assert.ok(dirty.some((item) => item.includes('frontend generated/business.ts')));
    assert.ok(dirty.some((item) => item.includes('mdqa-order')));
    assert.ok(dirty.some((item) => item.includes('mdqa-order.json')));
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('checkDirty detects any leftover business cmdb smoke directory', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  try {
    mkdirSync(path.join(generatedPaths.schemaBusinessDir, 'cmdb', 'host'), { recursive: true });

    const dirty = checkDirty(generatedPaths, registryFiles, repoRoot, new Set());

    assert.ok(dirty.some((item) => item.includes('schema/generated/business/cmdb')));
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup removes generated leftovers and restores clean registry templates', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const backendGeneratedDir = path.join(generatedPaths.backendBusinessDir, 'mdqa-order');
  const frontendGeneratedDir = path.join(generatedPaths.frontendBusinessDir, 'mdqa-order');
  const generatedSchemaPath = path.join(generatedPaths.schemaBusinessDir, 'mdqa-order.json');

  try {
    mkdirSync(backendGeneratedDir, { recursive: true });
    mkdirSync(frontendGeneratedDir, { recursive: true });
    writeFileSync(generatedSchemaPath, '{}\n', 'utf8');
    writeFileSync(
      registryFiles.backendRegistry,
      'import (\n\t"pantheon-base/modules/business/mdqaorder"\n)\n',
      'utf8',
    );
    writeFileSync(
      registryFiles.backendMenuRegistry,
      'var generatedMenuComponentKeys = map[string]struct{}{"business/mdqa-order": {}}\n',
      'utf8',
    );
    writeFileSync(
      registryFiles.frontendBusinessRegistry,
      "import './business/mdqaorder';\nexport const generatedBusinessModules = [];\n",
      'utf8',
    );
    writeFileSync(
      registryFiles.frontendComponentRegistry,
      "export const generatedComponentRegistry = {'business/mdqa-order': {}};\n",
      'utf8',
    );
    writeFileSync(
      path.join(generatedPaths.i18nDir, 'zh-CN.ts'),
      "const generatedzhCNFallback = {\n  'business.mdqa.order': '订单'\n};\nexport default generatedzhCNFallback;\n",
      'utf8',
    );
    mkdirSync(path.join(generatedPaths.schemaBusinessDir, 'cmdb', 'host'), { recursive: true });

    cleanup(generatedPaths, registryFiles, repoRoot, new Set());

    assert.equal(existsSync(backendGeneratedDir), false);
    assert.equal(existsSync(frontendGeneratedDir), false);
    assert.equal(existsSync(generatedSchemaPath), false);
    assert.equal(existsSync(path.join(generatedPaths.schemaBusinessDir, 'cmdb')), false);
    assert.deepEqual(checkDirty(generatedPaths, registryFiles, repoRoot, new Set()), []);
    assert.match(
      fs.readFileSync(registryFiles.backendRegistry, 'utf8'),
      /Intentionally empty: the low-code module generator rewrites this file/,
    );
    assert.match(
      fs.readFileSync(registryFiles.backendMenuRegistry, 'utf8'),
      /generatedMenuComponentKeys = map\[string\]struct\{\}\{\}/,
    );
    assert.match(
      fs.readFileSync(registryFiles.frontendBusinessRegistry, 'utf8'),
      /generatedBusinessModules: ModuleConfig\[\] = \[\];/,
    );
    assert.match(
      fs.readFileSync(registryFiles.frontendComponentRegistry, 'utf8'),
      /generatedComponentRegistry = \{\} satisfies/,
    );
    assert.match(
      fs.readFileSync(path.join(generatedPaths.i18nDir, 'zh-CN.ts'), 'utf8'),
      /generatedzhCNFallback = \{\};/,
    );
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup tolerates schema files disappearing during removal', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const generatedSchemaPath = path.join(generatedPaths.schemaBusinessDir, 'mdqa-order.json');
  const originalRmSync = fs.rmSync;
  let firstSchemaDelete = true;

  try {
    writeFileSync(generatedSchemaPath, '{}\n', 'utf8');

    fs.rmSync = ((targetPath, options) => {
      if (
        firstSchemaDelete
        && targetPath === generatedSchemaPath
        && options
        && typeof options === 'object'
        && 'force' in options
      ) {
        firstSchemaDelete = false;
        originalRmSync(targetPath, options);
        const error = new Error(`ENOENT: no such file or directory, unlink '${targetPath}'`);
        error.code = 'ENOENT';
        throw error;
      }
      return originalRmSync(targetPath, options);
    });

    cleanup(generatedPaths, registryFiles, repoRoot, new Set());

    assert.equal(existsSync(generatedSchemaPath), false);
  } finally {
    fs.rmSync = originalRmSync;
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup preserves tracked consumer overlays while removing nested generated modules', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const trackedBackendFile = path.join(
    generatedPaths.backendBusinessDir,
    'cmdb',
    'host',
    'host_service.go',
  );
  const trackedFrontendFile = path.join(
    generatedPaths.frontendBusinessDir,
    'cmdb',
    'host',
    'CmdbHostList.tsx',
  );
  const generatedBackendDir = path.join(generatedPaths.backendBusinessDir, 'cmdb', 'smoke');
  const generatedFrontendDir = path.join(generatedPaths.frontendBusinessDir, 'orderqa');
  const trackedSchemaFile = path.join(generatedPaths.schemaBusinessDir, 'cmdb', 'host.json');
  const generatedSchemaFile = path.join(generatedPaths.schemaBusinessDir, 'cmdb', 'smoke.json');
  const trackedFiles = new Set([
    path.relative(repoRoot, trackedBackendFile).replaceAll('\\', '/'),
    path.relative(repoRoot, trackedFrontendFile).replaceAll('\\', '/'),
    path.relative(repoRoot, trackedSchemaFile).replaceAll('\\', '/'),
  ]);

  try {
    mkdirSync(path.dirname(trackedBackendFile), { recursive: true });
    mkdirSync(path.dirname(trackedFrontendFile), { recursive: true });
    writeFileSync(trackedBackendFile, 'package host\n', 'utf8');
    writeFileSync(trackedFrontendFile, 'export default function CmdbHostList() {}\n', 'utf8');
    mkdirSync(generatedBackendDir, { recursive: true });
    mkdirSync(generatedFrontendDir, { recursive: true });
    writeFileSync(path.join(generatedBackendDir, 'smoke.go'), 'package smoke\n', 'utf8');
    writeFileSync(path.join(generatedFrontendDir, 'OrderqaList.tsx'), 'export default {}\n', 'utf8');
    mkdirSync(path.dirname(trackedSchemaFile), { recursive: true });
    writeFileSync(trackedSchemaFile, '{}\n', 'utf8');
    writeFileSync(generatedSchemaFile, '{}\n', 'utf8');

    cleanup(generatedPaths, registryFiles, repoRoot, trackedFiles);

    assert.equal(existsSync(trackedBackendFile), true);
    assert.equal(existsSync(trackedFrontendFile), true);
    assert.equal(existsSync(generatedBackendDir), false);
    assert.equal(existsSync(generatedFrontendDir), false);
    assert.equal(existsSync(trackedSchemaFile), true);
    assert.equal(existsSync(generatedSchemaFile), false);
    assert.deepEqual(checkDirty(generatedPaths, registryFiles, repoRoot, trackedFiles), []);
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup fails closed when repository ownership cannot be read', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  try {
    assert.throws(
      () => cleanup(generatedPaths, registryFiles, repoRoot),
      /git -C .* ls-files -z/,
    );
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup fails closed when a tracked registry baseline cannot be read', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const relativeRegistry = path.relative(repoRoot, registryFiles.backendRegistry).replaceAll('\\', '/');
  try {
    assert.throws(
      () => cleanup(generatedPaths, registryFiles, repoRoot, new Set([relativeRegistry])),
      /Unable to read tracked cleanup baseline from Git index/,
    );
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('checkDirty rejects generated markers even when the polluted file is tracked', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const relativeRegistry = path.relative(repoRoot, registryFiles.backendRegistry).replaceAll('\\', '/');
  try {
    writeFileSync(
      registryFiles.backendRegistry,
      'import "pantheon-base/modules/business/mdqaorder"\n',
      'utf8',
    );
    const dirty = checkDirty(generatedPaths, registryFiles, repoRoot, new Set([relativeRegistry]));
    assert.ok(dirty.some((item) => item.includes('backend generated_registry.go')));
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('cleanup restores tracked registry and i18n files from a real Git index', () => {
  const { repoRoot, generatedPaths, registryFiles } = createFixtureRepo();
  const i18nFile = path.join(generatedPaths.i18nDir, 'zh-CN.ts');
  const featureLedgerBaseline = '{"entries":[]}\n';
  const registryBaseline = 'package business\n\nfunc preserveConsumerOverlay() {}\n';
  const i18nBaseline = "const generatedzhCNFallback = { 'business.cmdb.title': 'CMDB' };\nexport default generatedzhCNFallback;\n";

  try {
    writeFileSync(registryFiles.backendRegistry, registryBaseline, 'utf8');
    writeFileSync(i18nFile, i18nBaseline, 'utf8');
    writeFileSync(generatedPaths.featureLedger, featureLedgerBaseline, 'utf8');
    execFileSync('git', ['init'], { cwd: repoRoot, stdio: 'ignore' });
    execFileSync('git', ['add', '.'], { cwd: repoRoot, stdio: 'ignore' });

    writeFileSync(registryFiles.backendRegistry, 'package business\n', 'utf8');
    writeFileSync(i18nFile, 'const generatedzhCNFallback = {};\n', 'utf8');
    writeFileSync(generatedPaths.featureLedger, '{"entries":[{"moduleKey":"business.orderqa"}]}', 'utf8');

    assert.ok(
      checkDirty(generatedPaths, registryFiles, repoRoot).some((item) =>
        item.includes('feature ledger differs from tracked baseline'),
      ),
    );

    cleanup(generatedPaths, registryFiles, repoRoot);

    assert.equal(fs.readFileSync(registryFiles.backendRegistry, 'utf8'), registryBaseline);
    assert.equal(fs.readFileSync(i18nFile, 'utf8'), i18nBaseline);
    assert.equal(fs.readFileSync(generatedPaths.featureLedger, 'utf8'), featureLedgerBaseline);
  } finally {
    rmSync(repoRoot, { recursive: true, force: true });
  }
});
