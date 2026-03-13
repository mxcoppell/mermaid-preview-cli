import { test, expect } from '@playwright/test';
import {
  startServer,
  stopServer,
  testdataPath,
  ServerInstance,
} from './helpers/server';

let server: ServerInstance;

test.afterEach(async () => {
  if (server) {
    await stopServer(server);
  }
});

test('renders flowchart with expected node text', async ({ page }) => {
  server = await startServer(testdataPath('flowchart.mmd'));
  await page.goto(server.url);

  // Wait for SVG to appear in the diagram container
  const svg = page.locator('#diagram svg');
  await expect(svg).toBeVisible();

  // Verify expected node text is present
  const diagramText = await svg.textContent();
  expect(diagramText).toContain('Start');
  expect(diagramText).toContain('Validate Input');
  expect(diagramText).toContain('Process Data');
  expect(diagramText).toContain('Show Error');
  expect(diagramText).toContain('Save to DB');
  expect(diagramText).toContain('End');
});

test('renders sequence diagram with actor names', async ({ page }) => {
  server = await startServer(testdataPath('sequence.mmd'));
  await page.goto(server.url);

  const svg = page.locator('#diagram svg');
  await expect(svg).toBeVisible();

  // Verify actor names are present
  const diagramText = await svg.textContent();
  expect(diagramText).toContain('Client');
  expect(diagramText).toContain('Server');
  expect(diagramText).toContain('Database');
});

test('shows error overlay for invalid syntax', async ({ page }) => {
  server = await startServer(testdataPath('invalid.mmd'));
  await page.goto(server.url);

  // The error overlay should become visible
  const errorOverlay = page.locator('#error-overlay');
  await expect(errorOverlay).toBeVisible();

  // The error overlay should not have the hidden class
  await expect(errorOverlay).not.toHaveClass(/hidden/);
});

test('renders markdown with multiple mermaid blocks', async ({ page }) => {
  server = await startServer(testdataPath('doc-with-diagrams.md'));
  await page.goto(server.url);

  // Wait for SVGs to render - the markdown file has 2 mermaid blocks
  const svgs = page.locator('#diagram svg');
  await expect(svgs.first()).toBeVisible();
  const count = await svgs.count();
  expect(count).toBeGreaterThanOrEqual(2);

  // Verify content from the flowchart block
  const allText = await page.locator('#diagram').textContent();
  expect(allText).toContain('Client');
  expect(allText).toContain('Gateway');

  // Verify content from the sequence diagram block
  expect(allText).toContain('User');
  expect(allText).toContain('API');
});
