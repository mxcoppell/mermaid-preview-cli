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

test('SVG export downloads valid SVG file', async ({ page }) => {
  // Open the export menu
  await page.locator('#export-btn').click();
  await expect(page.locator('#export-menu')).not.toHaveClass(/hidden/);

  // Set up download listener before clicking
  const downloadPromise = page.waitForEvent('download');
  await page.locator('#export-svg').click();
  const download = await downloadPromise;

  // Verify the filename ends with .svg
  expect(download.suggestedFilename()).toMatch(/\.svg$/);

  // Read the downloaded content and verify it's valid SVG
  const content = await download.path().then((p) => {
    if (!p) throw new Error('Download path is null');
    return require('fs').readFileSync(p, 'utf-8');
  });
  expect(content).toContain('<svg');
  expect(content).toContain('</svg>');
});

test('PNG export downloads valid PNG file', async ({ page }) => {
  // Open the export menu
  await page.locator('#export-btn').click();
  await expect(page.locator('#export-menu')).not.toHaveClass(/hidden/);

  // Set up download listener
  const downloadPromise = page.waitForEvent('download');
  await page.locator('#export-png').click();
  const download = await downloadPromise;

  // Verify the filename ends with .png
  expect(download.suggestedFilename()).toMatch(/\.png$/);

  // Verify the file exists and has content
  const filePath = await download.path();
  expect(filePath).toBeTruthy();
  const stats = require('fs').statSync(filePath);
  expect(stats.size).toBeGreaterThan(0);
});

test('Print triggers window.print', async ({ page }) => {
  // Override window.print to detect it was called
  await page.evaluate(() => {
    (window as any).__printCalled = false;
    window.print = () => {
      (window as any).__printCalled = true;
    };
  });

  // Open export menu and click print
  await page.locator('#export-btn').click();
  await expect(page.locator('#export-menu')).not.toHaveClass(/hidden/);
  await page.locator('#export-print').click();

  // Verify window.print was called
  const printCalled = await page.evaluate(() => (window as any).__printCalled);
  expect(printCalled).toBe(true);
});
