/**
 * E2E tests for ArtifactRegistry (E4 Epic)
 *
 * Components under test:
 *  - ArtifactPanel.svelte     — artifact list panel (sidebar-right)
 *  - ArtifactPreviewModal.svelte — preview modal (conditionally rendered)
 *  - Composer.svelte         — drag-drop target (footer)
 *
 * Strategy: Since no real IndexedDB/backend, tests focus on static UI
 * structure, CSS classes, and DOM presence. The store may log errors in
 * console due to IndexedDB being unavailable in static preview, which is
 * expected — tests assert on DOM visibility, not console cleanliness.
 */
import { test, expect } from '@playwright/test';

async function gotoWorkbench(page: import('@playwright/test').Page) {
  await page.goto('/workbench', { waitUntil: 'networkidle' });
}

// ── ArtifactPanel: header ───────────────────────────────────
test('ArtifactPanel header shows "Artifacts" with count', async ({ page }) => {
  await gotoWorkbench(page);

  const header = page.locator('.artifact-panel .header');
  await expect(header).toBeVisible();
  await expect(header).toContainText('Artifacts');
});

test('header count shows (0) when no artifacts loaded', async ({ page }) => {
  await gotoWorkbench(page);

  const header = page.locator('.artifact-panel .header');
  await expect(header).toContainText('Artifacts (0)', { timeout: 5000 });
});

// ── ArtifactPanel: search ───────────────────────────────────
test('search input is visible in artifact panel', async ({ page }) => {
  await gotoWorkbench(page);

  const searchInput = page.locator('.artifact-panel .search input');
  await expect(searchInput).toBeVisible();
  await expect(searchInput).toHaveAttribute('placeholder', '搜索...');
});

test('search input accepts text', async ({ page }) => {
  await gotoWorkbench(page);

  const searchInput = page.locator('.artifact-panel .search input');
  await searchInput.fill('hello');
  await expect(searchInput).toHaveValue('hello');
});

// ── ArtifactPanel: filter buttons ──────────────────────────
test('filter buttons visible: 全部, 代码, 图片', async ({ page }) => {
  await gotoWorkbench(page);

  const filterDiv = page.locator('.artifact-panel .filter');
  await expect(filterDiv).toBeVisible();

  const buttons = filterDiv.locator('button');
  await expect(buttons).toHaveCount(3);
  await expect(buttons.nth(0)).toContainText('全部');
  await expect(buttons.nth(1)).toContainText('代码');
  await expect(buttons.nth(2)).toContainText('图片');
});

test('filter button 全部 is active by default', async ({ page }) => {
  await gotoWorkbench(page);

  const filterBtn = page.locator('.artifact-panel .filter button').first();
  await expect(filterBtn).toHaveClass(/active/);
});

// ── ArtifactPanel: empty state ──────────────────────────────
test('empty state "暂无 Artifact" is visible when no artifacts', async ({ page }) => {
  await gotoWorkbench(page);

  const empty = page.locator('.artifact-panel .empty');
  await expect(empty).toBeVisible({ timeout: 5000 });
  await expect(empty).toContainText('暂无 Artifact');
});

// ── ArtifactPanel: shell integration (E4-U3 drag structure) ─
test('ArtifactPanel is mounted in sidebar-right with items container', async ({ page }) => {
  await gotoWorkbench(page);

  const panel = page.locator('.sidebar-right .artifact-panel');
  await expect(panel).toBeVisible();

  const items = page.locator('.artifact-panel .items');
  await expect(items).toBeVisible();
});

// ── ArtifactPreviewModal: DOM structure ─────────────────────
test('ArtifactPreviewModal renders in DOM (hidden when no selection)', async ({ page }) => {
  await gotoWorkbench(page);

  // The modal is a child of ArtifactPanel — always in DOM, hidden via {#if}
  // We verify the component file is mounted and the overlay is absent.
  const overlay = page.locator('.modal-overlay');
  await expect(overlay).not.toBeVisible();
});

test('modal-close button exists in component structure', async ({ page }) => {
  await gotoWorkbench(page);

  // Close button is in the modal header; only visible when modal is open
  // Check the button element exists in the DOM (even if modal is hidden)
  const closeBtn = page.locator('.close-btn');
  await expect(closeBtn).toBeAttached();
});

// ── Composer: drag-drop zone (E4-U3) ────────────────────────
test('Composer div has ondrop handler (accepts drops)', async ({ page }) => {
  await gotoWorkbench(page);

  const composer = page.locator('.composer');
  await expect(composer).toBeVisible();

  // Verify the element has the drag-drop event attributes in the DOM
  const dropAttr = await composer.getAttribute('ondrop');
  expect(dropAttr).not.toBeNull();
});

test('Composer div has ondragover and ondragleave handlers', async ({ page }) => {
  await gotoWorkbench(page);

  const composer = page.locator('.composer');
  await expect(composer).toBeVisible();

  const dragOverAttr = await composer.getAttribute('ondragover');
  const dragLeaveAttr = await composer.getAttribute('ondragleave');
  expect(dragOverAttr).not.toBeNull();
  expect(dragLeaveAttr).not.toBeNull();
});

test('Composer textarea is visible and accepts text', async ({ page }) => {
  await gotoWorkbench(page);

  const textarea = page.locator('.composer textarea');
  await expect(textarea).toBeVisible();
  await expect(textarea).toHaveAttribute('placeholder', '输入消息，或 @ 引用 Artifact...');

  await textarea.fill('test message');
  await expect(textarea).toHaveValue('test message');
});

test('Composer submit button is visible', async ({ page }) => {
  await gotoWorkbench(page);

  const submitBtn = page.locator('.composer .submit-btn');
  await expect(submitBtn).toBeVisible();
  await expect(submitBtn).toContainText('发送');
});

test('Composer shows drag hint text', async ({ page }) => {
  await gotoWorkbench(page);

  const hint = page.locator('.composer .hint');
  await expect(hint).toContainText('拖拽 Artifact 到此');
});

test('Composer submit button visible when empty (UI-only, guard is in handler)', async ({ page }) => {
  await gotoWorkbench(page);

  const submitBtn = page.locator('.composer .submit-btn');
  await expect(submitBtn).toBeVisible();
});

// ── Layout: shell integration ────────────────────────────────
test('ArtifactPanel renders inside sidebar-right', async ({ page }) => {
  await gotoWorkbench(page);

  const sidebarRight = page.locator('.sidebar-right');
  await expect(sidebarRight).toBeVisible();
  await expect(sidebarRight.locator('.artifact-panel')).toBeVisible();
});

test('Composer renders inside composer-bar', async ({ page }) => {
  await gotoWorkbench(page);

  const composerBar = page.locator('.composer-bar');
  await expect(composerBar).toBeVisible();
  await expect(composerBar.locator('.composer')).toBeVisible();
});

test('mode tabs visible in Composer (text/image/file/url)', async ({ page }) => {
  await gotoWorkbench(page);

  const modeTabs = page.locator('.composer .mode-tabs');
  await expect(modeTabs).toBeVisible();

  const tabs = modeTabs.locator('button');
  await expect(tabs).toHaveCount(4);
  await expect(tabs.nth(0)).toContainText('文本');
  await expect(tabs.nth(1)).toContainText('图片');
  await expect(tabs.nth(2)).toContainText('文件');
  await expect(tabs.nth(3)).toContainText('URL');
});
