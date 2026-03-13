import { test, expect } from '@playwright/test';
import {
  startServerWithStdin,
  stopServer,
  ServerInstance,
} from './helpers/server';

let server: ServerInstance;

test.afterEach(async () => {
  if (server) {
    await stopServer(server);
  }
});

test('renders diagram from piped stdin', async ({ page }) => {
  const diagramSource = `flowchart LR
    Input([Stdin Input]) --> Process[Process Data]
    Process --> Output([Stdin Output])
`;

  server = await startServerWithStdin(diagramSource);
  await page.goto(server.url);

  // Wait for the SVG to render
  const svg = page.locator('#diagram svg');
  await expect(svg).toBeVisible();

  // Verify the content from stdin was rendered
  const diagramText = await svg.textContent();
  expect(diagramText).toContain('Stdin Input');
  expect(diagramText).toContain('Process Data');
  expect(diagramText).toContain('Stdin Output');
});
