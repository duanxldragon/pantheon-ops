import { expect, test, type Route } from '@playwright/test';
import { signInAsAdmin } from '../../helpers/auth';

function initialWorkbench(roleKey: string) {
  return {
    overview: {
      roleCount: 1,
      enabledRoleCount: 1,
      navigationAssignmentCount: 0,
      pagePermissionAssignmentCount: 1,
      actionPermissionAssignmentCount: 1,
      apiActionCount: 0,
      unknownPermissionAssignmentCount: 0,
      pageGapRoleCount: 0,
      apiGapRoleCount: 1,
    },
    roles: [
      {
        id: 501,
        roleName: '权限工作台整改回归',
        roleKey,
        status: 1,
        menuCount: 0,
        pagePermissionCount: 1,
        actionPermissionCount: 1,
        apiPolicyCount: 0,
        requiredApiPolicyCount: 1,
        missingApiPolicyCount: 1,
        unknownPermissionCount: 0,
        hasPageGap: false,
        hasApiGap: true,
        menus: [],
        pagePermissions: [
          {
            key: 'system:generator:use',
            titleKey: 'system.menu.generator',
            path: '/system/generator',
            module: 'system.config',
            kind: 'page',
          },
        ],
        actionPermissions: [
          {
            key: 'system:module:generate',
            titleKey: 'system.permission.module.generate',
            path: '/system/generator',
            module: 'system.config',
            kind: 'action',
          },
        ],
        unknownPermissions: [],
        apiPolicies: [],
        missingApiPolicies: [
          {
            path: '/api/v1/system/dynamic-modules/generate',
            method: 'POST',
          },
        ],
      },
    ],
  };
}

function remediatedWorkbench(roleKey: string) {
  return {
    overview: {
      roleCount: 1,
      enabledRoleCount: 1,
      navigationAssignmentCount: 0,
      pagePermissionAssignmentCount: 1,
      actionPermissionAssignmentCount: 1,
      apiActionCount: 1,
      unknownPermissionAssignmentCount: 0,
      pageGapRoleCount: 0,
      apiGapRoleCount: 0,
    },
    roles: [
      {
        id: 501,
        roleName: '权限工作台整改回归',
        roleKey,
        status: 1,
        menuCount: 0,
        pagePermissionCount: 1,
        actionPermissionCount: 1,
        apiPolicyCount: 1,
        requiredApiPolicyCount: 1,
        missingApiPolicyCount: 0,
        unknownPermissionCount: 0,
        hasPageGap: false,
        hasApiGap: false,
        menus: [],
        pagePermissions: [
          {
            key: 'system:generator:use',
            titleKey: 'system.menu.generator',
            path: '/system/generator',
            module: 'system.config',
            kind: 'page',
          },
        ],
        actionPermissions: [
          {
            key: 'system:module:generate',
            titleKey: 'system.permission.module.generate',
            path: '/system/generator',
            module: 'system.config',
            kind: 'action',
          },
        ],
        unknownPermissions: [],
        apiPolicies: [
          {
            id: 9001,
            path: '/api/v1/system/dynamic-modules/generate',
            method: 'POST',
          },
        ],
        missingApiPolicies: [],
      },
    ],
  };
}

async function fulfillJson(route: Route, status: number, body: Record<string, unknown>) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

test('permission workbench remediation retries through secondary verify and closes api gap', async ({ page }) => {
  const roleKey = 'qa_perm_remediate_mock';
  const roleName = '权限工作台整改回归';
  let remediated = false;
  let firstAttemptCount = 0;
  let retriedWithToken = false;

  await signInAsAdmin(page);

  await page.addInitScript(() => {
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
    sessionStorage.removeItem('pantheon_op_token');
  });

  await page.route(/\/api\/v1\/auth\/me$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        id: 1,
        username: 'admin',
        nickname: '管理员',
        roles: ['admin'],
        perms: ['system:permission:list', 'system:permission:create'],
      },
    });
  });

  await page.route(/\/api\/v1\/auth\/operation-verify$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        operationToken: 'mock-operation-token',
      },
    });
  });

  await page.route(/\/api\/v1\/system\/refresh\/state(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        topics: {},
      },
    });
  });

  await page.route(/\/api\/v1\/system\/menu\/tree(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: [
        {
          id: 50,
          parentId: 0,
          titleKey: 'system.menu.access',
          path: '/system/access',
          component: '',
          pagePerm: '',
          perms: '',
          type: 'M',
          icon: 'idcard',
          routeName: 'system-access',
          module: 'system.iam',
          sort: 20,
          isVisible: 1,
          isCache: 0,
          isExternal: 0,
          activeMenu: '',
          children: [
            {
              id: 7,
              parentId: 50,
              titleKey: 'system.menu.permission',
              path: '/system/permission',
              component: 'system/permission/PermissionList',
              pagePerm: 'system:permission:list',
              perms: '',
              type: 'C',
              icon: 'lock',
              routeName: 'system-permission',
              module: 'system.iam',
              sort: 30,
              isVisible: 1,
              isCache: 0,
              isExternal: 0,
              activeMenu: '',
            },
          ],
        },
      ],
    });
  });

  await page.route(/\/api\/v1\/system\/role\/list(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        items: [
          {
            id: 501,
            roleName,
            roleKey,
          },
        ],
        total: 1,
        page: 1,
        pageSize: 100,
      },
    });
  });

  await page.route(/\/api\/v1\/system\/permission\/list(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        items: [],
        total: 0,
        page: 1,
        pageSize: 10,
      },
    });
  });

  await page.route(/\/api\/v1\/system\/permission\/workbench(?:\?.*)?$/, async (route) => {
    await fulfillJson(route, 200, {
      code: 200,
      data: remediated ? remediatedWorkbench(roleKey) : initialWorkbench(roleKey),
    });
  });

  await page.route(/\/api\/v1\/system\/permission\/workbench\/remediate$/, async (route) => {
    const token = route.request().headers()['x-operation-token'];
    if (!token) {
      firstAttemptCount += 1;
      await fulfillJson(route, 403, {
        code: 403,
        message: 'auth.operation.verification_required',
      });
      return;
    }
    retriedWithToken = true;
    remediated = true;
    await fulfillJson(route, 200, {
      code: 200,
      data: {
        roleKey,
        createdCount: 1,
        skippedCount: 0,
        createdPolicies: [
          {
            id: 9001,
            path: '/api/v1/system/dynamic-modules/generate',
            method: 'POST',
          },
        ],
      },
    });
  });

  await page.goto('/system/permission', { waitUntil: 'networkidle' });

  const roleRow = page.locator('.arco-table-tr').filter({ hasText: roleName }).first();
  await expect(roleRow).toBeVisible();
  await expect(roleRow.getByText('缺接口策略', { exact: true })).toBeVisible();
  await roleRow.getByRole('button', { name: '详情', exact: true }).click();

  const detailDialog = page.getByRole('dialog').filter({ hasText: roleKey }).first();
  await expect(detailDialog).toBeVisible();
  await expect(detailDialog.getByText('/api/v1/system/dynamic-modules/generate', { exact: true })).toBeVisible();

  await detailDialog.getByRole('button', { name: '一键补齐推荐策略', exact: true }).click();

  const verifyDialog = page.getByRole('dialog').filter({ has: page.getByText('敏感操作验证', { exact: true }) }).last();
  await expect(verifyDialog).toBeVisible();
  await verifyDialog.locator('input').first().fill('123456');
  await verifyDialog.getByRole('button', { name: '确定', exact: true }).click();

  await expect.poll(() => firstAttemptCount).toBe(1);
  await expect.poll(() => retriedWithToken).toBeTruthy();
  await expect(page.locator('.arco-message').getByText('已补齐 1 条推荐接口策略', { exact: false }).last()).toBeVisible();
  await expect(detailDialog.getByRole('button', { name: '一键补齐推荐策略', exact: true })).toHaveCount(0);
  await expect(detailDialog.getByText('缺接口策略', { exact: true })).toHaveCount(0);
  await expect(detailDialog.getByText('/api/v1/system/dynamic-modules/generate', { exact: true })).toHaveCount(1);
});
