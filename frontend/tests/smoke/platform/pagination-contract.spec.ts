import { test } from '../../fixtures/coverage';
import { expect, type Page, type Route } from '@playwright/test';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { signInAsAdmin } from '../helpers/auth';

const SOURCE_FILE_EXTENSIONS = new Set(['.ts', '.tsx']);

function buildUserRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 7000 + index,
    username: `pagination-user-${index + 1}`,
    nickname: `用户${index + 1}`,
    email: `pagination-user-${index + 1}@example.com`,
    phone: `1380000${String(index + 1).padStart(4, '0')}`,
    deptId: 1,
    deptName: '平台研发',
    postId: 1,
    postName: '工程师',
    status: 1,
    createdAt: '2026-05-21 10:00:00',
    roleIds: [1],
    roleKeys: ['admin'],
    roleNames: ['管理员'],
  }));
}

function buildRoleRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 8000 + index,
    roleName: `分页角色${index + 1}`,
    roleKey: `pagination_role_${index + 1}`,
    sort: index + 1,
    status: 1,
    createdAt: '2026-05-21 10:00:00',
    menuIds: [],
    permissionKeys: [],
  }));
}

function buildPermissionWorkbenchRoles(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 9000 + index,
    roleName: `权限角色${index + 1}`,
    roleKey: `permission_role_${index + 1}`,
    status: 1,
    menuCount: 2,
    pagePermissionCount: 3,
    actionPermissionCount: 2,
    apiPolicyCount: 4,
    requiredApiPolicyCount: 5,
    missingApiPolicyCount: 1,
    unknownPermissionCount: 0,
    hasPageGap: false,
    hasApiGap: true,
    governanceStatus: 'pending',
    menus: [],
    pagePermissions: [],
    actionPermissions: [],
    unknownPermissions: [],
    apiPolicies: [],
    missingApiPolicies: [{ id: 1, path: `/api/v1/mock/${index + 1}`, method: 'GET' }],
  }));
}

function buildDataScopeRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 9500 + index,
    roleName: `数据权限角色${index + 1}`,
    roleKey: `data_scope_role_${index + 1}`,
    status: 1,
    mode: 'all',
    deptIds: [],
    policyExists: false,
  }));
}

function buildPermissionPolicyRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 9700 + index,
    ptype: 'p',
    roleKey: `permission_role_${index + 1}`,
    path: `/api/v1/mock/resource/${index + 1}`,
    method: index % 2 === 0 ? 'GET' : 'POST',
  }));
}

function buildMenuRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 9900 + index,
    parentId: 0,
    titleKey: `system.menu.pagination.${index + 1}`,
    path: `/system/pagination-menu-${index + 1}`,
    component: 'system/menu/MenuList',
    pagePerm: 'system:menu:list',
    perms: '',
    type: 'C',
    icon: 'menu',
    routeName: `system-menu-pagination-${index + 1}`,
    module: 'system',
    sort: index + 1,
    isVisible: 1,
    isCache: 0,
    isExternal: 0,
    activeMenu: '',
    children: [],
  }));
}

function buildLoginLogRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 12000 + index,
    username: `pagination-login-${index + 1}`,
    ipaddr: '127.0.0.1',
    loginLocation: '本地',
    browser: 'Chrome',
    os: 'Windows',
    status: 1,
    msg: '登录成功',
    loginTime: `2026-05-${String((index % 28) + 1).padStart(2, '0')} 10:00:00`,
  }));
}

function buildOperationLogRows(total: number) {
  return Array.from({ length: total }, (_, index) => ({
    id: 13000 + index,
    title: `分页操作-${index + 1}`,
    businessType: 1,
    method: index % 2 === 0 ? 'GET' : 'POST',
    operName: `operator-${index + 1}`,
    operUrl: `/api/v1/mock/pagination/${index + 1}`,
    operIp: '127.0.0.1',
    sourceDomain: 'system',
    sourcePage: 'operationLog',
    operParam: '{}',
    jsonResult: '{}',
    status: index % 3 === 0 ? 1 : 0,
    failureCategory: index % 3 === 0 ? 'timeout' : '',
    errorMsg: index % 3 === 0 ? 'timeout' : '',
    operTime: `2026-05-${String((index % 28) + 1).padStart(2, '0')} 11:00:00`,
    costTime: 20 + index,
  }));
}

function collectSourceFiles(rootDir: string) {
  const files: string[] = [];
  for (const entry of readdirSync(rootDir)) {
    const absolutePath = join(rootDir, entry);
    const stats = statSync(absolutePath);
    if (stats.isDirectory()) {
      files.push(...collectSourceFiles(absolutePath));
      continue;
    }
    if (SOURCE_FILE_EXTENSIONS.has(extname(absolutePath))) {
      files.push(absolutePath);
    }
  }
  return files;
}

function readModuleSources(repoRoot: string) {
  const moduleRoot = join(repoRoot, 'src/modules');
  return collectSourceFiles(moduleRoot).map((absolutePath) => ({
    absolutePath,
    relativePath: relative(repoRoot, absolutePath).replace(/\\/g, '/'),
    source: readFileSync(absolutePath, 'utf8'),
  }));
}

function extractOpeningTags(source: string, tagName: 'AppTable' | 'Table') {
  return source
    .split(`<${tagName}`)
    .slice(1)
    .map((fragment) => {
      const tagEndIndex = fragment.indexOf('>');
      return tagEndIndex >= 0 ? `<${tagName}${fragment.slice(0, tagEndIndex + 1)}` : '';
    })
    .filter(Boolean);
}

async function expectStandardPager(page: Page) {
  const paginationShell = page.locator('.app-table__pagination-shell:visible').first();
  await expect(paginationShell).toBeVisible();
  await expect(paginationShell.getByRole('button', { name: /^(首页|First page)$/ })).toBeVisible();
  await expect(paginationShell.getByRole('button', { name: /^(末页|Last page)$/ })).toBeVisible();
  return paginationShell;
}

async function expectTotalSummaryVisible(page: Page, total: number) {
  const paginationShell = await expectStandardPager(page);
  await expect(
    paginationShell.locator('.app-table__pagination-total-text').filter({
      hasText: String(total),
    }),
  ).toBeVisible();
}

async function expectPaginationControlsAligned(page: Page) {
  const paginationShell = await expectStandardPager(page);
  const metrics = await paginationShell.evaluate((shell) => {
    const selectors = [
      '.app-table__pagination-item-first',
      '.arco-pagination-total',
      '.arco-pagination-item-active',
      '.arco-pagination-option .arco-select-view, .arco-pagination-options .arco-select-view',
      '.arco-pagination-jumper input',
      '.app-table__pagination-item-last',
    ];

    return selectors
      .map((selector) => shell.querySelector<HTMLElement>(selector))
      .filter((element): element is HTMLElement => Boolean(element))
      .map((element) => {
        const rect = element.getBoundingClientRect();
        return {
          height: rect.height,
          center: rect.top + rect.height / 2,
        };
      });
  });

  expect(metrics.length).toBeGreaterThanOrEqual(4);
  const centers = metrics.map((item) => item.center);
  const heights = metrics.map((item) => item.height);
  expect(Math.max(...centers) - Math.min(...centers)).toBeLessThan(8);
  expect(Math.min(...heights)).toBeGreaterThanOrEqual(28);
}

async function expectBoundaryNavigationWorks(page: Page) {
  const paginationShell = await expectStandardPager(page);
  const firstPageButton = paginationShell.getByRole('button', { name: /^(首页|First page)$/ });
  const lastPageButton = paginationShell.getByRole('button', { name: /^(末页|Last page)$/ });

  await expect(paginationShell.locator('.arco-pagination-item-active')).toContainText('1');
  await lastPageButton.click();
  await expect
    .poll(async () => paginationShell.locator('.arco-pagination-item-active').innerText())
    .toBe('2');
  await firstPageButton.click();
  await expect
    .poll(async () => paginationShell.locator('.arco-pagination-item-active').innerText())
    .toBe('1');
}

async function expectPaginationResponsiveMobile(page: Page) {
  const paginationShell = await expectStandardPager(page);
  const metrics = await paginationShell.evaluate((shell) => {
    const shellRect = shell.getBoundingClientRect();
    const first = shell.querySelector<HTMLElement>('.app-table__pagination-item-first');
    const last = shell.querySelector<HTMLElement>('.app-table__pagination-item-last');
    const native = shell.querySelector<HTMLElement>('.app-table__pagination-native');
    const firstRect = first?.getBoundingClientRect();
    const lastRect = last?.getBoundingClientRect();
    const nativeRect = native?.getBoundingClientRect();

    return {
      shell: {
        width: shellRect.width,
        height: shellRect.height,
        center: shellRect.top + shellRect.height / 2,
      },
      first: firstRect
        ? {
            top: firstRect.top,
            left: firstRect.left,
            center: firstRect.top + firstRect.height / 2,
          }
        : null,
      last: lastRect
        ? {
            top: lastRect.top,
            left: lastRect.left,
            center: lastRect.top + lastRect.height / 2,
          }
        : null,
      native: nativeRect
        ? {
            top: nativeRect.top,
            height: nativeRect.height,
            center: nativeRect.top + nativeRect.height / 2,
            width: nativeRect.width,
            left: nativeRect.left,
            right: nativeRect.right,
          }
        : null,
      scrollWidth: shell.scrollWidth,
      clientWidth: shell.clientWidth,
    };
  });

  expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth + 1);
  expect(metrics.shell.height).toBeGreaterThanOrEqual(60);
  expect(metrics.first).not.toBeNull();
  expect(metrics.last).not.toBeNull();
  expect(metrics.native).not.toBeNull();
  expect(Math.abs(metrics.first!.top - metrics.last!.top)).toBeLessThanOrEqual(2);
  expect(metrics.native!.width).toBeLessThanOrEqual(metrics.shell.width);
  expect(metrics.first!.left).toBeGreaterThanOrEqual(metrics.native!.left);
  expect(metrics.last!.left).toBeLessThanOrEqual(metrics.native!.right);
  expect(Math.abs(metrics.first!.center - metrics.native!.center)).toBeLessThanOrEqual(20);
  expect(Math.abs(metrics.last!.center - metrics.native!.center)).toBeLessThanOrEqual(20);
}

async function fulfillPagedRoute(route: Route, rows: unknown[], pageSize = 10) {
  const requestUrl = new URL(route.request().url());
  const currentPage = Number(requestUrl.searchParams.get('page') || '1');
  const currentPageSize = Number(requestUrl.searchParams.get('pageSize') || String(pageSize));
  const startIndex = (currentPage - 1) * currentPageSize;
  const items = rows.slice(startIndex, startIndex + currentPageSize);
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      code: 200,
      data: {
        items,
        total: rows.length,
        page: currentPage,
        pageSize: currentPageSize,
      },
    }),
  });
}

test('paginated module sources use the shared AppTable pagination contract', async () => {
  const repoRoot = process.cwd();
  const offenders = readModuleSources(repoRoot).flatMap(({ relativePath, source }) => {
    const issues: string[] = [];
    const appTableTags = extractOpeningTags(source, 'AppTable');
    const rawTableTags = extractOpeningTags(source, 'Table');
    const usesPaginatedAppTable = appTableTags.some((tag) => /pagination=\{(?!false\b)/.test(tag));
    const usesPaginatedRawTable = rawTableTags.some((tag) => /pagination=\{(?!false\b)/.test(tag));

    if (usesPaginatedAppTable && !source.includes('buildStandardPagination(')) {
      issues.push(`${relativePath}: AppTable pagination must be created by buildStandardPagination()`);
    }
    if (usesPaginatedRawTable) {
      issues.push(`${relativePath}: module paginated tables must use AppTable instead of raw Table`);
    }

    return issues;
  });

  expect(offenders).toEqual([]);
});

test('generator list templates emit the shared standard pagination helper', async () => {
  const repoRoot = process.cwd();
  const relativePath = 'src/modules/lowcode/generator/frontendGenerator.ts';
  const source = readFileSync(join(repoRoot, relativePath), 'utf8');
  const offenders = source.includes('buildStandardPagination(') ? [] : [relativePath];

  expect(offenders).toEqual([]);
});

test('standard pager shell stays visible on representative paginated surfaces', async ({ page }) => {
  const total = 11;
  await signInAsAdmin(page);

  await page.route(/\/api\/v1\/system\/user\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildUserRows(total));
  });
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildLoginLogRows(total));
  });
  await page.route(/\/api\/v1\/system\/operation-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildOperationLogRows(total));
  });

  await page.goto('/system/user', { waitUntil: 'networkidle' });
  await expectBoundaryNavigationWorks(page);
  await expectTotalSummaryVisible(page, total);
  await expectPaginationControlsAligned(page);

  const roleRows = buildRoleRows(total);
  await page.route(/\/api\/v1\/system\/role\/list(?:\?.*)?$/, async (route) => {
    const requestUrl = new URL(route.request().url());
    const requestedPageSize = Number(requestUrl.searchParams.get('pageSize') || '10');
    if (requestedPageSize >= 100) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          data: {
            items: roleRows,
            total: roleRows.length,
            page: 1,
            pageSize: requestedPageSize,
          },
        }),
      });
      return;
    }
    await fulfillPagedRoute(route, roleRows);
  });
  await page.route(/\/api\/v1\/system\/permission\/workbench(?:\?.*)?$/, async (route) => {
    const roles = buildPermissionWorkbenchRoles(total);
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          overview: {
            roleCount: total,
            enabledRoleCount: total,
            navigationAssignmentCount: total,
            pagePermissionAssignmentCount: total,
            actionPermissionAssignmentCount: total,
            apiActionCount: total,
            unknownPermissionAssignmentCount: 0,
            pageGapRoleCount: 0,
            apiGapRoleCount: total,
            pendingRemediationRoleCount: total,
            remediatedRoleCount: 0,
            recentRemediationCount: 0,
          },
          roles,
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/system\/permission\/data-scope(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: buildDataScopeRows(total),
          total,
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/system\/permission\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildPermissionPolicyRows(total));
  });
  await page.route(/\/api\/v1\/system\/menu\/tree(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: buildMenuRows(total),
      }),
    });
  });

  await page.goto('/system/permission', { waitUntil: 'networkidle' });
  const permissionTabs = page.locator('.permission-workbench__tabs .arco-tabs-header-title');
  await expect(permissionTabs).toHaveCount(3);
  await expect(page.getByText('权限角色1', { exact: true })).toBeVisible();
  await expectBoundaryNavigationWorks(page);

  await permissionTabs.nth(1).click();
  await expect(page.getByText('数据权限角色1', { exact: true })).toBeVisible();
  await expectBoundaryNavigationWorks(page);

  await permissionTabs.nth(2).click();
  await expect(page.getByText('/api/v1/mock/resource/1', { exact: true })).toBeVisible();
  await expectBoundaryNavigationWorks(page);

  await page.goto('/system/menu', { waitUntil: 'networkidle' });
  await expect(
    page.locator('.system-list__table').getByText('system.menu.pagination.1', { exact: true }),
  ).toBeVisible();
  await expectBoundaryNavigationWorks(page);
  await expectTotalSummaryVisible(page, total);
  await expectPaginationControlsAligned(page);

  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await expect(page.getByText('pagination-login-1', { exact: true })).toBeVisible();
  await expectBoundaryNavigationWorks(page);
  await expectTotalSummaryVisible(page, total);
  await expectPaginationControlsAligned(page);

  await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
  await expect(page.getByText('分页操作-1', { exact: true })).toBeVisible();
  await expectBoundaryNavigationWorks(page);
  await expectTotalSummaryVisible(page, total);
  await expectPaginationControlsAligned(page);
});

test('standard pager shell wraps cleanly on narrow mobile surfaces', async ({ page }) => {
  const total = 11;
  await signInAsAdmin(page);
  await page.setViewportSize({ width: 430, height: 900 });

  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildLoginLogRows(total));
  });
  await page.route(/\/api\/v1\/system\/operation-log\/list(?:\?.*)?$/, async (route) => {
    await fulfillPagedRoute(route, buildOperationLogRows(total));
  });

  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await expect(page.getByText('pagination-login-1', { exact: true })).toBeVisible();
  await expectPaginationResponsiveMobile(page);

  await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
  await expect(page.getByText('分页操作-1', { exact: true })).toBeVisible();
  await expectPaginationResponsiveMobile(page);
});
