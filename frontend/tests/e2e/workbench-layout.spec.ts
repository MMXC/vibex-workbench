/**
 * E2E tests for WorkbenchLayout — viewport-driven layout assertions
 *
 * Complements workbench-shell.spec.ts with additional viewport sizes
 * and cross-cutting layout properties (sidebar widths, canvas dimensions,
 * composer positioning, z-index, overflow, etc.)
 *
 * Breakpoints under test:
 *  - 1920px  (large desktop)
 *  - 1440px  (desktop, default 3-column)
 *  - 1280px  (desktop-narrow — within tablet range)
 *  - 1024px  (tablet landscape)
 *  - 900px   (tablet portrait — right panel hidden)
 *  - 768px   (small tablet boundary — right panel hidden)
 *  - 767px   (mobile trigger)
 *  - 600px   (large mobile)
 *  - 375px   (mobile)
 *
 * Note: getComputedStyle() resolves `1fr` to pixel values (viewport -
 * fixed columns). Assertions use pixel ranges with ±1px tolerance.
 */
import { test, expect, type Page } from '@playwright/test';

async function gotoWorkbench(page: Page) {
  await page.goto('/workbench', { waitUntil: 'networkidle' });
}

async function setViewport(page: Page, width: number, height = 900) {
  await page.setViewportSize({ width, height });
  await page.waitForTimeout(100);
}

/** Returns an array of column widths as CSS strings */
async function getComputedColumns(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const shell = document.querySelector('.shell') as HTMLElement;
    if (!shell) return [];
    return getComputedStyle(shell).gridTemplateColumns.split(/\s+/).filter(Boolean);
  });
}

/** Returns pixel widths for each grid column */
async function getColumnPx(page: Page): Promise<number[]> {
  return page.evaluate(() => {
    const shell = document.querySelector('.shell') as HTMLElement;
    if (!shell) return [];
    const cols = getComputedStyle(shell).gridTemplateColumns.split(/\s+/).filter(Boolean);
    return cols.map(c => {
      const n = parseFloat(c);
      return isNaN(n) ? -1 : n;
    });
  });
}

// ── Viewport: 1920px (large desktop) ────────────────────────

test('1920px: three columns — left≈280px, right≈320px', async ({ page }) => {
  await setViewport(page, 1920);
  await gotoWorkbench(page);

  const cols = await getColumnPx(page);
  expect(cols).toHaveLength(3);
  // Fixed columns match the desktop base CSS values
  expect(Math.abs(cols[0] - 280)).toBeLessThanOrEqual(1);
  expect(Math.abs(cols[2] - 320)).toBeLessThanOrEqual(1);
  // The 1fr column is content-sized
  expect(cols[1]).toBeGreaterThan(0);
});

test('1920px: all panels + composer visible', async ({ page }) => {
  await setViewport(page, 1920);
  await gotoWorkbench(page);

  await expect(page.locator('.sidebar-left')).toBeVisible();
  await expect(page.locator('.main-canvas')).toBeVisible();
  await expect(page.locator('.sidebar-right')).toBeVisible();
  await expect(page.locator('.composer-bar')).toBeVisible();
});

// ── Viewport: 1440px (desktop) ──────────────────────────────

test('1440px: three columns 280px 1fr 320px', async ({ page }) => {
  await setViewport(page, 1440);
  await gotoWorkbench(page);

  const cols = await getColumnPx(page);
  expect(cols).toHaveLength(3);
  // Fixed columns match the desktop base CSS values
  expect(Math.abs(cols[0] - 280)).toBeLessThanOrEqual(1);
  expect(Math.abs(cols[2] - 320)).toBeLessThanOrEqual(1);
  // The 1fr column is content-sized (positive value)
  expect(cols[1]).toBeGreaterThan(0);
});

test('1440px: sidebar-left has non-zero width', async ({ page }) => {
  await setViewport(page, 1440);
  await gotoWorkbench(page);

  const width = await page.evaluate(() => {
    const el = document.querySelector('.sidebar-left') as HTMLElement;
    return el ? el.getBoundingClientRect().width : null;
  });
  expect(width).toBeGreaterThan(0);
});

// ── Viewport: 1280px (desktop-narrow — within tablet range) ─

test('1280px: layout uses ~240px ~1fr ~280px (tablet breakpoint)', async ({ page }) => {
  await setViewport(page, 1280);
  await gotoWorkbench(page);

  const cols = await getColumnPx(page);
  expect(cols).toHaveLength(3);
  // Fixed columns match the CSS values defined in the 1024-1439px media query
  expect(Math.abs(cols[0] - 240)).toBeLessThanOrEqual(1);
  expect(Math.abs(cols[2] - 280)).toBeLessThanOrEqual(1);
  // The 1fr column is content-sized (overflow:hidden causes intrinsic sizing)
  expect(cols[1]).toBeGreaterThan(0);
});

test('1280px: sidebar-right is visible', async ({ page }) => {
  await setViewport(page, 1280);
  await gotoWorkbench(page);

  await expect(page.locator('.sidebar-right')).toBeVisible();
});

// ── Viewport: 1024px (tablet landscape) ────────────────────

test('1024px: layout adapts with right panel visible', async ({ page }) => {
  await setViewport(page, 1024);
  await gotoWorkbench(page);

  await expect(page.locator('.sidebar-right')).toBeVisible();
  const cols = await getColumnPx(page);
  expect(cols).toHaveLength(3);
  expect(Math.abs(cols[0] - 240)).toBeLessThanOrEqual(1);
  expect(Math.abs(cols[2] - 280)).toBeLessThanOrEqual(1);
});

// ── Viewport: 900px (tablet portrait — 768-1023 range) ─

test('900px: right panel hidden, left column still visible', async ({ page }) => {
  await setViewport(page, 900);
  await gotoWorkbench(page);

  await expect(page.locator('.sidebar-left')).toBeVisible();
  const rightVisible = await page.locator('.sidebar-right').isVisible();
  expect(rightVisible).toBe(false);
});

test('900px: composer-bar visible and not fixed', async ({ page }) => {
  await setViewport(page, 900);
  await gotoWorkbench(page);

  await expect(page.locator('.composer-bar')).toBeVisible();
  const position = await page.evaluate(() => {
    const el = document.querySelector('.composer-bar') as HTMLElement;
    return el ? getComputedStyle(el).position : 'unknown';
  });
  expect(position).not.toBe('fixed');
});

// ── Viewport: 768px (boundary — right panel exactly hidden) ─

test('768px: right panel is hidden at max-width boundary', async ({ page }) => {
  await setViewport(page, 768);
  await gotoWorkbench(page);

  const rightVisible = await page.locator('.sidebar-right').isVisible();
  expect(rightVisible).toBe(false);
});

test('768px: left sidebar still has non-zero width', async ({ page }) => {
  await setViewport(page, 768);
  await gotoWorkbench(page);

  const width = await page.evaluate(() => {
    const el = document.querySelector('.sidebar-left') as HTMLElement;
    return el ? el.getBoundingClientRect().width : null;
  });
  expect(width).toBeGreaterThan(0);
});

// ── Viewport: 767px (below 768px — mobile layout triggers) ─

test('767px: both sidebars hidden', async ({ page }) => {
  await setViewport(page, 767);
  await gotoWorkbench(page);

  await expect(page.locator('.sidebar-left')).not.toBeVisible();
  await expect(page.locator('.sidebar-right')).not.toBeVisible();
});

test('767px: main-canvas is visible', async ({ page }) => {
  await setViewport(page, 767);
  await gotoWorkbench(page);

  await expect(page.locator('.main-canvas')).toBeVisible();
});

test('767px: composer-bar is fixed at bottom', async ({ page }) => {
  await setViewport(page, 767);
  await gotoWorkbench(page);

  const position = await page.evaluate(() => {
    const el = document.querySelector('.composer-bar') as HTMLElement;
    return el ? getComputedStyle(el).position : null;
  });
  expect(position).toBe('fixed');
});

// ── Viewport: 600px (large mobile) ──────────────────────────

test('600px: composer-bar is fixed at bottom', async ({ page }) => {
  await setViewport(page, 600);
  await gotoWorkbench(page);

  const position = await page.evaluate(() => {
    const el = document.querySelector('.composer-bar') as HTMLElement;
    return el ? getComputedStyle(el).position : null;
  });
  expect(position).toBe('fixed');
});

test('600px: shell height equals viewport height (no page scroll)', async ({ page }) => {
  await setViewport(page, 600, 800);
  await gotoWorkbench(page);

  const height = await page.evaluate(() => {
    const shell = document.querySelector('.shell') as HTMLElement;
    return shell ? getComputedStyle(shell).height : null;
  });
  expect(height).toBe('800px');
});

// ── Viewport: 375px (mobile) ────────────────────────────────

test('375px: sidebar-left has zero computed width', async ({ page }) => {
  await setViewport(page, 375);
  await gotoWorkbench(page);

  const width = await page.evaluate(() => {
    const el = document.querySelector('.sidebar-left') as HTMLElement;
    return el ? el.getBoundingClientRect().width : null;
  });
  expect(width).toBe(0);
});

test('375px: shell overflow is hidden (no scrollable overflow)', async ({ page }) => {
  await setViewport(page, 375);
  await gotoWorkbench(page);

  const overflow = await page.evaluate(() => {
    const shell = document.querySelector('.shell') as HTMLElement;
    return shell ? getComputedStyle(shell).overflow : null;
  });
  expect(overflow).toBe('hidden');
});

// ── Cross-viewport consistency ──────────────────────────────

test('main-canvas is always visible across all tested viewports', async ({ page }) => {
  const viewports = [1920, 1440, 1280, 1024, 900, 768, 767, 600, 375];
  for (const w of viewports) {
    await setViewport(page, w);
    await gotoWorkbench(page);
    await expect(page.locator('.main-canvas')).toBeVisible();
  }
});

test('composer-bar is always visible across all tested viewports', async ({ page }) => {
  const viewports = [1920, 1440, 1280, 1024, 900, 768, 767, 600, 375];
  for (const w of viewports) {
    await setViewport(page, w);
    await gotoWorkbench(page);
    await expect(page.locator('.composer-bar')).toBeVisible();
  }
});

test('shell height equals 100vh at desktop, tablet, and mobile viewports', async ({ page }) => {
  for (const w of [1440, 1024, 375]) {
    await setViewport(page, w);
    await gotoWorkbench(page);

    const height = await page.evaluate(() => {
      const shell = document.querySelector('.shell') as HTMLElement;
      return shell ? getComputedStyle(shell).height : null;
    });
    expect(height).toBe('900px');
  }
});

// ── Grid area assignments ───────────────────────────────────

test('sidebar-left has grid-area: L', async ({ page }) => {
  await setViewport(page, 1440);
  await gotoWorkbench(page);

  const area = await page.evaluate(() => {
    const el = document.querySelector('.sidebar-left');
    return el ? getComputedStyle(el).gridArea : null;
  });
  expect(area).toBe('L');
});

test('sidebar-right has grid-area: R', async ({ page }) => {
  await setViewport(page, 1440);
  await gotoWorkbench(page);

  const area = await page.evaluate(() => {
    const el = document.querySelector('.sidebar-right');
    return el ? getComputedStyle(el).gridArea : null;
  });
  expect(area).toBe('R');
});

test('composer-bar has grid-area: B', async ({ page }) => {
  await setViewport(page, 1440);
  await gotoWorkbench(page);

  const area = await page.evaluate(() => {
    const el = document.querySelector('.composer-bar');
    return el ? getComputedStyle(el).gridArea : null;
  });
  expect(area).toBe('B');
});

// ── Responsive grid collapse verification ───────────────────

test('at 768px and below, right column is 0px (display:none via CSS)', async ({ page }) => {
  for (const w of [768, 767, 600, 375]) {
    await setViewport(page, w);
    await gotoWorkbench(page);

    const cols = await getColumnPx(page);
    // Right column should be ~0px (CSS sets grid-template-columns: ... 0px)
    expect(cols[2]).toBe(0);
  }
});

test('at 900px (tablet portrait), grid columns are 200px ~1fr 0px', async ({ page }) => {
  await setViewport(page, 900);
  await gotoWorkbench(page);

  const cols = await getColumnPx(page);
  expect(cols).toHaveLength(3);
  // Left: 200px (narrowed from 240px), center: 900-200-0=700px, right: 0px
  expect(Math.abs(cols[0] - 200)).toBeLessThanOrEqual(1);
  expect(Math.abs(cols[2])).toBeLessThanOrEqual(1);
});
