import fs from 'node:fs/promises';
import path from 'node:path';
import { chromium, request } from 'playwright';

const frontendBaseUrl = 'http://127.0.0.1:5173';
const apiBaseUrl = 'http://127.0.0.1:8080/api/v1';
const artifactDir = path.join(process.cwd(), 'test-results', 'narrow-table-layout');

const viewports = [
  { width: 1280, height: 900, slug: '1280' },
  { width: 1024, height: 900, slug: '1024' },
  { width: 900, height: 900, slug: '900' },
];

const pages = [
  { domain: 'system/iam', path: '/system/user', title: '用户管理', slug: 'system-user' },
  { domain: 'system/iam', path: '/system/role', title: '角色管理', slug: 'system-role' },
  { domain: 'system/iam', path: '/system/permission', title: '权限管理', slug: 'system-permission' },
  { domain: 'system/iam', path: '/system/menu', title: '菜单管理', slug: 'system-menu' },
  { domain: 'system/org', path: '/system/dept', title: '部门管理', slug: 'system-dept' },
  { domain: 'system/org', path: '/system/post', title: '岗位管理', slug: 'system-post' },
  { domain: 'system/config', path: '/system/dict', title: '字典管理', slug: 'system-dict' },
  { domain: 'system/config', path: '/system/setting', title: '系统设置', slug: 'system-setting' },
  { domain: 'system/config', path: '/system/i18n', title: '国际化管理', slug: 'system-i18n' },
  { domain: 'system/config', path: '/system/modules', title: '模块注册表', slug: 'system-modules' },
];

async function ensureDir(dir) {
  await fs.mkdir(dir, { recursive: true });
}

async function loginByApi() {
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
  await apiContext.dispose();
  if (payload.code !== 200 || !payload.data?.accessToken || !payload.data?.refreshToken) {
    throw new Error(`unexpected login payload: ${JSON.stringify(payload)}`);
  }
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

async function collectPageMetrics(page, meta, viewport) {
  await page.goto(`${frontendBaseUrl}${meta.path}`, { waitUntil: 'networkidle', timeout: 30000 });
  await page.waitForTimeout(300);

  const pageMetrics = await page.evaluate(({ expectedTitle, viewportWidth }) => {
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

    const overflowNodes = Array.from(document.querySelectorAll('*')).filter((node) => {
      if (!(node instanceof HTMLElement) || !isVisible(node)) {
        return false;
      }
      const style = window.getComputedStyle(node);
      if (style.overflowX === 'visible') {
        return false;
      }
      return node.scrollWidth - node.clientWidth > 12;
    });

    const tableElement = Array.from(document.querySelectorAll('table')).find((table) => isVisible(table));
    const pageHeading = document.querySelector('.page-header h1, .page-header [role="heading"], h1, h2');
    const headingText = pageHeading?.textContent?.trim() ?? '';
    const pageErrors = Array.from(document.querySelectorAll('body *'))
      .filter((node) => node instanceof HTMLElement && isVisible(node))
      .map((node) => node.textContent?.trim() ?? '')
      .filter((text) => text && ['加载失败', '网络异常', '请求超时', '500'].some((token) => text.includes(token)))
      .slice(0, 6);

    const filterBar = document.querySelector('.system-list__filters, .page-filter, .page-panel .arco-form');
    const filterRect = filterBar instanceof HTMLElement ? filterBar.getBoundingClientRect() : null;
    const toolbar = document.querySelector('.system-list__toolbar, .page-header__actions, .page-panel__header-extra');
    const toolbarRect = toolbar instanceof HTMLElement ? toolbar.getBoundingClientRect() : null;

    const result = {
      headingMatches: headingText.includes(expectedTitle),
      headingText,
      viewportWidth,
      pageErrors,
      overflowNodeCount: overflowNodes.length,
      filterWrapRisk: Boolean(filterRect && filterRect.height > 120),
      toolbarWrapRisk: Boolean(toolbarRect && toolbarRect.height > 72),
      table: null,
    };

    if (!(tableElement instanceof HTMLTableElement)) {
      return result;
    }

    const scrollContainer =
      tableElement.closest('.arco-table-content') ||
      tableElement.closest('.arco-table-body') ||
      tableElement.parentElement;
    const headerCells = Array.from(tableElement.querySelectorAll('thead th')).filter((cell) => isVisible(cell));
    const row = Array.from(tableElement.querySelectorAll('tbody tr')).find((candidate) => isVisible(candidate));
    const bodyCells = row ? Array.from(row.children).filter((cell) => cell instanceof HTMLTableCellElement && isVisible(cell)) : [];
    const actionIndex = headerCells.findIndex((cell) => /(操作|operations?)/i.test(cell.textContent || ''));
    const hiddenHeaders = Array.from(tableElement.querySelectorAll('thead th'))
      .filter((cell) => cell instanceof HTMLElement)
      .map((cell) => {
        const classes = cell.className || '';
        if (classes.includes('app-table__col--hide-md')) {
          return { text: (cell.textContent || '').trim(), tier: 'md', hidden: !isVisible(cell) };
        }
        if (classes.includes('app-table__col--hide-lg')) {
          return { text: (cell.textContent || '').trim(), tier: 'lg', hidden: !isVisible(cell) };
        }
        return null;
      })
      .filter(Boolean);

    const scrollMetrics =
      scrollContainer instanceof HTMLElement
        ? {
            clientWidth: Math.round(scrollContainer.clientWidth),
            scrollWidth: Math.round(scrollContainer.scrollWidth),
            scrollLeft: Math.round(scrollContainer.scrollLeft),
            hasHorizontalScroll: scrollContainer.scrollWidth - scrollContainer.clientWidth > 12,
            overflowX: window.getComputedStyle(scrollContainer).overflowX,
          }
        : null;

    let actionMetrics = null;
    if (actionIndex >= 0 && bodyCells[actionIndex] instanceof HTMLElement) {
      const actionCell = bodyCells[actionIndex];
      const buttonRects = Array.from(actionCell.querySelectorAll('button'))
        .filter((button) => isVisible(button))
        .map((button) => {
          const rect = button.getBoundingClientRect();
          return {
            text: (button.textContent || '').trim(),
            left: Math.round(rect.left),
            right: Math.round(rect.right),
            top: Math.round(rect.top),
            bottom: Math.round(rect.bottom),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
          };
        });

      const rowLines = new Set(buttonRects.map((item) => item.top));
      actionMetrics = {
        buttonCount: buttonRects.length,
        buttonTexts: buttonRects.map((item) => item.text).filter(Boolean),
        wraps: rowLines.size > 1,
        hiddenByViewport: buttonRects.some((item) => item.left < 0 || item.right > viewportWidth),
      };
    }

    result.table = {
      visibleHeaderCount: headerCells.length,
      hiddenHeaders,
      scrollMetrics,
      actionIndex,
      actionMetrics,
    };

    return result;
  }, { expectedTitle: meta.title, viewportWidth: viewport.width });

  const screenshotPath = path.join(
    artifactDir,
    `${meta.slug}-${viewport.slug}.png`,
  );
  await page.screenshot({ path: screenshotPath, fullPage: true });

  await page.evaluate(() => {
    const nodes = Array.from(document.querySelectorAll('.arco-table-content, .arco-table-body, .arco-table-container'));
    nodes.forEach((node) => {
      if (node instanceof HTMLElement) {
        node.scrollLeft = node.scrollWidth;
      }
    });
    window.scrollTo({ top: document.body.scrollHeight, behavior: 'instant' });
  });
  await page.waitForTimeout(200);

  const rightEdgeMetrics = await page.evaluate(({ viewportWidth }) => {
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

    const table = Array.from(document.querySelectorAll('table')).find((candidate) => isVisible(candidate));
    if (!(table instanceof HTMLTableElement)) {
      return null;
    }

    const headerCells = Array.from(table.querySelectorAll('thead th')).filter((cell) => isVisible(cell));
    const actionIndex = headerCells.findIndex((cell) => /(操作|operations?)/i.test(cell.textContent || ''));
    if (actionIndex < 0) {
      return null;
    }

    const row = Array.from(table.querySelectorAll('tbody tr')).find((candidate) => isVisible(candidate));
    if (!(row instanceof HTMLTableRowElement)) {
      return null;
    }

    const cells = Array.from(row.children).filter((cell) => cell instanceof HTMLTableCellElement && isVisible(cell));
    const actionCell = cells[actionIndex];
    if (!(actionCell instanceof HTMLElement)) {
      return null;
    }

    const rect = actionCell.getBoundingClientRect();
    const buttons = Array.from(actionCell.querySelectorAll('button'))
      .filter((button) => isVisible(button))
      .map((button) => {
        const buttonRect = button.getBoundingClientRect();
        return {
          text: (button.textContent || '').trim(),
          left: Math.round(buttonRect.left),
          right: Math.round(buttonRect.right),
        };
      });

    return {
      cellLeft: Math.round(rect.left),
      cellRight: Math.round(rect.right),
      cellWidth: Math.round(rect.width),
      buttons,
      allButtonsInsideViewport: buttons.every((button) => button.left >= 0 && button.right <= viewportWidth),
    };
  }, { viewportWidth: viewport.width });

  return {
    ...pageMetrics,
    rightEdgeMetrics,
    screenshotPath,
  };
}

function deriveIssues(result) {
  const issues = [];
  if (!result.headingMatches) {
    issues.push(`heading-mismatch:${result.headingText}`);
  }
  if (result.pageErrors.length > 0) {
    issues.push(`page-error:${result.pageErrors.join('|')}`);
  }
  if (!result.table) {
    issues.push('table-missing');
    return issues;
  }

  const { scrollMetrics, actionMetrics, hiddenHeaders } = result.table;
  if (scrollMetrics?.hasHorizontalScroll && scrollMetrics.overflowX === 'hidden') {
    issues.push('table-scroll-hidden');
  }
  if (scrollMetrics?.hasHorizontalScroll === false && result.overflowNodeCount > 0) {
    issues.push('page-overflow-without-table-scroll');
  }
  if (actionMetrics?.hiddenByViewport) {
    issues.push('action-buttons-hidden-before-scroll');
  }
  if (actionMetrics?.wraps) {
    issues.push('action-buttons-wrapped');
  }
  if (result.rightEdgeMetrics && !result.rightEdgeMetrics.allButtonsInsideViewport) {
    issues.push('action-buttons-hidden-after-scroll');
  }
  if (result.filterWrapRisk) {
    issues.push('filter-wrap-risk');
  }
  if (result.toolbarWrapRisk) {
    issues.push('toolbar-wrap-risk');
  }
  if (!hiddenHeaders.some((item) => item.hidden)) {
    issues.push('no-responsive-column-downgrade');
  }
  return issues;
}

async function main() {
  await ensureDir(artifactDir);
  const tokens = await loginByApi();
  const browser = await chromium.launch({ headless: true });
  const report = [];

  for (const viewport of viewports) {
    const context = await browser.newContext({ viewport: { width: viewport.width, height: viewport.height } });
    const page = await context.newPage();
    await installSession(page, tokens);

    for (const meta of pages) {
      try {
        const result = await collectPageMetrics(page, meta, viewport);
        report.push({
          domain: meta.domain,
          page: meta.path,
          title: meta.title,
          viewport,
          issues: deriveIssues(result),
          result,
        });
      } catch (error) {
        report.push({
          domain: meta.domain,
          page: meta.path,
          title: meta.title,
          viewport,
          issues: ['script-error'],
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    await context.close();
  }

  await browser.close();

  const reportPath = path.join(artifactDir, 'report.json');
  await fs.writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`, 'utf8');

  for (const item of report) {
    console.log(JSON.stringify({
      domain: item.domain,
      page: item.page,
      viewport: item.viewport.width,
      issues: item.issues,
      error: item.error ?? null,
      screenshotPath: item.result?.screenshotPath ?? null,
    }));
  }
  console.log(`REPORT ${reportPath}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
