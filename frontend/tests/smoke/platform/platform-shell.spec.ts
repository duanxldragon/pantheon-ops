import { expect, test, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  signInAsAdmin,
  verifiedHeaders,
} from '../helpers/auth';


function formItem(page: Page, label: string) {
  return page.locator('.arco-form-item').filter({ has: page.getByText(label, { exact: true }) }).first();
}

async function waitForDialog(page: Page, title: string) {
  const dialog = page.getByRole('dialog').filter({ has: page.getByText(title, { exact: true }) });
  await expect(dialog).toBeVisible();
  return dialog;
}

test.describe('enter submit smoke', () => {
  test('login page submits with Enter key', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/请输入密码|password/i).fill(adminCredentials.password);
    await page.locator('input[type="password"]').press('Enter');
    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.locator('.app-shell')).toBeVisible();
  });

  test('list filters submit with Enter key on auth pages', async ({ page }) => {
    await signInAsAdmin(page);

    await page.goto('/system/login-log', { waitUntil: 'networkidle' });
    const loginLogUsername = formItem(page, '用户名').locator('input').first();
    const loginLogRequest = page.waitForRequest((request) => (
      request.method() === 'GET'
      && request.url().includes('/api/v1/system/login-log/list')
      && request.url().includes('username=admin')
    ));
    await loginLogUsername.fill('admin');
    await loginLogUsername.press('Enter');
    await loginLogRequest;

    await page.goto('/system/session', { waitUntil: 'networkidle' });
    const sessionUsername = formItem(page, '用户名').locator('input').first();
    const sessionRequest = page.waitForRequest((request) => (
      request.method() === 'GET'
      && request.url().includes('/api/v1/system/session/list')
      && request.url().includes('username=admin')
    ));
    await sessionUsername.fill('admin');
    await sessionUsername.press('Enter');
    await sessionRequest;
  });

  test('dict, i18n, and permission dialogs submit with Enter key', async ({ page }) => {
    const accessToken = await signInAsAdmin(page);
    const dictCode = `enter_dict_${Date.now()}`;
    const dictName = `回车测试字典_${Date.now()}`;
    const i18nKey = `i18n.enter.${Date.now()}`;
    const permissionPath = `/api/v1/system/permission/enter-smoke-${Date.now()}`;
    let createdDictId = 0;
    let createdI18nId = 0;
    let createdPermissionId = 0;

    try {
      await page.goto('/system/dict', { waitUntil: 'networkidle' });
      await page.locator('.dict-page__actions').first().getByRole('button', { name: '新增', exact: true }).click();
      const dictDialog = await waitForDialog(page, '新增字典类型');
      const dictInputs = dictDialog.locator('input:not([disabled])');
      await dictInputs.nth(0).fill(dictCode);
      await dictInputs.nth(1).fill(dictName);
      await dictInputs.nth(1).press('Enter');
      await expect(dictDialog).toHaveCount(0);
      await expect(page.locator('.arco-message').getByText('创建成功', { exact: false }).last()).toBeVisible();

      await expect.poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/dict/type/list`, {
          headers: authHeaders(accessToken),
          params: { dictCode },
        });
        const payload = await response.json();
        const item = Array.isArray(payload.data) ? payload.data[0] : undefined;
        createdDictId = Number(item?.id || 0);
        return createdDictId > 0;
      }).toBeTruthy();

      await page.goto('/system/i18n', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const i18nDialog = await waitForDialog(page, '新增翻译');
      const i18nInputs = i18nDialog.locator('input:not([disabled])');
      await i18nInputs.nth(0).fill('system.config');
      await i18nInputs.nth(1).fill('messages');
      await i18nInputs.nth(2).fill(i18nKey);
      await i18nDialog.locator('textarea').first().fill('回车提交文案');
      await i18nInputs.nth(2).press('Enter');
      await expect(i18nDialog).toHaveCount(0);
      await expect(page.locator('.arco-message').getByText('创建成功', { exact: false }).last()).toBeVisible();

      await expect.poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
          headers: authHeaders(accessToken),
          params: { key: i18nKey, page: '1', pageSize: '10' },
        });
        const payload = await response.json();
        const item = payload.data?.items?.[0];
        createdI18nId = Number(item?.id || 0);
        return createdI18nId > 0;
      }).toBeTruthy();

      await page.goto('/system/permission', { waitUntil: 'networkidle' });
      await page.getByRole('tab', { name: '接口策略', exact: true }).click();
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const permissionDialog = await waitForDialog(page, '新增策略');
      await permissionDialog.locator('.arco-select-view').first().click();
      await page.getByRole('option', { name: 'superadmin', exact: true }).click();
      const pathInput = permissionDialog.getByPlaceholder('/api/v1/system/user/list');
      await pathInput.fill(permissionPath);
      await pathInput.press('Enter');
      await expect(permissionDialog).toHaveCount(0);
      await expect(page.locator('.arco-message').getByText('创建成功', { exact: false }).last()).toBeVisible();

      await expect.poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/permission/list`, {
          headers: authHeaders(accessToken),
          params: { path: permissionPath, page: '1', pageSize: '10' },
        });
        const payload = await response.json();
        const item = payload.data?.items?.find((row: { path?: string }) => row.path === permissionPath);
        createdPermissionId = Number(item?.id || 0);
        return createdPermissionId > 0;
      }).toBeTruthy();
    } finally {
      const headers = await verifiedHeaders(page, accessToken);
      if (createdPermissionId > 0) {
        await page.request.delete(`${apiBaseUrl}/system/permission/${createdPermissionId}`, {
          headers,
        }).catch(() => undefined);
      }
      if (createdI18nId > 0) {
        await page.request.delete(`${apiBaseUrl}/system/i18n/${createdI18nId}`, {
          headers,
        }).catch(() => undefined);
      }
      if (createdDictId > 0) {
        await page.request.delete(`${apiBaseUrl}/system/dict/type/${createdDictId}`, {
          headers,
        }).catch(() => undefined);
      }
    }
  });
});


test.describe('UI Structural Audit (Direct CSS/DOM Verification)', () => {
  test('verify global radii and gradient removal', async ({ page }) => {
    await page.goto('/login');
    
    // Audit Login Card
    const loginCard = page.locator('.auth-login-card');
    await expect(loginCard).toBeVisible();
    
    const audit = await loginCard.evaluate((el) => {
      const style = window.getComputedStyle(el);
      return {
        borderRadius: style.borderRadius,
        backgroundImage: style.backgroundImage,
        boxShadow: style.boxShadow
      };
    });

    console.log('Login Card Audit:', audit);
    
    // Radii should be small (4px standard)
    expect(parseInt(audit.borderRadius)).toBeLessThanOrEqual(8);
    // Background should not have linear-gradient (except maybe login grid background on the page, but not the card)
    expect(audit.backgroundImage).toBe('none');
  });

  test('verify modal structural integrity and layering', async ({ page }) => {
    // We can check the CSS directly in the stylesheet if we want, but checking computed is safer
    await page.goto('/login');
    
    const zIndexAudit = await page.evaluate(() => {
      // Create a dummy app-dialog to check its computed style
      const div = document.createElement('div');
      div.className = 'app-dialog';
      div.style.position = 'fixed'; // Required for z-index to matter
      document.body.appendChild(div);
      const style = window.getComputedStyle(div);
      const val = style.zIndex;
      document.body.removeChild(div);
      return val;
    });

    console.log('App Dialog Z-Index:', zIndexAudit);
    expect(parseInt(zIndexAudit)).toBeGreaterThanOrEqual(2500);
  });

  test('verify layout overflow protection (Physical Isolation)', async ({ page }) => {
    await page.goto('/login');
    
    const overflowAudit = await page.evaluate(() => {
      const div = document.createElement('div');
      div.className = 'page-main-column';
      document.body.appendChild(div);
      const style = window.getComputedStyle(div);
      const results = {
        overflow: style.overflow,
        minWidth: style.minWidth
      };
      document.body.removeChild(div);
      return results;
    });

    console.log('Main Column Overflow Audit:', overflowAudit);
    expect(overflowAudit.overflow).toBe('hidden');
    expect(overflowAudit.minWidth).toBe('0px');
  });

  test('verify page panel border strength', async ({ page }) => {
    await page.goto('/login');
    
    const panelAudit = await page.evaluate(() => {
      const div = document.createElement('div');
      div.className = 'page-panel';
      document.body.appendChild(div);
      const style = window.getComputedStyle(div);
      const border = style.border;
      const borderColor = style.borderColor;
      document.body.removeChild(div);
      return { border, borderColor };
    });

    console.log('Panel Border Audit:', panelAudit);
    // Should have 1px solid border
    expect(panelAudit.border).toContain('1px solid');
  });
});
