import { chromium } from 'playwright';

const BASE = 'http://localhost:5173';
const pages = [
  { name: 'user', path: '/system/user', desc: 'User Management' },
  { name: 'role', path: '/system/role', desc: 'Role Management' },
  { name: 'dept', path: '/system/dept', desc: 'Department Management' },
  { name: 'menu', path: '/system/menu', desc: 'Menu Management' },
  { name: 'permission', path: '/system/permission', desc: 'Permission Management' },
  { name: 'post', path: '/system/post', desc: 'Post Management' },
  { name: 'setting', path: '/system/setting', desc: 'System Settings' },
  { name: 'operation-log', path: '/system/operation-log', desc: 'Operation Logs' },
  { name: 'dict', path: '/system/dict', desc: 'Dictionary Management' },
  { name: 'i18n', path: '/system/i18n', desc: 'i18n Management' },
  { name: 'modules', path: '/system/modules', desc: 'Module Manager' },
];

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

// Login
console.log('=== LOGGING IN ===');
await page.goto(BASE + '/login', { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(1000);
await page.locator('input[placeholder="请输入用户名"]').waitFor({ timeout: 10000 });
await page.locator('input[placeholder="请输入用户名"]').fill('admin');
await page.locator('input[placeholder="请输入密码"]').fill('123456');
await page.locator('button[type="submit"]').first().click();
await page.waitForTimeout(2000);
await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});

const allIssues = [];

for (const p of pages) {
  console.log(`\n===== ${p.name}: ${p.desc} =====`);
  await page.goto(BASE + p.path, { waitUntil: 'networkidle', timeout: 20000 });
  await page.waitForTimeout(1500);

  const info = await page.evaluate(() => {
    const r = {
      // 1. Container border analysis
      cardBorders: [],
      panelBorders: [],
      // 2. Toolbar positioning
      filterPanel: null,
      tableHead: null,
      batchBar: null,
      overlaps: [],
      // 3. Button style consistency
      primaryBtns: [],
      dangerBtns: [],
      textBtns: [],
    };

    // === 1. All card/panel borders ===
    const cards = document.querySelectorAll('.arco-card, [class*="card"], [class*="panel"]');
    const seenBorders = new Set();
    for (const card of cards) {
      const cs = getComputedStyle(card);
      const borderKey = cs.borderWidth + '|' + cs.borderStyle + '|' + cs.borderColor + '|' + cs.borderRadius;
      if (!seenBorders.has(borderKey) && cs.borderWidth !== '0px') {
        seenBorders.add(borderKey);
        r.cardBorders.push({
          selector: card.className?.slice(0, 50) || card.tagName,
          width: cs.borderWidth,
          style: cs.borderStyle,
          color: cs.borderColor,
          radius: cs.borderRadius,
          bg: cs.backgroundColor?.slice(0, 40),
        });
      }
      if (r.cardBorders.length >= 15) break;
    }

    // Also check specific panels
    const pagePanels = document.querySelectorAll('.page-panel, .filter-panel, .system-page-hero');
    for (const panel of pagePanels) {
      const cs = getComputedStyle(panel);
      r.panelBorders.push({
        selector: panel.className?.slice(0, 60),
        width: cs.borderWidth,
        style: cs.borderStyle,
        color: cs.borderColor,
        radius: cs.borderRadius,
        padding: cs.padding,
      });
    }

    // === 2. Toolbar / filter panel positioning ===
    const filterPanel = document.querySelector('.filter-panel, .arco-card');
    const tableHead = document.querySelector('.system-list__table-head, .list-header-actions');
    const batchBar = document.querySelector('.table-batch-action-bar');

    if (filterPanel) {
      const rect = filterPanel.getBoundingClientRect();
      r.filterPanel = { top: Math.round(rect.top), left: Math.round(rect.left), width: Math.round(rect.width), height: Math.round(rect.height), visible: rect.height > 0 };
    }
    if (tableHead) {
      const rect = tableHead.getBoundingClientRect();
      r.tableHead = { top: Math.round(rect.top), left: Math.round(rect.left), width: Math.round(rect.width), height: Math.round(rect.height), visible: rect.height > 0 };
    }
    if (batchBar) {
      const rect = batchBar.getBoundingClientRect();
      r.batchBar = { top: Math.round(rect.top), left: Math.round(rect.left), width: Math.round(rect.width), height: Math.round(rect.height), visible: rect.height > 0 && rect.width > 0 };
    }

    // Check for overlapping elements in the toolbar area
    const headerArea = document.querySelector('.system-page-hero, .arco-layout-content');
    if (headerArea) {
      const headerRect = headerArea.getBoundingClientRect();
      const children = headerArea.querySelectorAll('.arco-card, .filter-panel, .system-list__table-head, .list-header-actions, .table-batch-action-bar, .arco-space');
      for (let i = 0; i < children.length; i++) {
        for (let j = i + 1; j < children.length; j++) {
          const a = children[i].getBoundingClientRect();
          const b = children[j].getBoundingClientRect();
          if (a.height > 0 && b.height > 0) {
            const overlapX = Math.max(0, Math.min(a.right, b.right) - Math.max(a.left, b.left));
            const overlapY = Math.max(0, Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top));
            if (overlapX > 10 && overlapY > 5) {
              r.overlaps.push({
                a: children[i].className?.slice(0, 40),
                b: children[j].className?.slice(0, 40),
                overlapX: Math.round(overlapX),
                overlapY: Math.round(overlapY),
              });
            }
          }
        }
      }
    }

    // === 3. Button style sampling ===
    const primaryBtns = document.querySelectorAll('.arco-btn-primary');
    for (const btn of primaryBtns) {
      if (r.primaryBtns.length >= 3) break;
      const cs = getComputedStyle(btn);
      r.primaryBtns.push({ text: btn.textContent?.trim().slice(0, 20), bg: cs.backgroundColor, color: cs.color, radius: cs.borderRadius, height: cs.height, fontSize: cs.fontSize });
    }

    const dangerBtns = document.querySelectorAll('.arco-btn-status-danger');
    for (const btn of dangerBtns) {
      if (r.dangerBtns.length >= 3) break;
      const cs = getComputedStyle(btn);
      r.dangerBtns.push({ text: btn.textContent?.trim().slice(0, 20), bg: cs.backgroundColor, color: cs.color, radius: cs.borderRadius, height: cs.height });
    }

    return r;
  });

  // Print findings
  console.log(`  Panels:`);
  for (const p of info.panelBorders) {
    console.log(`    ${p.selector}: border=${p.width} ${p.style} radius=${p.radius} padding=${p.padding}`);
  }

  if (info.cardBorders.length > 0) {
    console.log(`  Card border variations (${info.cardBorders.length} unique):`);
    for (const c of info.cardBorders) {
      console.log(`    ${c.selector}: w=${c.width} s=${c.style} c=${c.color?.slice(0,30)} r=${c.radius} bg=${c.bg}`);
    }
  }

  if (info.overlaps.length > 0) {
    console.log(`  ⚠ OVERLAPS detected:`);
    for (const o of info.overlaps) {
      console.log(`    "${o.a}" OVER "${o.b}" by ${o.overlapX}x${o.overlapY}px`);
    }
  }

  if (info.primaryBtns.length > 0) {
    console.log(`  Primary buttons:`);
    for (const b of info.primaryBtns) {
      console.log(`    "${b.text}": bg=${b.bg} radius=${b.radius} h=${b.height} fs=${b.fontSize}`);
    }
  }

  // Accumulate issues
  allIssues.push({
    page: p.name,
    desc: p.desc,
    panelBorderCount: info.panelBorders.length,
    cardBorderVariations: info.cardBorders.length,
    overlaps: info.overlaps.length,
    filterVisible: info.filterPanel?.visible,
    tableHeadVisible: info.tableHead?.visible,
    batchBarVisible: info.batchBar?.visible,
  });

  // Screenshot
  await page.screenshot({ path: `detail-${p.name}.png`, fullPage: true });
}

// Check modals on user page (open create dialog)
console.log('\n===== MODAL CHECK: User Create Dialog =====');
await page.goto(BASE + '/system/user', { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(1500);

// Click "新增" button to open modal
const addBtn = page.locator('button:has-text("新增")').first();
if (await addBtn.count() > 0) {
  await addBtn.click();
  await page.waitForTimeout(1500);

  const modalInfo = await page.evaluate(() => {
    const r = {};
    const modal = document.querySelector('.arco-modal');
    if (!modal) return { modalFound: false };

    const cs = getComputedStyle(modal);
    r.modalFound = true;
    r.width = cs.width;
    r.borderRadius = cs.borderRadius;

    const content = modal.querySelector('.arco-modal-content');
    if (content) {
      const ccs = getComputedStyle(content);
      r.contentBg = ccs.backgroundColor;
      r.contentPadding = ccs.padding;
    }

    const header = modal.querySelector('.arco-modal-header');
    if (header) {
      const hcs = getComputedStyle(header);
      r.headerBorder = hcs.borderBottom;
      r.headerPadding = hcs.padding;
    }

    const footer = modal.querySelector('.arco-modal-footer');
    if (footer) {
      const fcs = getComputedStyle(footer);
      r.footerBorder = fcs.borderTop;
      r.footerPadding = fcs.padding;
    }

    // Form inside modal
    const form = modal.querySelector('.arco-form');
    if (form) {
      const fcs = getComputedStyle(form);
      r.formPadding = fcs.padding;
    }

    // Buttons in footer
    const footerBtns = footer ? [...footer.querySelectorAll('.arco-btn')].map(b => ({
      text: b.textContent?.trim().slice(0, 20),
      type: b.className?.includes('primary') ? 'primary' : 'default',
      height: getComputedStyle(b).height,
    })) : [];

    r.footerBtns = footerBtns;
    return r;
  });

  console.log(JSON.stringify(modalInfo, null, 2));
  await page.screenshot({ path: 'detail-modal-create.png', fullPage: true });

  // Close modal
  await page.keyboard.press('Escape');
  await page.waitForTimeout(500);
} else {
  console.log('No "新增" button found on user page');
}

// Check delete confirmation popconfirm
console.log('\n===== POPCONFIRM CHECK =====');
const deleteBtn = page.locator('.system-list__actions button, .arco-btn-status-danger').first();
if (await deleteBtn.count() > 0) {
  await deleteBtn.click();
  await page.waitForTimeout(800);

  const popconfirmInfo = await page.evaluate(() => {
    const pc = document.querySelector('.arco-popconfirm, .arco-popover');
    if (!pc) return { popconfirmFound: false };
    const cs = getComputedStyle(pc);
    return {
      popconfirmFound: true,
      borderRadius: cs.borderRadius,
      bg: cs.backgroundColor,
      padding: cs.padding,
      boxShadow: cs.boxShadow,
    };
  });

  console.log(JSON.stringify(popconfirmInfo, null, 2));
  await page.screenshot({ path: 'detail-popconfirm.png', fullPage: true });
  await page.keyboard.press('Escape');
} else {
  console.log('No delete button found');
}

await browser.close();

// Summary
console.log('\n\n========================================');
console.log('DETAIL CHECK SUMMARY');
console.log('========================================');
for (const i of allIssues) {
  const flags = [];
  if (i.overlaps > 0) flags.push(`${i.overlaps} OVERLAPS`);
  if (!i.filterVisible) flags.push('FILTER_HIDDEN');
  if (!i.batchBarVisible) flags.push('BATCHBAR_HIDDEN');
  console.log(`${i.page}: ${i.panelBorderCount} panels, ${i.cardBorderVariations} border-variations${flags.length ? ' ⚠ ' + flags.join(', ') : ''}`);
}
