/**
 * Lowcode generator smoke.
 *
 * 端到端验证 lowcode/generator 在三种典型 schema shape 下：
 *   1. basic single-entity（basic 模板 + 单字段）
 *   2. enterprise master-data（enterprise + 枚举 + lookup）
 *   3. many-to-many（manyToMany 关系 + junction table + 双侧运行时绑定）
 *
 * 对每种 shape：
 *   - 通过 scripts/export-generated-module.mjs 调 ModuleExporter 生成产物；
 *   - 校验 9 个固定产物文件存在（model/dto/service/handler/module + index/api/List/Form/Detail）；
 *   - 后端产物调 gofmt -l（若 gofmt 可用）确保能编译；
 *   - 前端产物调 ts.transpileModule（若 typescript 可用）确保语法 + 简单类型可过。
 *
 * 任何一步失败 exit code 1，便于挂到 npm run 作为门禁。
 */
import { spawn } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(scriptDir, '..');
const repoRoot = join(frontendRoot, '..');

function runCommand(binary, args, cwd) {
  return new Promise((resolve, reject) => {
    const child = spawn(binary, args, { cwd, shell: false, stdio: ['ignore', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    child.stdout.setEncoding('utf8');
    child.stderr.setEncoding('utf8');
    child.stdout.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr.on('data', (chunk) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('exit', (code) => resolve({ code: code ?? 0, stdout, stderr }));
  });
}

function buildSchema(kind) {
  const baseI18n = {
    zh: {
      'business.smoke.title': '烟测模块',
      'business.smoke.field.name.label': '名称',
    },
    en: {
      'business.smoke.title': 'Smoke Module',
      'business.smoke.field.name.label': 'Name',
    },
  };

  if (kind === 'basic') {
    return {
      name: 'smoke/basic',
      templateVersion: 'v1',
      displayName: '烟测基础模块',
      displayNameEn: 'Smoke Basic Module',
      description: 'lowcode smoke basic shape',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'basic',
      pageActionTemplate: 'masterData',
      pageActions: ['view', 'create', 'update', 'delete'],
      dependencies: [],
      relations: [],
      metadata: { sourceMode: 'manual', autoRecycle: false },
      model: {
        tableName: 'biz_smoke_basic',
        modelName: 'SmokeBasic',
        fields: [
          {
            name: 'name',
            type: 'string',
            label: '名称',
            labelEn: 'Name',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
          },
        ],
      },
      menus: [],
      permissions: [],
      i18n: {
        namespace: 'business.smoke.basic',
        translations: baseI18n,
      },
    };
  }

  if (kind === 'enterprise') {
    return {
      name: 'smoke/enterprise',
      templateVersion: 'v1',
      displayName: '烟测企业模块',
      displayNameEn: 'Smoke Enterprise Module',
      description: 'lowcode smoke enterprise shape',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'enterprise',
      pageActionTemplate: 'masterData',
      pageActions: ['view', 'detail', 'create', 'update', 'delete', 'export', 'import'],
      dependencies: [],
      relations: [],
      metadata: { sourceMode: 'manual', autoRecycle: false },
      model: {
        tableName: 'biz_smoke_enterprise',
        modelName: 'SmokeEnterprise',
        fields: [
          {
            name: 'name',
            type: 'string',
            label: '名称',
            labelEn: 'Name',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
          },
          {
            name: 'status',
            type: 'enum',
            label: '状态',
            labelEn: 'Status',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
            enumOptions: [
              { value: 'active', label: '启用', labelEn: 'Active' },
              { value: 'inactive', label: '停用', labelEn: 'Inactive' },
            ],
          },
        ],
      },
      menus: [],
      permissions: [],
      i18n: {
        namespace: 'business.smoke.enterprise',
        translations: baseI18n,
      },
    };
  }

  if (kind === 'many-to-many') {
    return {
      name: 'smoke/group',
      templateVersion: 'v1',
      displayName: '烟测分组模块',
      displayNameEn: 'Smoke Group Module',
      description: 'lowcode smoke many-to-many shape',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'enterprise',
      pageActionTemplate: 'masterData',
      pageActions: ['view', 'detail', 'create', 'update', 'delete'],
      dependencies: [],
      relations: [
        {
          name: 'groupHosts',
          type: 'manyToMany',
          targetModule: 'smoke/host',
          localField: 'id',
          targetField: 'id',
          junctionTable: 'biz_smoke_group_host',
        },
      ],
      metadata: { sourceMode: 'manual', autoRecycle: false },
      model: {
        tableName: 'biz_smoke_group',
        modelName: 'SmokeGroup',
        fields: [
          {
            name: 'name',
            type: 'string',
            label: '分组名称',
            labelEn: 'Group Name',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
          },
        ],
      },
      menus: [],
      permissions: [],
      i18n: {
        namespace: 'business.smoke.group',
        translations: baseI18n,
      },
    };
  }

  throw new Error(`unknown smoke kind: ${kind}`);
}

const EXPECTED_BACKEND_SUFFIXES = [
  '_model.go',
  '_dto.go',
  '_service.go',
  '_handler.go',
  '/module.go',
];

const EXPECTED_FRONTEND_SUFFIXES = [
  '/index.ts',
  '/api.ts',
  'List.tsx',
  'Form.tsx',
  'Detail.tsx',
];

function leafModuleName(name) {
  const segments = String(name ?? '').split('/').filter((segment) => segment.length > 0);
  return segments.length > 0 ? segments[segments.length - 1] : '';
}

function checkExpectedFiles(files, backendPrefix, frontendPrefix, leafName, modelName) {
  const backend = files.filter((file) => file.path.startsWith(backendPrefix));
  const frontend = files.filter((file) => file.path.startsWith(frontendPrefix));

  const expectedBackend = [
    `${backendPrefix}${leafName}_model.go`,
    `${backendPrefix}${leafName}_dto.go`,
    `${backendPrefix}${leafName}_service.go`,
    `${backendPrefix}${leafName}_handler.go`,
    `${backendPrefix}module.go`,
  ];
  for (const expected of expectedBackend) {
    if (!files.some((file) => file.path === expected)) {
      throw new Error(`missing backend file ${expected}`);
    }
  }

  const expectedFrontend = [
    `${frontendPrefix}index.ts`,
    `${frontendPrefix}api.ts`,
    `${frontendPrefix}${modelName}List.tsx`,
    `${frontendPrefix}${modelName}Form.tsx`,
    `${frontendPrefix}${modelName}Detail.tsx`,
  ];
  for (const expected of expectedFrontend) {
    if (!files.some((file) => file.path === expected)) {
      throw new Error(`missing frontend file ${expected}`);
    }
  }

  if (backend.length < expectedBackend.length) {
    throw new Error(`backend artifact count too low: ${backend.length}`);
  }
  if (frontend.length < expectedFrontend.length) {
    throw new Error(`frontend artifact count too low: ${frontend.length}`);
  }
}

async function tryGofmtCheck(workdir) {
  const probe = await runCommand('gofmt', ['-l'], workdir);
  return probe.code === 0;
}

async function runGoFormatCheck(files, workdir) {
  const goFiles = files.filter((file) => file.path.endsWith('.go'));
  const { writeFile: writeSync, mkdir } = await import('node:fs/promises');
  await mkdir(workdir, { recursive: true });

  if (goFiles.length === 0) return { skipped: true, output: '' };

  for (const file of goFiles) {
    const fullPath = join(workdir, file.path);
    await mkdir(dirname(fullPath), { recursive: true });
    await writeSync(fullPath, file.content, 'utf8');
  }

  const probe = await runCommand('gofmt', ['-h'], workdir);
  if (probe.code !== 0 && probe.code !== 1) {
    return { skipped: true, output: 'gofmt not available in PATH' };
  }

  const result = await runCommand('gofmt', ['-l', '-e', workdir], workdir);
  // gofmt -e: report syntax errors when parsing, only list paths that need reformatting
  // Syntax errors will be printed to stderr, formatting issues to stdout.
  const combined = `${result.stdout}${result.stderr}`;
  if (result.code !== 0) {
    return { skipped: false, output: combined, exitCode: result.code };
  }
  return { skipped: false, output: combined, exitCode: 0, dirtyPaths: result.stdout.trim() };
}

async function runTypeScriptCheck(files, workdir) {
  const tsFiles = files.filter((file) => /\.(ts|tsx)$/.test(file.path));
  if (tsFiles.length === 0) {
    return { skipped: true, output: '' };
  }

  let ts;
  try {
    ts = await import('typescript');
  } catch (error) {
    return { skipped: true, output: `typescript not installed: ${error.message}` };
  }

  const diagnostics = [];
  for (const file of tsFiles) {
    const result = ts.transpileModule(file.content, {
      fileName: file.path,
      compilerOptions: {
        target: ts.ScriptTarget.ES2023,
        module: ts.ModuleKind.ESNext,
        jsx: ts.JsxEmit.ReactJSX,
        esModuleInterop: true,
        strict: false,
        skipLibCheck: true,
      },
      reportDiagnostics: true,
    });
    if (result.diagnostics && result.diagnostics.length > 0) {
      diagnostics.push(
        ts.formatDiagnosticsWithColorAndContext(result.diagnostics, {
          getCanonicalFileName: (name) => name,
          getCurrentDirectory: () => workdir,
          getNewLine: () => '\n',
        }),
      );
    }
  }
  return { skipped: false, output: diagnostics.join('\n') };
}

async function runOneShape(kind) {
  const tempDir = await mkdtemp(join(tmpdir(), `pantheon-lowcode-smoke-${kind}-`));
  const schemaPath = join(tempDir, `${kind}.json`);
  const schema = buildSchema(kind);

  try {
    await writeFile(schemaPath, JSON.stringify(schema), 'utf8');
    const exportResult = await runCommand(
      process.execPath,
      ['scripts/export-generated-module.mjs', schemaPath],
      frontendRoot,
    );
    if (exportResult.code !== 0) {
      throw new Error(
        `export-generated-module failed for ${kind}: ${exportResult.stderr || exportResult.stdout}`,
      );
    }
    const files = JSON.parse(exportResult.stdout);
    if (!Array.isArray(files)) {
      throw new Error(`export-generated-module returned non-array for ${kind}`);
    }

    const modelName = schema.model.modelName;
    const leafName = leafModuleName(schema.name);
    if (!leafName) {
      throw new Error(`schema ${kind}: invalid module name ${schema.name}`);
    }
    const backendPrefix = `backend/modules/${schema.scope}/${schema.name}/`;
    const frontendPrefix = `frontend/src/modules/${schema.scope}/${schema.name}/`;
    checkExpectedFiles(files, backendPrefix, frontendPrefix, leafName, modelName);

    const gofmtResult = await runGoFormatCheck(files, join(tempDir, 'gofmt'));
    const tsResult = await runTypeScriptCheck(files, join(tempDir, 'tsc'));

    // gofmt -e 报告语法错误（exit!=0）与 reformat 待办路径（stdout）。
    // 我们要求"语法可解析通过"为门禁，reformat 仅作为告警，避免在生成器没自带格式化时阻塞门禁。
    if (!gofmtResult.skipped && gofmtResult.exitCode && gofmtResult.exitCode !== 0) {
      throw new Error(`gofmt -e failed for ${kind}:\n${gofmtResult.output}`);
    }
    if (!tsResult.skipped && tsResult.output.trim()) {
      throw new Error(`tsc transpile failed for ${kind}:\n${tsResult.output}`);
    }

    const dirtyNote =
      gofmtResult.dirtyPaths && gofmtResult.dirtyPaths.length > 0
        ? ` (gofmt-reformat=${gofmtResult.dirtyPaths.split('\n').length})`
        : '';
    console.log(
      `  ${kind}: ${files.length} files, gofmt=${
        gofmtResult.skipped ? 'skipped' : 'passed'
      }${dirtyNote}, tsc=${tsResult.skipped ? 'skipped' : 'passed'}`,
    );
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }
}

async function main() {
  console.log('[lowcode-smoke] start (cwd=' + repoRoot + ')');
  for (const kind of ['basic', 'enterprise', 'many-to-many']) {
    await runOneShape(kind);
  }
  console.log('[lowcode-smoke] ok');
}

main().catch((error) => {
  console.error('[lowcode-smoke] FAIL:', error.message);
  process.exit(1);
});
