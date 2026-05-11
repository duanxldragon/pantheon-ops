import fs from 'node:fs/promises';
import path from 'node:path';
import { chromium, request } from 'playwright';

const frontendBaseUrl = 'http://127.0.0.1:5173';
const apiBaseUrl = 'http://127.0.0.1:8080/api/v1';
const artifactDir = path.join(process.cwd(), 'test-results', 'action-layout-review');

const pages = [
  { path: '/system/user', title: '用户管理', slug: 'system-user' },
  { path: '/system/role', title: '角色管理', slug: 'system-role' },
  { path: '/system/menu', title: '菜单管理', slug: 'system-menu' },
  { path: '/system/dept', title: '部门管理', slug: 'system-dept' },
  { path: '/system/post', title: '岗位管理', slug: 'system-post' },
  { path: '/system/permission', title: '权限管理', slug: 'system-permission' },
  { path: '/system/dict', title: '字典管理', slug: 'system-dict' },
  { path: '/system/setting', title: '系统设置', slug: 'system-setting' },
  { path: '/system/i18n', title: '国际化管理', slug: 'system-i18n' },
  { path: '/system/session', title: '会话管理', slug: 'system-session' },
  { path: '/system/operation-log', title: '操作日志', slug: 'system-operation-log' },
  { path: '/system/modules', title: '模块注册表', slug: 'system-modules' },
  { path: '/auth/security', title: '安全中心', slug: 'auth-security' },
];

async function ensureDir(dir) {
  await fs.mkdir(dir, { recursive: true });
}

async function signInAsAdmin() {
  const apiContext = await request.newContext();
  const response = await apiContext.post(`${apiBaseUrl}/auth/login`, {
    data: {
      username: 'admin',
      password: '123456',
    },
  });
  if (!response.ok()) {
    throw new Error(`login failed: ${response.status()} ${response.statusText()}`);
  }
  const payload = await response.json();
  if (payload.code !== 200 || !payload.data?.accessToken || !payload.data?.refreshToken) {
    throw new Error(`unexpected login payload: ${JSON.stringify(payload)}`);
  }
  await apiContext.dispose();
  return {
    accessToken: payload.data.accessToken,
    refreshToken: payload.data.refreshToken,
  };
}

async function installSession(page, tokens) {
  await page.goto(`${frontendBaseUrl}/login`, { waitUntil: 'domcontentloaded' });
  await page.evaluate(({ accessToken, refreshToken }) => {
    localStorage.setItem('pantheon_access_token', accessToken);
    localStorage.setItem('pantheon_refresh_token', refreshToken);
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
    sessionStorage.removeItem('pantheon_op_token');
  }, tokens);
}

async function inspectActionCell(page) {
  const metrics = await page.evaluate(() => {
    const isVisible = (element) => {
      if (!(element instanceof HTMLElement)) {
        return false;
      }
      const style = window.getComputedStyle(element);
      if (style.display === 'none' || style.visibility === 'hidden' || Number(style.opacity) === 0) {
        return false;
      }
      const rect = element.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0;
    };

    const clearMarkers = () => {
      document.querySelectorAll('[data-qa-action-cell]').forEach((node) => {
        node.removeAttribute('data-qa-action-cell');
      });
    };

    clearMarkers();

    const headerPattern = /(操作|operations?)/i;
    const tables = Array.from(document.querySelectorAll('table'));

    for (const table of tables) {
      const headerCells = Array.from(table.querySelectorAll('thead th'));
      const actionIndex = headerCells.findIndex((cell) => isVisible(cell) && headerPattern.test(cell.textContent || ''));
      if (actionIndex < 0) {
        continue;
      }

      const rows = Array.from(table.querySelectorAll('tbody tr')).filter((row) => isVisible(row));
      for (const row of rows) {
        const cells = Array.from(row.children).filter((cell) => cell instanceof HTMLTableCellElement);
        const actionCell = cells[actionIndex];
        if (!(actionCell instanceof HTMLElement) || !isVisible(actionCell)) {
          continue;
        }

        const buttons = Array.from(actionCell.querySelectorAll('button')).filter((button) => isVisible(button));
        if (buttons.length === 0) {
          continue;
        }

        actionCell.setAttribute('data-qa-action-cell', 'true');
        const cellRect = actionCell.getBoundingClientRect();
        const buttonRects = buttons.map((button) => button.getBoundingClientRect());
        const topRows = new Set(buttonRects.map((rect) => Math.round(rect.top)));
        const buttonTexts = buttons.map((button) => (button.textContent || '').trim()).filter(Boolean);
        const classNames = Array.from(actionCell.querySelectorAll('[class]'))
          .map((node) => node.className)
          .filter((value) => typeof value === 'string');

        return {
          found: true,
          headerText: (headerCells[actionIndex]?.textContent || '').trim(),
          buttonCount: buttons.length,
          buttonTexts,
          cellHeight: Math.round(cellRect.height),
          cellWidth: Math.round(cellRect.width),
          rowCount: topRows.size,
          wraps: topRows.size > 1,
          overflowX: actionCell.scrollWidth > actionCell.clientWidth,
          overflowY: actionCell.scrollHeight > actionCell.clientHeight,
          usesSystemActionClass: classNames.some((value) => value.includes('system-list__actions')),
        };
      }
    }

    return {
      found: false,
      reason: 'no action cell with visible buttons found',
    };
  });

  if (!metrics.found) {
    return metrics;
  }

  const locator = page.locator('[data-qa-action-cell="true"]').first();
  await locator.scrollIntoViewIfNeeded();
  const box = await locator.boundingBox();
  if (!box) {
    return {
      ...metrics,
      found: false,
      reason: 'action cell bounding box unavailable',
    };
  }

  return {
    ...metrics,
    clip: {
      x: Math.max(0, Math.floor(box.x - 16)),
      y: Math.max(0, Math.floor(box.y - 10)),
      width: Math.ceil(box.width + 32),
      height: Math.ceil(box.height + 20),
    },
  };
}

async function prepareActionTableFallback(page) {
  await page.evaluate(() => {
    window.scrollTo({ top: document.body.scrollHeight, behavior: 'instant' });
    const scrollContainers = Array.from(document.querySelectorAll('.arco-table-body, .arco-table-content, .arco-table-container'));
    scrollContainers.forEach((node) => {
      if (node instanceof HTMLElement) {
        node.scrollLeft = node.scrollWidth;
      }
    });
  });
  await page.waitForTimeout(200);
}

async function main() {
  await ensureDir(artifactDir);
  const tokens = await signInAsAdmin();
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  await installSession(page, tokens);

  const report = [];

  for (const meta of pages) {
    const fullPath = path.join(artifactDir, `${meta.slug}.png`);
    const actionPath = path.join(artifactDir, `${meta.slug}-action.png`);
    const entry = { ...meta, screenshot: fullPath, actionScreenshot: null, check: null, error: null };

    try {
      await page.goto(`${frontendBaseUrl}${meta.path}`, { waitUntil: 'networkidle', timeout: 30000 });
      await page.screenshot({ path: fullPath, fullPage: true });
      let check = await inspectActionCell(page);
      if (!check.found) {
        await prepareActionTableFallback(page);
        check = await inspectActionCell(page);
      }
      entry.check = check;
      if (check.found && check.clip) {
        await page.screenshot({ path: actionPath, clip: check.clip });
        entry.actionScreenshot = actionPath;
      }
    } catch (error) {
      entry.error = error instanceof Error ? error.message : String(error);
    }

    report.push(entry);
  }

  await fs.writeFile(
    path.join(artifactDir, 'report.json'),
    `${JSON.stringify(report, null, 2)}\n`,
    'utf8',
  );

  await browser.close();

  for (const item of report) {
    console.log(JSON.stringify(item));
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
