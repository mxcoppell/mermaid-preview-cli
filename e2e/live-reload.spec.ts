import { test, expect } from '@playwright/test';
import {
  startServer,
  stopServer,
  testdataPath,
  ServerInstance,
} from './helpers/server';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

let server: ServerInstance;
let tempFile: string | null = null;

test.afterEach(async () => {
  if (server) {
    await stopServer(server);
  }
  if (tempFile && fs.existsSync(tempFile)) {
    fs.unlinkSync(tempFile);
  }
  tempFile = null;
});

test('live reloads when file changes', async ({ page }) => {
  // Create a temp copy of the flowchart
  const srcContent = fs.readFileSync(testdataPath('flowchart.mmd'), 'utf-8');
  tempFile = path.join(os.tmpdir(), `e2e-reload-${Date.now()}.mmd`);
  fs.writeFileSync(tempFile, srcContent);

  server = await startServer(tempFile);
  await page.goto(server.url);

  // Verify initial content renders
  const svg = page.locator('#diagram svg');
  await expect(svg).toBeVisible();
  const initialText = await svg.textContent();
  expect(initialText).toContain('Start');

  // Modify the file with new content
  const newContent = `flowchart TD
    Alpha([Alpha Node]) --> Beta[Beta Node]
    Beta --> Gamma([Gamma Node])
`;
  fs.writeFileSync(tempFile, newContent);

  // Wait for the new content to appear via live reload
  await expect(page.locator('#diagram')).toContainText('Alpha Node', {
    timeout: 10_000,
  });
  await expect(page.locator('#diagram')).toContainText('Beta Node');
});

test('shows disconnected banner when server stops', async ({ page }) => {
  tempFile = path.join(os.tmpdir(), `e2e-disconnect-${Date.now()}.mmd`);
  fs.writeFileSync(
    tempFile,
    'flowchart TD\n    A[Hello] --> B[World]\n',
  );

  server = await startServer(tempFile);
  await page.goto(server.url);

  // Verify diagram renders
  await expect(page.locator('#diagram svg')).toBeVisible();

  // Banner should be hidden initially
  const banner = page.locator('#disconnected-banner');
  await expect(banner).toHaveClass(/hidden/);

  // Kill the server process
  server.process.kill('SIGTERM');

  // The disconnected banner should appear
  await expect(banner).not.toHaveClass(/hidden/, { timeout: 10_000 });
  await expect(banner).toBeVisible();
});

test('recovers from invalid syntax on next valid save', async ({ page }) => {
  tempFile = path.join(os.tmpdir(), `e2e-recover-${Date.now()}.mmd`);
  fs.writeFileSync(
    tempFile,
    'flowchart TD\n    OK[Working] --> Fine[Good]\n',
  );

  server = await startServer(tempFile);
  await page.goto(server.url);

  // Wait for initial render
  await expect(page.locator('#diagram svg')).toBeVisible();
  await expect(page.locator('#diagram')).toContainText('Working');

  // Write invalid syntax
  fs.writeFileSync(tempFile, 'flowchart TD\n    A --> {broken\n');

  // Error overlay should appear
  await expect(page.locator('#error-overlay')).toBeVisible({ timeout: 10_000 });

  // Now write valid content again
  fs.writeFileSync(
    tempFile,
    'flowchart TD\n    Recovered[Recovered] --> Success[Success]\n',
  );

  // Should recover: error overlay hidden, new content visible
  await expect(page.locator('#error-overlay')).toHaveClass(/hidden/, {
    timeout: 10_000,
  });
  await expect(page.locator('#diagram')).toContainText('Recovered');
});
