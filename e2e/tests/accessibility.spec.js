import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const views = ['overview', 'inbounds', 'egress', 'routes', 'subscriptions', 'rulesets', 'observability', 'logs', 'settings'];

function seriousViolations(results) {
  return results.violations.filter((violation) => ['serious', 'critical'].includes(violation.impact));
}

async function waitForPageTransition(page) {
  await page.locator('#app .page').evaluate(async (element) => {
    await Promise.all(element.getAnimations().map((animation) => animation.finished.catch(() => {})));
  });
}

test('authenticated views have no serious accessibility violations', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop-chromium', 'axe scan runs once on Chromium');

  for (const view of views) {
    await page.goto(`/#${view}`);
    await expect(page.locator('#app .page')).toBeVisible();
    await waitForPageTransition(page);
    const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
    expect(seriousViolations(results), `${view}: ${JSON.stringify(seriousViolations(results), null, 2)}`).toEqual([]);
  }
});

test('login view has no serious accessibility violations', async ({ browser }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop-chromium', 'axe scan runs once on Chromium');
  const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
  const page = await context.newPage();
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '欢迎回来' })).toBeVisible();
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  expect(seriousViolations(results), JSON.stringify(seriousViolations(results), null, 2)).toEqual([]);
  await context.close();
});
