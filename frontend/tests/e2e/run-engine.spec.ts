/**
 * E2E tests for RunEngine — RunStatusBar UI in Composer.svelte
 *
 * Coverage:
 * 1. Composer renders with textarea and submit button
 * 2. Status bar NOT visible initially (showStatus=false)
 * 3. UI structure: .run-status-bar has correct CSS class slots
 * 4. .run-status-bar.running/.completed/.failed CSS classes are defined
 * 5. RunStatusBar animation/transition keyframes are present
 * 6. Mode tabs (text/image/file/url) are visible
 *
 * Note: Store-driven status changes (running/completed/failed) require SSE events
 * from the backend, which are not available in preview mode. These tests verify
 * the component structure and CSS so that when SSE delivers events, the UI works.
 */
import { test, expect } from '@playwright/test';

const WORKBENCH_URL = '/workbench';

async function gotoWorkbench(page: any) {
  await page.goto(WORKBENCH_URL, { waitUntil: 'networkidle' });
  // Allow Svelte hydration
  await page.waitForTimeout(500);
}

// ── Test 1: Composer renders with textarea and submit button ─
test('Composer renders with textarea and submit button', async ({ page }) => {
  await gotoWorkbench(page);

  const textarea = page.locator('.composer textarea');
  await expect(textarea).toBeVisible();

  const submitBtn = page.locator('.composer .submit-btn');
  await expect(submitBtn).toBeVisible();
  await expect(submitBtn).toContainText('发送');
});

// ── Test 2: Status bar NOT visible initially ─────────────────
// The run-status-bar should be hidden when no run is active
test('run-status-bar NOT visible initially (showStatus=false)', async ({ page }) => {
  await gotoWorkbench(page);

  // {#if showStatus} controls visibility; initially false → 0 elements
  const statusBar = page.locator('.run-status-bar');
  await expect(statusBar).toHaveCount(0);
});

// ── Test 3: Mode tabs visible ────────────────────────────────
test('mode tabs (text/image/file/url) are visible', async ({ page }) => {
  await gotoWorkbench(page);

  const tabs = page.locator('.composer .mode-tabs button');
  await expect(tabs).toHaveCount(4);
  await expect(tabs.nth(0)).toContainText('文本');
  await expect(tabs.nth(1)).toContainText('图片');
  await expect(tabs.nth(2)).toContainText('文件');
  await expect(tabs.nth(3)).toContainText('URL');
});

// ── Test 4: Tool count hint visible ─────────────────────────
test('hint shows tool count', async ({ page }) => {
  await gotoWorkbench(page);

  const hint = page.locator('.composer .hint');
  await expect(hint).toBeVisible();
  // Initial tool count is 0
  await expect(hint).toContainText('0 tools');
});

// ── Test 5: CSS: run-status-bar class slots are defined ───────
// We verify that the compiled CSS contains the expected class definitions
// by checking that CSS rules matching the class selectors exist in the page.
test('CSS: .run-status-bar.running/.completed/.failed styles are defined', async ({ page }) => {
  await gotoWorkbench(page);

  const cssChecks = await page.evaluate(() => {
    const results: Record<string, boolean> = {};
    const sheets = Array.from(document.styleSheets);

    for (const sheet of sheets) {
      try {
        const rules = Array.from(sheet.cssRules as CSSRuleList);
        for (const rule of rules) {
          if (rule instanceof CSSStyleRule) {
            const selector = rule.selectorText ?? '';
            if (selector.includes('.run-status-bar.running')) results['running'] = true;
            if (selector.includes('.run-status-bar.completed')) results['completed'] = true;
            if (selector.includes('.run-status-bar.failed')) results['failed'] = true;
          }
        }
      } catch (_) { /* cross-origin sheet — skip */ }
    }
    return results;
  });

  expect(cssChecks['running']).toBe(true);
  expect(cssChecks['completed']).toBe(true);
  expect(cssChecks['failed']).toBe(true);
});

// ── Test 6: CSS: slideIn keyframe animation is defined ───────
test('CSS: slideIn keyframe animation is defined', async ({ page }) => {
  await gotoWorkbench(page);

  const hasSlideIn = await page.evaluate(() => {
    const sheets = Array.from(document.styleSheets);
    for (const sheet of sheets) {
      try {
        const rules = Array.from(sheet.cssRules as CSSRuleList);
        for (const rule of rules) {
          if (rule instanceof CSSKeyframesRule) {
            const cssText = rule.cssText;
            if (rule.name === 'slideIn' || cssText.includes('slideIn')) return true;
          }
          // Also check for @keyframes in CSS text (in case keyframe rule isn't exposed)
          if (rule instanceof CSSStyleRule && rule.cssText.includes('@keyframes slideIn')) return true;
        }
      } catch (_) { /* cross-origin or inaccessible sheet — skip */ }
    }
    // Also search inline stylesheets (Svelte inlines them as <style> tags)
    const inline = Array.from(document.querySelectorAll('style'));
    for (const el of inline) {
      if (el.textContent?.includes('@keyframes slideIn')) return true;
    }
    return false;
  });

  expect(hasSlideIn).toBe(true);
});

// ── Test 7: CSS: spin keyframe is defined for running icon ───
test('CSS: spin keyframe is defined for running icon animation', async ({ page }) => {
  await gotoWorkbench(page);

  const hasSpin = await page.evaluate(() => {
    const sheets = Array.from(document.styleSheets);
    for (const sheet of sheets) {
      try {
        const rules = Array.from(sheet.cssRules as CSSRuleList);
        for (const rule of rules) {
          if (rule instanceof CSSKeyframesRule) {
            const cssText = rule.cssText;
            if (rule.name === 'spin' || cssText.includes('spin')) return true;
          }
          if (rule instanceof CSSStyleRule && rule.cssText.includes('@keyframes spin')) return true;
        }
      } catch (_) { /* cross-origin or inaccessible sheet — skip */ }
    }
    const inline = Array.from(document.querySelectorAll('style'));
    for (const el of inline) {
      if (el.textContent?.includes('@keyframes spin')) return true;
    }
    return false;
  });

  expect(hasSpin).toBe(true);
});

// ── Test 8: .composer element has correct structure ──────────
test('.composer has textarea, mode-tabs, and actions', async ({ page }) => {
  await gotoWorkbench(page);

  const composer = page.locator('.composer');
  await expect(composer).toBeVisible();

  // All three key children present
  await expect(composer.locator('textarea')).toBeVisible();
  await expect(composer.locator('.mode-tabs')).toBeVisible();
  await expect(composer.locator('.actions')).toBeVisible();
});
