import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

test('administrator password rotation returns a one-time random secret', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('/#settings');
  await expect(page.getByRole('heading', { name: '访问安全' })).toBeVisible();
  await expect(page.getByText('最近轮换于')).toBeVisible();

  await page.getByRole('button', { name: '随机轮换' }).click();
  const confirm = page.getByRole('dialog', { name: '轮换管理员口令？' });
  await expect(confirm).toContainText('不接受自定义输入');
  await expect(confirm).toContainText('所有现有管理员会话都会被撤销');
  await confirm.getByRole('button', { name: '生成新口令' }).click();

  const result = page.getByRole('dialog', { name: '新管理员口令已生成' });
  await expect(result).toBeVisible();
  await expect(result).toContainText('这是唯一一次显示新口令');
  const accessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  expect(accessibility.violations.filter((violation) => ['serious', 'critical'].includes(violation.impact))).toEqual([]);
  const password = await result.locator('#rotated-admin-password').textContent();
  expect(password).toMatch(/^[A-HJ-NP-Za-km-z2-9]{32}$/);
  expect(password).not.toContain('Boxd-E2E');

  await result.getByRole('button', { name: '复制新管理员口令' }).click();
  await expect(page.locator('.toast').filter({ hasText: '新口令已复制' })).toBeVisible();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(password);

  await result.getByRole('button', { name: '我已安全保存' }).click();
  await expect(result).toBeHidden();
  const retained = await page.evaluate((secret) => {
    const storage = { ...localStorage, ...sessionStorage };
    return document.body.innerHTML.includes(secret) || Object.values(storage).some((value) => String(value).includes(secret));
  }, password);
  expect(retained).toBe(false);

  await expect(page.getByText('会话有效', { exact: true })).toBeVisible();
});
