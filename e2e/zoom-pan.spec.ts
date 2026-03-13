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

test('+ key zooms in', async ({ page }) => {
  // Initial zoom should be "Fit"
  await expect(page.locator('#zoom-level')).toHaveText('Fit');

  // Press + to zoom in
  await page.keyboard.press('+');

  // Zoom level should change from "Fit" to a percentage
  const zoomText = await page.locator('#zoom-level').textContent();
  expect(zoomText).toMatch(/^\d+%$/);
  expect(parseInt(zoomText!)).toBeGreaterThan(100);

  // CSS transform should include a scale > 1
  const transform = await page.locator('#diagram-wrapper').evaluate(
    (el) => getComputedStyle(el).transform || el.style.transform,
  );
  expect(transform).toContain('scale');
});

test('- key zooms out', async ({ page }) => {
  // Press - to zoom out
  await page.keyboard.press('-');

  const zoomText = await page.locator('#zoom-level').textContent();
  expect(zoomText).toMatch(/^\d+%$/);
  expect(parseInt(zoomText!)).toBeLessThan(100);
});

test('0 key resets zoom', async ({ page }) => {
  // Zoom in first
  await page.keyboard.press('+');
  await page.keyboard.press('+');

  const zoomedText = await page.locator('#zoom-level').textContent();
  expect(zoomedText).not.toBe('Fit');

  // Press 0 to reset
  await page.keyboard.press('0');

  await expect(page.locator('#zoom-level')).toHaveText('Fit');
});

test('mouse wheel zooms', async ({ page }) => {
  const container = page.locator('#diagram-container');
  const box = await container.boundingBox();
  expect(box).not.toBeNull();

  // Scroll up (zoom in) at the center of the container
  await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
  await page.mouse.wheel(0, -100);

  // Zoom level should change
  const zoomText = await page.locator('#zoom-level').textContent();
  expect(zoomText).not.toBe('Fit');
});

test('drag to pan', async ({ page }) => {
  // First zoom in so that pan is meaningful
  await page.keyboard.press('+');
  await page.keyboard.press('+');

  const container = page.locator('#diagram-container');
  const box = await container.boundingBox();
  expect(box).not.toBeNull();

  // Get the initial transform
  const initialTransform = await page
    .locator('#diagram-wrapper')
    .evaluate((el) => el.style.transform);

  // Drag from center to the right
  const startX = box!.x + box!.width / 2;
  const startY = box!.y + box!.height / 2;

  await page.mouse.move(startX, startY);
  await page.mouse.down();
  await page.mouse.move(startX + 100, startY + 50, { steps: 5 });
  await page.mouse.up();

  // The transform should have changed (indicating pan)
  const finalTransform = await page
    .locator('#diagram-wrapper')
    .evaluate((el) => el.style.transform);

  expect(finalTransform).not.toBe(initialTransform);
});
