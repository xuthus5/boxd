import { expect, test } from '@playwright/test';

test('kernel lifecycle reports real running state', async ({ page, request }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop-chromium', 'lifecycle runs once on Chromium');
  test.setTimeout(120_000);

  await request.post('/api/v1/kernel/stop');
  const snapshot = await (await request.get('/api/v1/snapshot')).json();
  const mixed = snapshot.inbounds.find((inbound) => inbound.kind === 'mixed' && inbound.enabled);
  expect(mixed).toBeTruthy();

  const patch = await request.patch(`/api/v1/inbounds/${mixed.id}`, {
    data: { listen: { host: '127.0.0.1', port: 18080 } },
  });
  expect(patch.ok()).toBeTruthy();

  try {
    const start = await request.post('/api/v1/kernel/start');
    expect(start.ok()).toBeTruthy();
    const status = await (await request.get('/api/v1/kernel/status')).json();
    expect(status.state).toBe('running');

    await page.goto('/#overview');
    await expect(page.getByRole('heading', { name: '控制平面总览' })).toBeVisible();
    await expect(page.locator('.overview-runtime-priority')).toContainText('运行中');
  } finally {
    const stop = await request.post('/api/v1/kernel/stop');
    expect(stop.ok()).toBeTruthy();
  }
});
