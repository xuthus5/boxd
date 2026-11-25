import { expect, test } from '@playwright/test';
import { E2E_PASSWORD } from '../support/env.js';

test.use({ storageState: { cookies: [], origins: [] } });

test('administrator can authenticate and reach the console', async ({ page }) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: '欢迎回来' })).toBeVisible();
  await page.getByLabel('管理员口令').fill('incorrect-password');
  await page.getByRole('button', { name: '登录控制面' }).click();
  await expect(page.getByRole('alert')).toBeVisible();

  await page.getByLabel('管理员口令').fill(E2E_PASSWORD);
  await page.getByRole('button', { name: '登录控制面' }).click();

  await expect(page.locator('.app-shell')).toBeVisible();
  await expect(page.locator('#current-view-label')).toHaveText('总览');
  await expect(page.getByRole('heading', { name: '控制平面总览' })).toBeVisible();
});

test('unauthenticated API requests are rejected', async ({ request }) => {
  const response = await request.get('/api/v1/snapshot');
  expect(response.status()).toBe(401);
  const body = await response.json();
  expect(body.error.code).toBe('kernel_unauthorized');
});
