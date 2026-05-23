import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  installClientSession,
  installOperationToken,
  loginByApi,
  type BrowserLoginResult,
} from '../../helpers/auth';

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(currentDir, '../../../../..');
const platformWorkspaceRoot = path.resolve(repoRoot, '..');
const moduleName = 'orderqa';
const moduleKey = `business.${moduleName}`;
const moduleRoute = `/business/${moduleName}`;
const backendModuleRelativePath = path.join('backend', 'modules', 'business', moduleName);
const frontendModuleRelativePath = path.join('frontend', 'src', 'modules', 'business', moduleName);
const schemaRelativePath = path.join('schema', 'generated', 'business', `${moduleName}.json`);
const backendRegistryRelativePath = path.join('backend', 'modules', 'business', 'generated_registry.go');
const frontendRegistryRelativePath = path.join('frontend', 'src', 'modules', 'generated', 'business.ts');
const componentRegistryRelativePath = path.join(
  'frontend',
  'src',
  'core',
  'router',
  'generatedComponentRegistry.ts',
);

async function listCandidateRepoRoots() {
  const entries = await fs.readdir(platformWorkspaceRoot, { withFileTypes: true });
  const repoRoots = entries
    .filter((entry) => entry.isDirectory() && entry.name.startsWith('pantheon-'))
    .map((entry) => path.join(platformWorkspaceRoot, entry.name));
  return Array.from(new Set([repoRoot, ...repoRoots]));
}

async function hasTarget(target: string) {
  try {
    await fs.stat(target);
    return true;
  } catch {
    return false;
  }
}

async function locateGeneratedRepo(relativePaths: string[]) {
  const candidateRoots = await listCandidateRepoRoots();
  for (const candidateRoot of candidateRoots) {
    const results = await Promise.all(
      relativePaths.map((relativePath) => hasTarget(path.join(candidateRoot, relativePath))),
    );
    if (results.every(Boolean)) {
      return candidateRoot;
    }
  }
  return null;
}

function buildGenerateRequest() {
  return {
    schema: {
      name: moduleName,
      displayName: '订单测试',
      displayNameEn: 'Order QA',
      description: '真实闭环测试模块',
      scope: 'business',
      parentMenu: '',
      templateLevel: 'enterprise',
      metadata: {
        boundedContext: 'Order QA',
        owner: 'Pantheon QA',
        summary: '用于验证模块生成、注册与清理闭环',
        sourceMode: 'manual',
        sourceTable: '',
      },
      model: {
        tableName: 'biz_orderqa',
        modelName: 'Orderqa',
        fields: [
          {
            name: 'name',
            type: 'string',
            label: '名称',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
            placeholder: '请输入名称',
          },
          {
            name: 'status',
            type: 'enum',
            label: '状态',
            required: true,
            searchable: true,
            sortable: true,
            visibleInList: true,
            visibleInForm: true,
            placeholder: '请选择状态',
          },
        ],
      },
      i18n: {
        namespace: 'business.orderqa',
        translations: {
          zh: {
            'business.orderqa.title': '订单测试',
          },
          en: {
            'business.orderqa.title': 'Order QA',
          },
        },
      },
    },
    overwrite: false,
  };
}

async function formItem(page: Page, label: string) {
  return page.locator('.arco-form-item').filter({ has: page.getByText(label, { exact: true }) }).first();
}

async function cleanupModule(request: APIRequestContext, login: BrowserLoginResult, operationToken: string) {
  const response = await request.delete(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}?dropTable=false&purgeSource=true`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
    failOnStatusCode: false,
  });
  if (response.status() === 404 || response.status() === 500 || response.status() === 403) {
    return;
  }
}

async function expectPathMissing(target: string) {
  await expect.poll(async () => {
    try {
      await fs.stat(target);
      return false;
    } catch {
      return true;
    }
  }).toBe(true);
}

test('real module governance flow can generate register and purge a temporary business module', async ({ page }) => {
  const login = await loginByApi(page.request, adminCredentials);
  const operationToken = await getApiOperationToken(page.request, login);
  await cleanupModule(page.request, login, operationToken);
  await installClientSession(page, login);
  await installOperationToken(page, login.accessToken);

  await page.goto('/system/generator', { waitUntil: 'networkidle' });
  await expect(page.getByText(/模块生成(?:器|向导)/).filter({ visible: true }).first()).toBeVisible();
  await expect(page.locator('.generator-wizard-card')).toBeVisible();
  await expect(page.locator('.system-list__work-actions .arco-btn')).toBeVisible();

  await (await formItem(page, '模块名')).locator('input').first().fill(moduleName);
  await (await formItem(page, '显示名')).locator('input').first().fill('订单测试');
  await page.getByRole('button', { name: '下一步', exact: true }).click();

  await page.getByRole('button', { name: '名称', exact: true }).click();
  await page.getByRole('button', { name: '状态', exact: true }).click();
  const fieldTable = page.locator('.arco-table').first();
  await expect(fieldTable.getByText('名称', { exact: true }).first()).toBeVisible();
  await expect(fieldTable.getByText('状态', { exact: true }).first()).toBeVisible();
  await page.getByRole('button', { name: '下一步', exact: true }).click();

  await expect(page.getByText('默认权限', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '生成代码', exact: true }).click();

  await expect(page.getByRole('heading', { name: '预览与接入' })).toBeVisible();
  await expect(page.getByText('共生成 10 个文件', { exact: true })).toBeVisible();

  const generateResponse = await page.request.post(`${apiBaseUrl}/system/dynamic-modules/generate`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
    data: buildGenerateRequest(),
  });
  expect(generateResponse.ok()).toBeTruthy();
  const generatePayload = await generateResponse.json();
  expect(generatePayload.code).toBe(200);
  expect(generatePayload.data?.module?.name).toBe(moduleKey);
  expect([1, 3]).toContain(generatePayload.data?.module?.status);
  expect(generatePayload.data?.summary?.routePath).toBe(moduleRoute);

  await expect.poll(async () => {
    const response = await page.request.get(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}`, {
      headers: apiRequestHeaders(login),
    });
    const payload = await response.json();
    return payload.data?.status === 1 || payload.data?.status === 3;
  }).toBe(true);

  await expect.poll(async () => {
    return locateGeneratedRepo([
      backendModuleRelativePath,
      frontendModuleRelativePath,
      schemaRelativePath,
    ]);
  }).toBeTruthy();
  const generatedRepoRoot = await locateGeneratedRepo([
    backendModuleRelativePath,
    frontendModuleRelativePath,
    schemaRelativePath,
  ]);
  expect(generatedRepoRoot).toBeTruthy();

  const backendModuleDir = path.join(generatedRepoRoot!, backendModuleRelativePath);
  const frontendModuleDir = path.join(generatedRepoRoot!, frontendModuleRelativePath);
  const schemaFile = path.join(generatedRepoRoot!, schemaRelativePath);
  const backendRegistry = path.join(generatedRepoRoot!, backendRegistryRelativePath);
  const frontendRegistry = path.join(generatedRepoRoot!, frontendRegistryRelativePath);
  const componentRegistry = path.join(generatedRepoRoot!, componentRegistryRelativePath);

  await expect.poll(async () => {
    const content = await fs.readFile(backendRegistry, 'utf8');
    return content.includes(`backend/modules/business/${moduleName}`);
  }).toBe(true);
  await expect.poll(async () => {
    const content = await fs.readFile(frontendRegistry, 'utf8');
    return content.includes('OrderqaModule') && content.includes(`../business/${moduleName}`);
  }).toBe(true);
  await expect.poll(async () => {
    const content = await fs.readFile(componentRegistry, 'utf8');
    return content.includes(`business/${moduleName}/OrderqaList`);
  }).toBe(true);

  await page.goto('/system/modules', { waitUntil: 'networkidle' });
  const row = page.getByRole('row', { name: new RegExp(moduleKey) }).first();
  await expect(row).toBeVisible();
  await expect(row.getByText(/已接入|待激活/).first()).toBeVisible();

  const cleanupResponse = await page.request.delete(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}?dropTable=false&purgeSource=true`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': operationToken,
    },
  });
  expect(cleanupResponse.ok()).toBeTruthy();
  const cleanupPayload = await cleanupResponse.json();
  expect(cleanupPayload.code).toBe(200);

  await expect.poll(async () => {
    const response = await page.request.get(`${apiBaseUrl}/system/dynamic-modules/${moduleKey}`, {
      headers: apiRequestHeaders(login),
    });
    const payload = await response.json();
    return payload.data?.status;
  }).toBe(2);

  await expectPathMissing(backendModuleDir);
  await expectPathMissing(frontendModuleDir);
  await expectPathMissing(schemaFile);

  await expect.poll(async () => {
    const content = await fs.readFile(backendRegistry, 'utf8');
    return content.includes(`backend/modules/business/${moduleName}`);
  }).toBe(false);
  await expect.poll(async () => {
    const content = await fs.readFile(frontendRegistry, 'utf8');
    return content.includes('OrderqaModule') || content.includes(`../business/${moduleName}`);
  }).toBe(false);
  await expect.poll(async () => {
    const content = await fs.readFile(componentRegistry, 'utf8');
    return content.includes(`business/${moduleName}/OrderqaList`);
  }).toBe(false);
});
