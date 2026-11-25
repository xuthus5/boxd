import { expect, test } from '@playwright/test';

const views = [
  ['overview', '总览'],
  ['inbounds', '入口'],
  ['egress', '出口编排'],
  ['routes', '路由'],
  ['subscriptions', '订阅'],
  ['rulesets', '规则集'],
  ['observability', '观测'],
  ['logs', '日志'],
  ['settings', '设置'],
];

test('all console views render without page errors or overflow', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));

  for (const [view, label] of views) {
    await page.goto(`/#${view}`);
    await expect(page.locator('.app-shell')).toBeVisible();
    await expect(page.locator('#current-view-label')).toHaveText(label);
    await expect(page.locator('#app .page')).toBeVisible();
    const overflow = await page.evaluate(() => ({
      document: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      body: document.body.scrollWidth - document.body.clientWidth,
    }));
    expect(overflow.document, `${view} document overflow`).toBeLessThanOrEqual(1);
    expect(overflow.body, `${view} body overflow`).toBeLessThanOrEqual(1);
  }

  expect(pageErrors).toEqual([]);
});

test('theme choice applies and survives reload', async ({ page }) => {
  await page.goto('/#overview');
  await expect(page.locator('.app-shell')).toBeVisible();
  const root = page.locator('html');
  const initial = await root.getAttribute('data-theme');
  await page.locator('#theme-toggle').click();
  const changed = await root.getAttribute('data-theme');
  expect(changed).not.toBe(initial);

  await page.reload();
  await expect(page.locator('.app-shell')).toBeVisible();
  await expect(root).toHaveAttribute('data-theme', changed);
});

test('administrator avatar renders a shield icon instead of initials', async ({ page }) => {
  await page.goto('/#overview');
  await expect(page.locator('.app-shell')).toBeVisible();

  const avatar = page.getByRole('button', { name: '管理员账户' });
  await expect(avatar).toBeVisible();
  await expect(avatar.locator('svg.icon')).toHaveCount(1);
  await expect(avatar).not.toContainText('LX');
});

test('long log messages scroll horizontally and copy the full row', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('/#logs');
  await expect(page.getByRole('log', { name: '应用与内核日志' })).toBeVisible();

  const marker = `e2e-log-layout-${Date.now()}`;
  const requestPath = `/api/v1/nodes/${marker}-${'x'.repeat(320)}`;
  await page.evaluate(async (path) => {
    await fetch(path, { headers: { Accept: 'application/json' } });
  }, requestPath);

  const logRow = page.locator('#logs-console .log-line').filter({ hasText: marker });
  await expect(logRow).toHaveCount(1);
  const layout = await logRow.evaluate((row) => {
    const message = row.querySelector('.log-message');
    const consoleElement = row.closest('#logs-console');
    const messageStyle = getComputedStyle(message);
    const lineStyle = getComputedStyle(row);
    consoleElement.scrollLeft = consoleElement.scrollWidth;
    return {
      rowHeight: row.getBoundingClientRect().height,
      messageWhiteSpace: messageStyle.whiteSpace,
      messageOverflow: messageStyle.overflow,
      messageTextOverflow: messageStyle.textOverflow,
      lineWidth: row.getBoundingClientRect().width,
      lineCursor: lineStyle.cursor,
      role: row.getAttribute('role'),
      tabIndex: row.tabIndex,
      consoleClientWidth: consoleElement.clientWidth,
      consoleScrollWidth: consoleElement.scrollWidth,
      consoleHorizontalOverflow: consoleElement.scrollWidth - consoleElement.clientWidth,
      consoleMaxScrollLeft: consoleElement.scrollLeft,
      documentOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });

  expect(layout.rowHeight, 'long log row height').toBeLessThanOrEqual(28);
  expect(layout.messageWhiteSpace).toBe('nowrap');
  expect(layout.messageOverflow).toBe('visible');
  expect(layout.messageTextOverflow).toBe('clip');
  expect(layout.lineWidth).toBeGreaterThan(layout.consoleClientWidth);
  expect(layout.lineCursor).toBe('copy');
  expect(layout.role).toBe('button');
  expect(layout.tabIndex).toBe(0);
  expect(layout.consoleScrollWidth).toBeGreaterThan(layout.consoleClientWidth);
  expect(layout.consoleHorizontalOverflow, 'log console exposes horizontal scrolling').toBeGreaterThan(1);
  expect(layout.consoleMaxScrollLeft, 'log console can scroll to the end').toBeGreaterThan(1);
  expect(layout.documentOverflow).toBeLessThanOrEqual(1);

  await logRow.locator('.log-time').click();
  await expect(page.locator('.toast').filter({ hasText: '日志已复制' })).toBeVisible();
  const clipboard = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipboard).toContain(marker);
  expect(clipboard).toContain('[APP]');

  await page.evaluate(async () => {
    window.__boxdNativeClipboard = navigator.clipboard;
    await window.__boxdNativeClipboard.writeText('clipboard-fallback-sentinel');
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
  });
  await logRow.locator('.log-time').click();
  await expect.poll(() => page.evaluate(() => window.__boxdNativeClipboard.readText())).toContain(marker);
});

test('kernel lifecycle controls are only available on the overview page', async ({ page }) => {
  await page.goto('/#overview');
  await expect(page.locator('.app-shell')).toBeVisible();
  await expect(page.locator('.overview-runtime-actions [data-action="start-kernel"], .overview-runtime-actions [data-action="stop-kernel"]')).toHaveCount(1);

  await page.goto('/#settings');
  await expect(page.getByRole('heading', { name: '关于控制面' })).toBeVisible();
  await expect(page.locator('article.settings-card h2')).toHaveText(['访问安全', 'DNS 配置', '关于控制面', '外观']);
  await expect(page.getByRole('region', { name: '内核基本信息' })).toBeVisible();
  await expect(page.getByText('生命周期由总览页统一管理，此处仅展示基本状态')).toBeVisible();
  const aboutCard = page.locator('article.settings-card').filter({ has: page.getByRole('heading', { name: '关于控制面' }) });
  await expect(aboutCard.locator('.card-head .tag')).toContainText('事件流');
  await expect(page.locator('[data-action="start-kernel"], [data-action="stop-kernel"], [data-action="restart-kernel"]')).toHaveCount(0);
});

test('mobile navigation opens and switches views', async ({ page }, testInfo) => {
  test.skip(!testInfo.project.name.startsWith('mobile-'), 'mobile viewport only');
  await page.goto('/#overview');
  await expect(page.locator('.app-shell')).toBeVisible();

  await page.getByRole('button', { name: '打开导航' }).click();
  await expect(page.locator('body')).toHaveClass(/sidebar-open/);
  await page.locator('.sidebar button[data-view="subscriptions"]').click();
  await expect(page.locator('#current-view-label')).toHaveText('订阅');
  await expect(page.locator('body')).not.toHaveClass(/sidebar-open/);
});
