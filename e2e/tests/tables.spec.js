import { expect, test } from '@playwright/test';

const tableSpecs = [
  { view: 'overview', className: 'overview-connections-table', columns: 4, heading: '目标' },
  { view: 'nodes', className: 'nodes-table', columns: 9, heading: '节点' },
  { view: 'routes', className: 'routes-table', columns: 8, heading: '规则' },
  { view: 'rulesets', className: 'ruleset-table', columns: 8, heading: '规则集' },
  { view: 'connections', className: 'connections-table', columns: 8, heading: '目标' },
];

const longValue = `https://${'very-long-table-value-'.repeat(18)}example.invalid/path?token=${'z'.repeat(160)}`;

function fixtureTable({ className, columns, heading }) {
  const header = Array.from({ length: columns }, (_, index) => `<th>${index === 0 ? heading : `列 ${index + 1}`}</th>`).join('');
  const dataCells = Array.from({ length: columns - 1 }, (_, index) => {
    if (index === 0) {
      return '<td><div class="cell-primary"><strong></strong><span></span></div></td>';
    }
    return `<td>${index === 2 ? `<span class="tag is-solid">${longValue}</span>` : ''}</td>`;
  }).join('');
  const actionCell = '<td><div class="cell-actions"><button class="btn btn-sm btn-icon-only" type="button">查看</button><button class="btn btn-sm btn-danger" type="button">删除</button></div></td>';
  return `
    <article class="card table-fixture" data-fixture="${className}">
      <div class="table-wrap">
        <table class="data-table ${className}">
          <thead><tr>${header}</tr></thead>
          <tbody>
            <tr>${dataCells}${actionCell}</tr>
            <tr>${dataCells}${actionCell}</tr>
          </tbody>
        </table>
      </div>
    </article>
  `;
}

test('all table layouts truncate long values without growing row height', async ({ page }) => {
  await page.goto('/#overview');
  await expect(page.locator('.app-shell')).toBeVisible();

  for (const spec of tableSpecs) {
    const result = await page.evaluate(({ spec, longValue, html }) => {
      const pageRoot = document.querySelector('#app .page');
      pageRoot.insertAdjacentHTML('beforeend', html);
      const fixture = pageRoot.querySelector(`[data-fixture="${spec.className}"]`);
      const table = fixture.querySelector('table');
      const wrap = fixture.querySelector('.table-wrap');
      const rows = [...table.tBodies[0].rows];
      const firstRow = rows[0];
      const firstCell = firstRow.cells[0];
      const primary = firstCell.querySelector('.cell-primary');
      const textTargets = primary ? [...primary.children] : [...firstCell.children];

      for (const target of textTargets) target.textContent = longValue;
      const secondRow = rows[1];
      for (const target of secondRow.cells[0].querySelectorAll('.cell-primary > *')) target.textContent = longValue;

      const actionCell = firstRow.cells[firstRow.cells.length - 1];
      const actionButtons = [...actionCell.querySelectorAll('button')].map((button) => button.getBoundingClientRect());
      const rowHeights = rows.map((row) => row.getBoundingClientRect().height);
      const firstCellStyle = getComputedStyle(firstCell);
      const result = {
        tableLayout: getComputedStyle(table).tableLayout,
        wrapOverflowX: getComputedStyle(wrap).overflowX,
        maxRowHeight: Math.max(...rowHeights),
        cellWhiteSpace: firstCellStyle.whiteSpace,
        cellOverflow: firstCellStyle.overflow,
        cellTextOverflow: firstCellStyle.textOverflow,
        documentOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        actionRowCount: new Set(actionButtons.map((rect) => Math.round(rect.top))).size,
      };
      fixture.remove();
      return result;
    }, { spec, longValue, html: fixtureTable(spec) });

    expect(result.tableLayout, `${spec.className} table layout`).toBe('fixed');
    expect(result.wrapOverflowX, `${spec.className} wrapper overflow`).toBe('auto');
    expect(result.maxRowHeight, `${spec.className} row height`).toBeLessThanOrEqual(60);
    expect(result.cellWhiteSpace, `${spec.className} cell white-space`).toBe('nowrap');
    expect(result.cellOverflow, `${spec.className} cell overflow`).toBe('hidden');
    expect(result.cellTextOverflow, `${spec.className} cell text overflow`).toBe('ellipsis');
    expect(result.actionRowCount, `${spec.className} actions should stay on one line`).toBe(1);
    expect(result.documentOverflow, `${spec.className} page overflow`).toBeLessThanOrEqual(1);
  }
});

test('real route rows expose full values while keeping the table compact', async ({ page, request }) => {
  const ruleName = `E2E Table ${Date.now()}`;
  const longDomain = `${'a'.repeat(120)}.${'b'.repeat(120)}.example.invalid`;
  const response = await request.post('/api/v1/routes/rules', {
    data: {
      name: ruleName,
      match: { domain_suffixes: [longDomain] },
      action: 'direct',
    },
  });
  expect(response.ok()).toBeTruthy();
  const rule = await response.json();

  try {
    await page.goto('/#routes');
    await expect(page.getByRole('heading', { name: '路由策略' })).toBeVisible();
    const row = page.locator('.routes-table tbody tr').filter({ hasText: ruleName });
    await expect(row).toHaveCount(1);
    await expect(row.locator('.cell-primary')).toHaveAttribute('title', new RegExp(ruleName));
    await expect(row.locator('.condition-text')).toHaveAttribute('title', `*.${longDomain}`);

    const layout = await row.evaluate((element) => {
      const cell = element.cells[2];
      const style = getComputedStyle(cell);
      return {
        height: element.getBoundingClientRect().height,
        whiteSpace: style.whiteSpace,
        overflow: style.overflow,
        textOverflow: style.textOverflow,
      };
    });
    expect(layout.height).toBeLessThanOrEqual(60);
    expect(layout.whiteSpace).toBe('nowrap');
    expect(layout.overflow).toBe('hidden');
    expect(layout.textOverflow).toBe('ellipsis');
  } finally {
    await request.delete(`/api/v1/routes/rules/${rule.id}`);
  }
});

test('table action columns fit their buttons without vertical wrapping', async ({ page, request }) => {
  const ruleName = `E2E Actions ${Date.now()}`;
  const response = await request.post('/api/v1/routes/rules', {
    data: {
      name: ruleName,
      match: { domain_suffixes: ['actions.example.invalid'] },
      action: 'direct',
    },
  });
  expect(response.ok()).toBeTruthy();
  const rule = await response.json();

  try {
    for (const [view, selector] of [['egress', '.nodes-table'], ['routes', '.routes-table'], ['rulesets', '.ruleset-table']]) {
      await page.goto(`/#${view}`);
      const table = page.locator(selector);
      await expect(table).toBeVisible();
      const rows = table.locator('tbody tr').filter({ hasNot: page.locator('.empty-state') });
      const layout = await rows.first().evaluate((row) => {
        const cell = row.cells[row.cells.length - 1];
        const actions = cell.querySelector('.cell-actions');
        const cellRect = cell.getBoundingClientRect();
        const cellStyle = getComputedStyle(cell);
        const actionRect = actions.getBoundingClientRect();
        const buttons = [...actions.querySelectorAll('button')].map((button) => button.getBoundingClientRect());
        return {
          clipped: actionRect.right > cellRect.right - parseFloat(cellStyle.paddingRight) + 1,
          overflow: actions.scrollWidth > actions.clientWidth + 1,
          buttonRowCount: new Set(buttons.map((rect) => Math.round(rect.top + (rect.height / 2)))).size,
        };
      });
      expect(layout.clipped, `${view} action column clips buttons`).toBe(false);
      expect(layout.overflow, `${view} action column overflows`).toBe(false);
      expect(layout.buttonRowCount, `${view} action buttons wrap`).toBe(1);
    }
  } finally {
    await request.delete(`/api/v1/routes/rules/${rule.id}`);
  }
});
