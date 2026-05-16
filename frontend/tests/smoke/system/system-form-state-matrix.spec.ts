import { expect, test, type Locator, type Page, type Route } from '@playwright/test';
import { installOperationToken, signInAsAdmin } from '../helpers/auth';

type Deferred<T = void> = {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
};

type FormMatrixCase = {
  key: string;
  path: string;
  formTitle: string;
  submitText: string;
  openForm: (page: Page) => Promise<Locator>;
  fillValid: (form: Locator, page: Page) => Promise<void>;
  prepareRequired?: (form: Locator, page: Page) => Promise<void>;
  submitRoutePattern: RegExp;
  successBody: Record<string, unknown>;
  requiredMessages?: string[];
  formatScenario?: {
    prepare: (form: Locator, page: Page) => Promise<void>;
    messages: string[];
  };
  submitLocator?: (form: Locator, page: Page) => Locator;
};

function createDeferred<T = void>(): Deferred<T> {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

function collectRuntimeErrors(page: Page) {
  const runtimeErrors: string[] = [];

  page.on('console', (message) => {
    if (message.type() === 'error') {
      runtimeErrors.push(message.text());
    }
  });

  page.on('pageerror', (error) => {
    runtimeErrors.push(error.message);
  });

  return runtimeErrors;
}

function expectNoRuntimeErrors(runtimeErrors: string[], allowedPatterns: RegExp[] = []) {
  const filteredErrors = runtimeErrors.filter((message) => !(
    /Failed to load resource: the server responded with a status of 502/i.test(message)
    || /Failed to load i18n pack RequestError: Request failed with status code 502/i.test(message)
    || allowedPatterns.some((pattern) => pattern.test(message))
  ));
  expect(filteredErrors).toEqual([]);
}

async function waitForDialog(page: Page, title: string) {
  const dialog = page.getByRole('dialog').filter({ has: page.getByText(title, { exact: true }) });
  await expect(dialog).toBeVisible();
  return dialog;
}

function enabledInputs(dialog: Locator) {
  return dialog.locator('input:not([type="hidden"]):not([disabled]), textarea:not([disabled])');
}

function passwordInput(dialog: Locator, index: number) {
  return dialog.locator('input[type="password"]').nth(index);
}

function resolveSubmitButton(matrixCase: FormMatrixCase, form: Locator, page: Page) {
  return matrixCase.submitLocator
    ? matrixCase.submitLocator(form, page)
    : form.locator('.submit-bar').getByRole('button', { name: matrixCase.submitText, exact: true });
}

async function expectToast(page: Page, text: string) {
  await expect(page.locator('.arco-message').getByText(text, { exact: false }).last()).toBeVisible();
}

async function fulfillJson(route: Route, status: number, body: Record<string, unknown>) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

const matrixCases: FormMatrixCase[] = [
  {
    key: 'user-reset-password',
    path: '/system/user',
    formTitle: '重置用户密码',
    submitText: '重置密码',
    openForm: async (page) => {
      await page.goto('/system/user', { waitUntil: 'networkidle' });
      await page.locator('.system-user-list__table-card').getByRole('button', { name: '重置密码' }).first().click();
      return waitForDialog(page, '重置用户密码');
    },
    fillValid: async (form) => {
      await passwordInput(form, 0).fill('123456');
      await passwordInput(form, 1).fill('123456');
    },
    submitRoutePattern: /\/api\/v1\/system\/user\/\d+\/reset-password$/,
    successBody: {
      code: 200,
      data: {
        passwordReset: true,
        revokedSessionCount: 1,
      },
    },
    requiredMessages: ['请输入密码', '请再次输入新密码'],
    formatScenario: {
      prepare: async (form) => {
        await passwordInput(form, 0).fill('123456');
        await passwordInput(form, 1).fill('654321');
      },
      messages: ['两次输入的新密码不一致'],
    },
  },
  {
    key: 'role-create',
    path: '/system/role',
    formTitle: '新增角色',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/role', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增角色');
    },
    fillValid: async (form) => {
      await enabledInputs(form).nth(0).fill('矩阵测试角色');
      await enabledInputs(form).nth(1).fill('matrix_role');
    },
    submitRoutePattern: /\/api\/v1\/system\/role$/,
    successBody: {
      code: 200,
      data: {
        id: 90001,
      },
    },
    requiredMessages: ['请输入角色名称', '请输入角色标识'],
  },
  {
    key: 'menu-create',
    path: '/system/menu',
    formTitle: '新增菜单',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/menu', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增菜单');
    },
    fillValid: async (form) => {
      await form.getByPlaceholder('例如：system.menu.example').fill('system.menu.matrix');
      await form.getByPlaceholder('例如：/system/example').fill('/system/menu-matrix');
      await form.getByPlaceholder('例如：business/cmdb/CMDBTypeList').fill('system/menu/MenuList');
      await form.getByPlaceholder('例如：system-example').fill('system-menu-matrix');
      await form.getByPlaceholder('例如：system.iam / system.auth / platform / business.order').fill('system.iam');
      await form.getByPlaceholder('例如：system:example:list').nth(0).fill('system:menu:matrix');
    },
    submitRoutePattern: /\/api\/v1\/system\/menu$/,
    successBody: {
      code: 200,
      data: {
        id: 90002,
      },
    },
    requiredMessages: ['请输入标题键', '请输入组件键', '请输入路由名称', '请输入页面权限标识'],
    formatScenario: {
      prepare: async (form) => {
        await form.getByPlaceholder('例如：system.menu.example').fill('system.menu.matrix');
        await form.getByPlaceholder('例如：/system/example').fill('/system/menu-matrix');
        await form.getByPlaceholder('例如：business/cmdb/CMDBTypeList').fill('system/menu/UnknownView');
        await form.getByPlaceholder('例如：system-example').fill('system-menu-matrix');
        await form.getByPlaceholder('例如：system.iam / system.auth / platform / business.order').fill('system.iam');
        await form.getByPlaceholder('例如：system:example:list').nth(0).fill('system:menu:matrix');
      },
      messages: ['平台、系统或业务页面必须使用已注册组件键'],
    },
  },
  {
    key: 'dept-create',
    path: '/system/dept',
    formTitle: '新增部门',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/dept', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增部门');
    },
    fillValid: async (form) => {
      await enabledInputs(form).nth(1).fill('矩阵测试部门');
    },
    submitRoutePattern: /\/api\/v1\/system\/dept$/,
    successBody: {
      code: 200,
      data: {
        id: 90003,
      },
    },
    requiredMessages: ['请输入部门名称'],
    formatScenario: {
      prepare: async (form) => {
        await enabledInputs(form).nth(1).fill('矩阵测试部门');
        await enabledInputs(form).nth(4).fill('invalid-email');
      },
      messages: ['邮箱格式不正确'],
    },
  },
  {
    key: 'dict-type-create',
    path: '/system/dict',
    formTitle: '新增字典类型',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/dict', { waitUntil: 'networkidle' });
      await page.locator('.dict-page__actions').first().getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增字典类型');
    },
    fillValid: async (form) => {
      await enabledInputs(form).nth(0).fill('matrix_dict');
      await enabledInputs(form).nth(1).fill('矩阵测试字典');
    },
    submitRoutePattern: /\/api\/v1\/system\/dict\/type$/,
    successBody: {
      code: 200,
      data: {
        id: 90004,
      },
    },
    requiredMessages: ['请输入字典编码', '请输入字典名称'],
  },
  {
    key: 'post-create',
    path: '/system/post',
    formTitle: '新增岗位',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/post', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增岗位');
    },
    fillValid: async (form) => {
      await enabledInputs(form).nth(1).fill('matrix_post');
      await enabledInputs(form).nth(2).fill('矩阵测试岗位');
    },
    submitRoutePattern: /\/api\/v1\/system\/post$/,
    successBody: {
      code: 200,
      data: {
        id: 90005,
      },
    },
    requiredMessages: ['请输入岗位编码', '请输入岗位名称'],
  },
  {
    key: 'permission-create',
    path: '/system/permission',
    formTitle: '新增策略',
    submitText: '新增',
    openForm: async (page) => {
      await page.goto('/system/permission', { waitUntil: 'networkidle' });
      await page.getByRole('tab', { name: '接口策略', exact: true }).click();
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增策略');
    },
    fillValid: async (form, page) => {
      await form.locator('.arco-select-view').first().click();
      await page.getByRole('option', { name: /超级管理员|superadmin/i }).first().click();
      await form.getByPlaceholder('/api/v1/system/user/list').fill('/api/v1/system/permission/matrix');
    },
    submitRoutePattern: /\/api\/v1\/system\/permission$/,
    successBody: {
      code: 200,
      data: {
        id: 90006,
      },
    },
    requiredMessages: ['请选择角色', '请输入接口路径'],
  },
  {
    key: 'setting-security-update',
    path: '/system/setting',
    formTitle: '安全策略',
    submitText: '保存',
    openForm: async (page) => {
      await page.goto('/system/setting/security', { waitUntil: 'networkidle' });
      const form = page.locator('.setting-page__config-card').first();
      await expect(form.getByText('安全策略', { exact: true }).first()).toBeVisible();
      return form;
    },
    fillValid: async (form) => {
      await form.getByRole('spinbutton').first().fill('8');
    },
    prepareRequired: async (form) => {
      await form.getByRole('spinbutton').first().fill('');
    },
    submitRoutePattern: /\/api\/v1\/system\/setting\/group\/security$/,
    successBody: {
      code: 200,
      data: {
        groupKey: 'security',
      },
    },
    requiredMessages: ['请输入最小密码长度'],
    submitLocator: (form) => form.locator('.submit-bar').getByRole('button', { name: '保存', exact: true }),
  },
  {
    key: 'i18n-create',
    path: '/system/i18n',
    formTitle: '新增翻译',
    submitText: '确定',
    openForm: async (page) => {
      await page.goto('/system/i18n', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      return waitForDialog(page, '新增翻译');
    },
    fillValid: async (form) => {
      await enabledInputs(form).nth(0).fill('system.matrix');
      await enabledInputs(form).nth(2).fill('system.matrix.key');
      await form.locator('textarea').first().fill('矩阵测试文案');
    },
    submitRoutePattern: /\/api\/v1\/system\/i18n$/,
    successBody: {
      code: 200,
      data: {
        id: 90007,
      },
    },
    requiredMessages: ['请输入所属模块', '请输入翻译键', '请输入内容'],
    submitLocator: (_form, page) => page.getByRole('button', { name: '确定', exact: true }),
  },
];

test.describe('system form state matrix', () => {
  test.describe.configure({ timeout: 180000 });

  test('required state matrix uses natural prompts', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    const accessToken = await signInAsAdmin(page);

    for (const matrixCase of matrixCases) {
      if (!matrixCase.requiredMessages?.length) {
        continue;
      }

      const form = await matrixCase.openForm(page);
      if (matrixCase.key === 'setting-security-update') {
        await installOperationToken(page, accessToken);
      }
      if (matrixCase.prepareRequired) {
        await matrixCase.prepareRequired(form, page);
      }
      await resolveSubmitButton(matrixCase, form, page).click({ noWaitAfter: true });

      for (const message of matrixCase.requiredMessages) {
        await expect(form.getByText(message, { exact: true })).toBeVisible();
      }

      const cancelButton = page.getByRole('button', { name: '取消', exact: true }).last();
      if (await cancelButton.isVisible().catch(() => false)) {
        await cancelButton.click();
        if (matrixCase.path !== '/system/setting') {
          await expect(form).toBeHidden();
        }
      }
    }

    expectNoRuntimeErrors(runtimeErrors);
  });

  test('format state matrix keeps validation copy natural', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    const accessToken = await signInAsAdmin(page);

    const formatCases = matrixCases.filter((matrixCase) => matrixCase.formatScenario);
    expect(formatCases.length).toBeGreaterThan(0);

    for (const matrixCase of formatCases) {
      const form = await matrixCase.openForm(page);
      await matrixCase.formatScenario?.prepare(form, page);
      if (matrixCase.key === 'setting-security-update') {
        await installOperationToken(page, accessToken);
      }
      await resolveSubmitButton(matrixCase, form, page).click({ noWaitAfter: true });

      for (const message of matrixCase.formatScenario?.messages || []) {
        await expect(form.getByText(message, { exact: true })).toBeVisible();
      }

      await page.getByRole('button', { name: '取消', exact: true }).last().click();
      await expect(form).toBeHidden();
    }

    expectNoRuntimeErrors(runtimeErrors);
  });

  test('submitting state matrix shows deterministic loading feedback', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    const accessToken = await signInAsAdmin(page);

    for (const matrixCase of matrixCases) {
      const gate = createDeferred<void>();
      let intercepted = false;

      await page.route(matrixCase.submitRoutePattern, async (route) => {
        if (route.request().method() !== 'POST' && route.request().method() !== 'PUT') {
          await route.fallback();
          return;
        }
        intercepted = true;
        await gate.promise;
        await fulfillJson(route, 200, matrixCase.successBody);
      });

      const form = await matrixCase.openForm(page);
      if (matrixCase.key === 'setting-security-update') {
        await installOperationToken(page, accessToken);
      }
      await matrixCase.fillValid(form, page);

      const submit = resolveSubmitButton(matrixCase, form, page);
      await submit.click({ noWaitAfter: true });

      await expect.poll(() => intercepted).toBeTruthy();
      await expect(submit).toHaveClass(/arco-btn-loading/);

      gate.resolve();
      if (matrixCase.path === '/system/setting') {
        await expectToast(page, '更新成功');
      } else {
        await expect(form).toBeHidden();
      }
      await page.unroute(matrixCase.submitRoutePattern);
    }

    expectNoRuntimeErrors(runtimeErrors);
  });

  test('server error matrix keeps modal state and shows friendly copy', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    const accessToken = await signInAsAdmin(page);

    for (const matrixCase of matrixCases) {
      await page.route(matrixCase.submitRoutePattern, async (route) => {
        if (route.request().method() !== 'POST' && route.request().method() !== 'PUT') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 500, {
          code: 500,
          message: 'request.failed',
        });
      });

      const form = await matrixCase.openForm(page);
      if (matrixCase.key === 'setting-security-update') {
        await installOperationToken(page, accessToken);
      }
      await matrixCase.fillValid(form, page);
      await resolveSubmitButton(matrixCase, form, page).click({ noWaitAfter: true });

      await expect(form).toBeVisible();
      await expectToast(page, '请求失败，请稍后重试');

      if (matrixCase.path !== '/system/setting') {
        await page.getByRole('button', { name: '取消', exact: true }).last().click();
        await expect(form).toBeHidden();
      }
      await page.unroute(matrixCase.submitRoutePattern);
    }

    expectNoRuntimeErrors(runtimeErrors, [
      /Failed to load resource: the server responded with a status of 500/i,
      /^request\.failed$/,
    ]);
  });
});
