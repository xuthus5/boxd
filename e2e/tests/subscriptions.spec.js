import { expect, test } from '@playwright/test';

async function deleteSubscriptionByName(request, name) {
  const response = await request.get('/api/v1/subscriptions');
  const subscriptions = await response.json();
  const subscription = subscriptions.find((item) => item.name === name);
  if (subscription) await request.delete(`/api/v1/subscriptions/${subscription.id}`);
}

test('subscription can be created and edited through the real API', async ({ page, request }) => {
  const name = `E2E UI ${Date.now()}`;
  const renamed = `${name} Edited`;
  await page.goto('/#subscriptions');
  await expect(page.getByRole('heading', { name: '订阅管理' })).toBeVisible();

  await page.getByRole('button', { name: '添加订阅' }).click();
  await page.locator('#form-name').fill(name);
  await page.locator('#form-url').fill('https://ui.e2e.invalid/subscription?token=not-a-real-token');
  await page.locator('#form-interval').selectOption('3600');
  await page.locator('.modal-root [data-action="modal-submit"]').click();

  const card = page.locator('.sub-card').filter({ hasText: name });
  await expect(card).toBeVisible();
  await card.getByRole('button', { name: `编辑订阅 ${name}` }).click();
  await page.locator('#form-name').fill(renamed);
  await page.locator('.modal-root [data-action="modal-submit"]').click();
  await expect(page.locator('.sub-card').filter({ hasText: renamed })).toBeVisible();

  await deleteSubscriptionByName(request, renamed);
});

test('single node links can be imported from the subscription page', async ({ page, request }) => {
  const name = `E2E Import ${Date.now()}`;
  await page.goto('/#subscriptions');
  await page.getByRole('button', { name: '导入单节点' }).click();
  await page.locator('#node-import-links').fill(
    `vless://9b1c4d6e-0000-0000-0000-000000000202@import.e2e.invalid:443?security=tls&sni=import.e2e.invalid#${encodeURIComponent(name)}`,
  );
  await page.locator('.modal-root [data-action="do-import-node"]').click();
  await expect(page.locator('.modal-root')).toBeHidden();

  await page.goto('/#egress');
  await expect(page.getByText(name, { exact: true })).toBeVisible();

  const nodes = await (await request.get('/api/v1/nodes')).json();
  const node = nodes.find((item) => item.name === name);
  expect(node).toBeTruthy();
  await request.delete(`/api/v1/nodes/${node.id}`);
});
