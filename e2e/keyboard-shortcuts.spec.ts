import { test, expect } from '@playwright/test';
import {
  startServer,
  stopServer,
  testdataPath,
  ServerInstance,
} from './helpers/server';

let server: ServerInstance;

const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';

test.afterEach(async () => {
  if (server) {
    await stopServer(server);
  }
});

test('Esc quits server when search is closed', async ({ page }) => {
  server = await startServer(testdataPath('flowchart.mmd'));
  await page.goto(server.url);
  await expect(page.locator('#diagram svg')).toBeVisible();

  // Make sure search is closed
  await expect(page.locator('#search-bar')).toHaveClass(/hidden/);

  // Press Escape - should trigger server shutdown via POST /api/shutdown
  const shutdownPromise = page.waitForResponse(
    (resp) => resp.url().includes('/api/shutdown') && resp.status() === 200,
  );
  await page.keyboard.press('Escape');
  await shutdownPromise;
});

test('Esc closes search first, then quits on second press', async ({
  page,
}) => {
  server = await startServer(testdataPath('flowchart.mmd'));
  await page.goto(server.url);
  await expect(page.locator('#diagram svg')).toBeVisible();

  // Open search
  await page.keyboard.press(`${modifier}+f`);
  await expect(page.locator('#search-bar')).not.toHaveClass(/hidden/);

  // First Escape should close search (not quit)
  await page.keyboard.press('Escape');
  await expect(page.locator('#search-bar')).toHaveClass(/hidden/);

  // Second Escape should trigger shutdown
  const shutdownPromise = page.waitForResponse(
    (resp) => resp.url().includes('/api/shutdown') && resp.status() === 200,
  );
  await page.keyboard.press('Escape');
  await shutdownPromise;
});
