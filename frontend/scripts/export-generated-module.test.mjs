import assert from 'node:assert/strict';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { spawn } from 'node:child_process';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(scriptDir, '..');

function runNode(args, cwd) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, args, {
      cwd,
      shell: false,
      stdio: ['ignore', 'pipe', 'pipe'],
    });

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
    child.once('exit', (code, signal) => {
      resolve({
        code: code ?? 0,
        signal: signal ?? null,
        stdout,
        stderr,
      });
    });
  });
}

test('export-generated-module script loads the lowcode exporter after module path refactors', async () => {
  const tempDir = await mkdtemp(join(tmpdir(), 'pantheon-export-script-'));
  const schemaPath = join(tempDir, 'schema.json');

  try {
    const schema = {
      name: 'asset',
      templateVersion: 'v1',
      displayName: '资产管理',
      displayNameEn: 'Asset Management',
      description: 'script regression test',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'enterprise',
      pageActionTemplate: 'masterData',
      pageActions: ['view', 'create', 'update', 'delete', 'detail'],
      dependencies: [],
      relations: [],
      metadata: {
        boundedContext: 'qa',
        owner: 'codex',
        summary: 'script regression',
        sourceMode: 'manual',
        sourceTable: '',
        autoRecycle: true,
      },
      model: {
        tableName: 'biz_asset',
        modelName: 'Asset',
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
      menus: [
        {
          key: 'business.asset',
          titleKey: 'business.asset.title',
          path: '/operations/asset',
          component: 'business/asset/AssetList',
          pagePermission: 'business:asset:list',
          type: 'C',
          icon: 'apps',
          routeName: 'business-asset',
          module: 'business.asset',
        },
      ],
      permissions: [
        {
          key: 'business:asset:list',
          name: 'business.asset.permission.view',
          type: 'menu',
          module: 'business.asset',
        },
      ],
      i18n: {
        namespace: 'business.asset',
        translations: {
          zh: {
            'business.asset.title': '资产管理',
            'business.asset.permission.view': '查看资产',
            'business.asset.field.name.label': '名称',
            'business.asset.audit.create': '新增资产',
            'business.asset.audit.update': '编辑资产',
            'business.asset.audit.delete': '删除资产',
          },
          en: {
            'business.asset.title': 'Asset Management',
            'business.asset.permission.view': 'View Asset',
            'business.asset.field.name.label': 'Name',
            'business.asset.audit.create': 'Create Asset',
            'business.asset.audit.update': 'Update Asset',
            'business.asset.audit.delete': 'Delete Asset',
          },
        },
      },
      enableExport: false,
      enableImport: false,
      enableAudit: true,
      enableDataScope: false,
    };

    await writeFile(schemaPath, JSON.stringify(schema), 'utf8');
    const result = await runNode(['scripts/export-generated-module.mjs', schemaPath], frontendRoot);

    assert.equal(result.code, 0, result.stderr);
    const files = JSON.parse(result.stdout);
    assert.ok(Array.isArray(files));
    assert.ok(files.some((file) => file.path === 'backend/modules/business/asset/asset_model.go'));
    assert.ok(files.some((file) => file.path === 'frontend/src/modules/business/asset/AssetList.tsx'));
  } finally {
    await rm(tempDir, { recursive: true, force: true });
  }
});
