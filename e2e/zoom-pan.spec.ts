import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import {
  startServer,
  stopServer,
  testdataPath,
  ServerInstance,
} from './helpers/server';

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

test.describe('basic zoom/pan with flowchart', () => {
  test.beforeEach(async ({ page }) => {
    server = await startServer(testdataPath('flowchart.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();
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

    // CSS transform should include a scale > 1 (read from style, not computed matrix)
    const transform = await page.locator('#diagram-wrapper').evaluate(
      (el) => el.style.transform,
    );
    expect(transform).toContain('scale');
  });

  test('- key zooms out', async ({ page }) => {
    // Zoom in first so we have a known numeric baseline
    await page.keyboard.press('+');
    const zoomedIn = await page.locator('#zoom-level').textContent();
    const zoomedInVal = parseInt(zoomedIn!);

    // Press - to zoom out
    await page.keyboard.press('-');

    const zoomText = await page.locator('#zoom-level').textContent();
    expect(zoomText).toMatch(/^\d+%$/);
    expect(parseInt(zoomText!)).toBeLessThan(zoomedInVal);
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
});

/**
 * Helper: extract scale from the wrapper's inline transform style.
 */
function getScale(el: HTMLElement): number {
  const t = el.style.transform;
  const m = t.match(/scale\(([^)]+)\)/);
  return m ? parseFloat(m[1]) : 1;
}

/**
 * Helper: measure how well the SVG content is centered within the container.
 * Returns { offsetX, offsetY } — the distance in px from perfect center.
 * Also returns containerRect and svgRect for further assertions.
 */
async function measureFit(page: import('@playwright/test').Page) {
  return page.evaluate(() => {
    const container = document.getElementById('diagram-container')!;
    const svg = document.querySelector('#diagram svg') as SVGElement;
    if (!container || !svg) return null;
    const cr = container.getBoundingClientRect();
    const sr = svg.getBoundingClientRect();
    const containerCX = cr.left + cr.width / 2;
    const containerCY = cr.top + cr.height / 2;
    const svgCX = sr.left + sr.width / 2;
    const svgCY = sr.top + sr.height / 2;
    return {
      offsetX: Math.abs(containerCX - svgCX),
      offsetY: Math.abs(containerCY - svgCY),
      svgWidth: sr.width,
      svgHeight: sr.height,
      containerWidth: cr.width,
      containerHeight: cr.height,
      svgLeft: sr.left - cr.left,
      svgRight: cr.right - sr.right,
      svgTop: sr.top - cr.top,
      svgBottom: cr.bottom - sr.bottom,
    };
  });
}

// Max acceptable centering offset in px (accounts for rounding, padding asymmetry)
const CENTER_TOLERANCE = 30;

test.describe('auto-fit', () => {
  test('tall diagram (large.mmd): scales down and centers', async ({ page }) => {
    server = await startServer(testdataPath('large.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const scale = await page.locator('#diagram-wrapper').evaluate(getScale);
    expect(scale).toBeLessThan(1);

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('wide diagram (wide.mmd): fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('wide.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
    // Wide diagram: left/right margins should be roughly equal
    expect(Math.abs(fit!.svgLeft - fit!.svgRight)).toBeLessThan(CENTER_TOLERANCE);
  });

  test('tiny diagram (tiny.mmd): fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('tiny.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('small diagram (flowchart.mmd): fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('flowchart.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('wide balanced diagram (sequence.mmd): fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('sequence.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
    // SVG should be fully visible (not clipped)
    expect(fit!.svgLeft).toBeGreaterThanOrEqual(-1);
    expect(fit!.svgTop).toBeGreaterThanOrEqual(-1);
  });

  test('radial diagram (mindmap.mmd): fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('mindmap.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('gantt chart (gantt.mmd): wide timeline fits and centers', async ({ page }) => {
    server = await startServer(testdataPath('gantt.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('0 key re-fits after manual zoom', async ({ page }) => {
    server = await startServer(testdataPath('wide.mmd'));
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    // Zoom in, disrupting fit
    await page.keyboard.press('+');
    await page.keyboard.press('+');
    await expect(page.locator('#zoom-level')).not.toHaveText('Fit');

    // Press 0 to re-fit
    await page.keyboard.press('0');
    await expect(page.locator('#zoom-level')).toHaveText('Fit');

    // Should be centered again
    const fit = await measureFit(page);
    expect(fit).not.toBeNull();
    expect(fit!.offsetX).toBeLessThan(CENTER_TOLERANCE);
    expect(fit!.offsetY).toBeLessThan(CENTER_TOLERANCE);
  });

  test('live reload preserves zoom/pan state', async ({ page }) => {
    const srcContent = fs.readFileSync(testdataPath('flowchart.mmd'), 'utf-8');
    tempFile = path.join(os.tmpdir(), `e2e-zoom-preserve-${Date.now()}.mmd`);
    fs.writeFileSync(tempFile, srcContent);

    server = await startServer(tempFile);
    await page.goto(server.url);
    await expect(page.locator('#diagram svg')).toBeVisible();

    // Zoom in twice
    await page.keyboard.press('+');
    await page.keyboard.press('+');

    const zoomBefore = await page.locator('#zoom-level').textContent();
    expect(zoomBefore).toMatch(/^\d+%$/);

    // Overwrite file with different valid content
    const newContent = `flowchart TD
    X([New Start]) --> Y[New Process]
    Y --> Z([New End])
`;
    fs.writeFileSync(tempFile, newContent);

    // Wait for new content to appear
    await expect(page.locator('#diagram')).toContainText('New Start', {
      timeout: 10_000,
    });

    // Zoom level should be approximately the same (not reset to "Fit")
    const zoomAfter = await page.locator('#zoom-level').textContent();
    expect(zoomAfter).not.toBe('Fit');
    expect(zoomAfter).toBe(zoomBefore);
  });
});
