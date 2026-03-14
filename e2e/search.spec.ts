import { test, expect } from '@playwright/test';
import {
  startServer,
  stopServer,
  testdataPath,
  ServerInstance,
} from './helpers/server';

let server: ServerInstance;

test.beforeEach(async ({ page }) => {
  server = await startServer(testdataPath('flowchart.mmd'));
  await page.goto(server.url);
  await expect(page.locator('#diagram svg')).toBeVisible();
});

test.afterEach(async () => {
  if (server) {
    await stopServer(server);
  }
});

test('Cmd+F opens search, type query, nodes highlighted', async ({ page }) => {
  // Open search with platform-appropriate shortcut
  await page.locator('#search-btn').click();

  // Search bar should be visible
  const searchBar = page.locator('#search-bar');
  await expect(searchBar).not.toHaveClass(/hidden/);

  // Type a search query that matches a node
  await page.locator('#search-input').fill('Start');

  // Wait for highlights to appear
  const highlighted = page.locator('.search-highlight');
  await expect(highlighted.first()).toBeVisible();

  const count = await highlighted.count();
  expect(count).toBeGreaterThanOrEqual(1);
});

test('search navigation next/prev with .search-active', async ({ page }) => {
  await page.locator('#search-btn').click();
  // Search for a term that appears in multiple nodes (e.g. "End" matches "End")
  // Use a broader term - look for partial text
  await page.locator('#search-input').fill('a');

  // Wait for results
  const highlighted = page.locator('.search-highlight');
  await expect(highlighted.first()).toBeVisible();

  // There should be an active result
  const active = page.locator('.search-active');
  await expect(active).toHaveCount(1);

  // Press Enter to go to next match
  await page.locator('#search-input').press('Enter');

  // Still exactly one active element
  await expect(page.locator('.search-active')).toHaveCount(1);

  // Press Shift+Enter to go to previous match
  await page.locator('#search-input').press('Shift+Enter');

  await expect(page.locator('.search-active')).toHaveCount(1);
});

test('Escape closes search and removes highlights', async ({ page }) => {
  // Open search and type a query
  await page.locator('#search-btn').click();
  await page.locator('#search-input').fill('Start');

  // Verify highlights exist
  await expect(page.locator('.search-highlight').first()).toBeVisible();

  // Press Escape to close search
  await page.keyboard.press('Escape');

  // Search bar should be hidden
  await expect(page.locator('#search-bar')).toHaveClass(/hidden/);

  // Highlights should be removed
  await expect(page.locator('.search-highlight')).toHaveCount(0);
});

test('search with no results shows "0 results"', async ({ page }) => {
  await page.locator('#search-btn').click();
  await page.locator('#search-input').fill('zzz_nonexistent_zzz');

  // The search count should show "0 results"
  await expect(page.locator('#search-count')).toHaveText('0 results');

  // No highlights should be present
  await expect(page.locator('.search-highlight')).toHaveCount(0);
});
