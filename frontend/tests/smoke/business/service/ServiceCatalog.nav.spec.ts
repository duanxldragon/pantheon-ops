import { expect, type Page, test } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { adminCredentials, signInWithUi } from '../../helpers/auth';

// React root of the sidebar business block driving the side menu. Entries render as
// .app-shell__sider .app-shell__menu-entry--level-0 (roots / groups) and --level-1
// (direct children); the label spans .app-shell__menu-entry-label. There is no
// .arco-menu-item / .arco-menu-inline-header in this layout.
const sider = (page: Page) => page.locator('.app-shell__sider');
const siderLabel = (page: Page) => sider(page).locator('.app-shell__menu-entry-label');

// A compact plan of the business tree: each row is [path, expected body regex]. The
// body regex is a distinctive string the rendered main content must contain after
// the entry finishes loading (mirrors the MenuContract /Codex probe expectations).
// Parent rows navigate to their first navigable child, so their body regex is that
// child's landing content. Rows are ordered to match BACKEND_SEED_ORDER below.
const BUSINESS_ENTRIES: Array<[string, RegExp]> = [
  // cmdb (1)
  ['/business/cmdb', /主机管理/],
  ['/business/cmdb/host', /主机管理/],
  ['/business/cmdb/group', /主机分组/],
  ['/business/cmdb/label', /标签管理/],
  // bizscope (2)
  ['/business/business-scope', /业务域总数/],
  ['/business/business-scope/host-binding', /主机绑定/],
  ['/business/business-scope/environments', /环境管理/],
  // service catalog (3)
  ['/business/service-catalog', /新增应用/],
  ['/business/service-catalog/applications', /新增应用/],
  ['/business/service-catalog/services', /新增服务/],
  ['/business/service-catalog/instances', /新增服务实例/],
  // deploy (4)
  ['/business/deploy', /软件组件/],
  ['/business/deploy/package', /软件组件/],
  ['/business/deploy/template', /任务模板/],
  ['/business/deploy/task', /部署任务/],
  // filedist (5)
  ['/business/filedist/files', /文件库/],
  // k8s (6)
  ['/business/k8s', /K8s 集群/],
  ['/business/k8s/cluster', /K8s 集群/],
  ['/business/k8s/workload', /工作负载/],
  ['/business/k8s/service', /负载均衡服务/],
  ['/business/k8s/release', /创建发布/],
  ['/business/k8s/ingress', /Ingress/],
  ['/business/k8s/storage', /StorageClass/],
  ['/business/k8s/backup', /存储位置/],
  // argocd (7)
  ['/business/cicd/argocd', /服务端/],
  // observability (8)
  ['/business/observability', /监控中心/],
  ['/business/observability/monitoring', /监控中心/],
  ['/business/observability/alerts', /告警规则/],
  ['/business/observability/logs', /最多日志条数/],
  ['/business/observability/traces', /最多展示 100 条链路/],
];

// Backend seed Sort-driven top-level block. Filedist and ArgoCD are roots that sit
// next to K8s before system governance (files=5, k8s=6, argocd=7, observability=8).
const BACKEND_SEED_ORDER = [
  '/business/cmdb',
  '/business/business-scope',
  '/business/service-catalog',
  '/business/deploy',
  '/business/filedist/files',
  '/business/k8s',
  '/business/cicd/argocd',
  '/business/observability',
];

// Resolve the locale source regardless of whether the runner cwd is repo root or
// frontend/. Store a simple key->label map for zh-CN so the sidebar locators address
// entries by their translated label without hardcoding drift-prone text on each row.
function resolveLocaleFile(): string {
  const candidates = [
    path.resolve(process.cwd(), 'frontend', 'src', 'i18n', 'resources', 'business', 'zh-CN.ts'),
    path.resolve(process.cwd(), 'src', 'i18n', 'resources', 'business', 'zh-CN.ts'),
  ];
  const found = candidates.find((file) => fs.existsSync(file));
  if (!found) {
    throw new Error('Unable to locate business zh-CN locale source.');
  }
  return found;
}

const zhLabels = Object.fromEntries(
  [...fs.readFileSync(resolveLocaleFile(), 'utf8').matchAll(/['"]([^'"]+)['"]\s*:\s*['"]([^'"]+)['"]/g)].map(
    (match) => [match[1], match[2]],
  ),
);

interface MenuNodeRow {
  id: number;
  parentId: number | null;
  path?: string;
  titleKey?: string;
  type: 'M' | 'C' | 'B';
  children: MenuNodeRow[];
}

/** Root menu node keyed, honoring titleKey so a translated label can be located. */
async function businessTree(page: Page): Promise<MenuNodeRow[]> {
  const response = await page.request.get('/api/v1/system/menu/tree');
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data ?? [];
}

function menuLabel(labelKey: string | undefined): string {
  const label = labelKey ? zhLabels[labelKey] : undefined;
  if (!label) {
    throw new Error(`No zh-CN label found for titleKey ${labelKey}.`);
  }
  return label;
}

// A level-0 root locator scoped to the sider, addressed by its exact label and
// positioned among same-label roots (there should never be a duplicate, but keep
// the index for a clear failure). Exact text avoids substring collisions such as
// 应用交付 vs 应用 or 服务 vs 服务实例 on the shared surface.
async function rootLocator(
  page: Page,
  node: MenuNodeRow,
  roots: MenuNodeRow[],
): Promise<ReturnType<typeof sider>> {
  const label = menuLabel(node.titleKey);
  const sameLabel = roots.filter((item) => menuLabel(item.titleKey) === label);
  const index = Math.max(0, sameLabel.findIndex((item) => item.id === node.id));
  return sider(page).locator('.app-shell__menu-entry--level-0:visible').getByText(label, { exact: true }).nth(index);
}

async function childLocator(page: Page, node: MenuNodeRow): Promise<ReturnType<typeof sider>> {
  return sider(page)
    .locator('.app-shell__menu-entry--level-1:visible')
    .getByText(menuLabel(node.titleKey), { exact: true });
}

// Scope children to the service catalog submenu so Kubernetes/服务 cannot match.
async function serviceCatalogChild(page: Page, label: string): Promise<ReturnType<typeof sider>> {
  return sider(page)
    .locator('.arco-menu-inline')
    .filter({
      has: page.locator('.app-shell__menu-entry--level-0').getByText('服务目录', { exact: true }),
    })
    .locator('.app-shell__menu-entry--level-1:visible')
    .getByText(label, { exact: true });
}

async function walkEntry(
  page: Page,
  node: MenuNodeRow,
  ancestors: MenuNodeRow[],
  roots: MenuNodeRow[],
  expectBody: RegExp,
) {
  // Start from a known idle state so group expansion does not stack.
  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
  await page.getByText('高频管理入口', { exact: true }).waitFor({ timeout: 20000 });

  // Expand each ancestor level-0 group first.
  for (const parent of ancestors) {
    await (await rootLocator(page, parent, roots)).click({ timeout: 8000 });
    await page.waitForTimeout(150);
  }

  // Click the target: a root navigates directly; a leaf expands via its parent above.
  if (node.parentId) {
    await (await childLocator(page, node)).click({ timeout: 8000 });
  } else {
    await (await rootLocator(page, node, roots)).click({ timeout: 8000 });
  }
  await page.waitForURL((url) => node.type === 'M'
    ? url.pathname.startsWith(`${node.path}/`)
    : url.pathname === node.path, { timeout: 15000 });
  if (node.path === '/business/deploy/task') {
    const title = page.locator('main .governance-summary-bar__title');
    await expect(title).toHaveText('部署任务', { timeout: 15000 });
    await expect(title).toBeVisible();
  }
  await expect(page.locator('main'), `main content for ${node.path}`).toContainText(
    expectBody,
    { timeout: 15000 },
  );
  await expect(page.locator('main')).not.toContainText('高频管理入口');
}

test.describe.serial('Business navigation repair', () => {
  test('sidebar business block is contiguous, one 服务目录, distinct ArgoCD label', async ({ page }) => {
    await signInWithUi(page, adminCredentials);
    await expect(siderLabel(page).first()).toBeVisible({ timeout: 15000 });

    // Exactly one 服务目录 entry: the legacy id159 duplicate was merged into it.
    await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);

    // ArgoCD root carries the full 应用交付 label (it is not bare 应用交付).
    await expect(sider(page).getByText('ArgoCD 应用交付', { exact: true })).toHaveCount(1);

    // The only 应用发布 is the K8s release child, but the K8s group starts
    // collapsed — expand it first so the level-1 entry renders before asserting.
    await sider(page)
      .locator('.app-shell__menu-entry--level-0')
      .getByText('Kubernetes', { exact: true })
      .click({ timeout: 8000 });
    await expect(sider(page).getByText('应用发布', { exact: true })).toHaveCount(1);

    // Top-level roots (level-0) appear in DOM order as the sidebar block: a platform
    // workbench root (e.g. 工作台) precedes the business run, which must sit as a
    // contiguous run in backend seed Sort (files=5, k8s=6, argocd=7, obs=8), ahead
    // of the system governance roots.
    const rootLabels = (await sider(page)
      .locator('.app-shell__menu-entry--level-0 .app-shell__menu-entry-label')
      .allTextContents()).map((text) => text.replace(/\s+/g, ' ').trim());

    const tree = await businessTree(page);
    const roots = tree.filter((node) => node.path?.startsWith('/business/'));
    const ordinal = (p: string) => BACKEND_SEED_ORDER.indexOf(p);
    const ordered = [...roots].sort((a, b) => {
      const diff = ordinal(a.path ?? '') - ordinal(b.path ?? '');
      return diff !== 0 ? diff : a.id - b.id;
    });
    const expectedRootLabels = ordered.map((node) => menuLabel(node.titleKey));

    // The business run starts at the CMDB root (a workbench root precedes it, so it
    // is not at index 0) and must be the exact, contiguous 8-label seed-order slice.
    const bizStart = rootLabels.indexOf(expectedRootLabels[0]);
    expect(bizStart, 'business block should begin at the CMDB root').toBeGreaterThanOrEqual(0);
    expect(rootLabels.slice(bizStart, bizStart + expectedRootLabels.length)).toEqual(expectedRootLabels);

    // Nothing business-shaped recurs after the block, so system governance follows.
    const afterBiz = rootLabels.slice(bizStart + expectedRootLabels.length);
    for (const label of Array.from(new Set(expectedRootLabels))) {
      expect(afterBiz, `business root ${label} must not recur after the contiguous block`).not.toContain(label);
    }
  });

  test('every business parent/leaf entry loads the expected main content', async ({ page }) => {
    test.setTimeout(180000);
    await signInWithUi(page, adminCredentials);
    const tree = await businessTree(page);
    const businessRoots = tree.filter((node) => node.path?.startsWith('/business/'));
    // Index live rows by path and record each node's parent path so the walk can
    // rebuild its ancestor chain top-down for a given route.
    const byPath = new Map<string, MenuNodeRow>();
    const parentPath = new Map<string, string | null>();
    const flatten = (nodes: MenuNodeRow[], parent: string | null) => {
      for (const node of nodes) {
        const route = node.path ?? '';
        byPath.set(route, node);
        parentPath.set(route, parent);
        flatten(node.children ?? [], route);
      }
    };
    flatten(businessRoots, null);

    // Iterate the 30-stop plan and require every seed path to resolve to a live row
    // and be navigated — a missing seed is a hard failure, never a silent skip.
    for (const [route, body] of BUSINESS_ENTRIES) {
      const node = byPath.get(route);
      if (!node) {
        throw new Error(`seed menu row ${route} is missing from the live menu tree`);
      }
      if (!node.titleKey) {
        throw new Error(`seed menu row ${route} lacks a titleKey`);
      }
      const ancestors: MenuNodeRow[] = [];
      let cursor = parentPath.get(route);
      while (cursor) {
        const parent = byPath.get(cursor);
        if (parent) ancestors.unshift(parent);
        cursor = parentPath.get(cursor);
      }
      await test.step(`click ${route}`, async () => {
        try {
          await walkEntry(page, node, ancestors, businessRoots, body);
        } catch (error) {
          throw new Error(`Failed navigating to ${route}: ${(error as Error).message}`, { cause: error });
        }
      });
    }
  });

  test('ServiceCatalog sidebar 3 items, page tabs, back/forward, reload', async ({ page }) => {
    const badNav: string[] = [];
    page.on('response', (resp) => {
      if (resp.status() === 404 && resp.request().url().includes('/business/service')) badNav.push(resp.url());
    });

    test.setTimeout(90000);
    await signInWithUi(page, adminCredentials);
    // Sidebar shows the service catalog group with its three direct children.
    await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
    await sider(page).getByText('服务目录', { exact: true }).first().click({ timeout: 8000 });

    const catalogCases = [
      { key: 'applications', label: '应用', button: '新增应用' },
      { key: 'services', label: '服务', button: '新增服务' },
      { key: 'instances', label: '服务实例', button: '新增服务实例' },
    ] as const;

    // Locators scoped to MAIN so top-level header tabs with the same name can't match.
    const assertOn = async (item: (typeof catalogCases)[number]) => {
      await expect(page).toHaveURL(new RegExp(`/business/service-catalog/${item.key}$`));
      const main = page.locator('main');
      await expect(main.getByRole('button', { name: item.button, exact: true })).toBeVisible({ timeout: 15000 });
      await expect(main.getByRole('tab', { name: item.label, exact: true })).toHaveAttribute('aria-selected', 'true');
    };

    // Sidebar: really clicking each of the three level-1 children loads its own page
    // and shows the matching 新增 button. Scoped to the 服务目录 ancestor so the
    // Kubernetes 服务 child cannot collide in strict mode.
    for (const item of catalogCases) {
      await test.step(`sidebar click ${item.key}`, async () => {
        await (await serviceCatalogChild(page, item.label)).click({ timeout: 8000 });
        await assertOn(item);
      });
    }

    // In-page tabs drive the URL and write it back for all three.
    for (const item of catalogCases) {
      await test.step(`page tab ${item.label}`, async () => {
        await page.locator('main').getByRole('tab', { name: item.label, exact: true }).click({ timeout: 8000 });
        await assertOn(item);
      });
    }

    // Browser back / forward stay consistent with the URL-driven active tab: the tab
    // loop above ends on 服务实例, so back lands on 服务 and forward re-enters 服务实例.
    await page.goBack();
    await assertOn(catalogCases[1]);
    await page.goForward();
    await assertOn(catalogCases[2]);

    // Deep-link + real reload keeps the URL-driven tab for all three.
    for (const item of catalogCases) {
      await test.step(`deep link + reload ${item.key}`, async () => {
        await page.goto(`/business/service-catalog/${item.key}`, { waitUntil: 'domcontentloaded' });
        await assertOn(item);
        await page.reload({ waitUntil: 'domcontentloaded' });
        await assertOn(item);
      });
    }

    expect(badNav).toEqual([]);
  });
});
