import { ChildProcess, spawn, execFileSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

export interface ServerInstance {
  url: string;
  port: number;
  process: ChildProcess;
}

const PROJECT_ROOT = path.resolve(__dirname, '..', '..');
const BINARY_PATH = path.join(PROJECT_ROOT, 'bin', 'mermaid-preview');

/**
 * Build the Go binary if it doesn't already exist.
 */
function ensureBinary(): void {
  if (!fs.existsSync(BINARY_PATH)) {
    execFileSync('go', ['build', '-o', 'bin/mermaid-preview', '.'], {
      cwd: PROJECT_ROOT,
      stdio: 'pipe',
    });
  }
}

/**
 * Start the mermaid-preview server for a given file.
 * Waits for the "listening on" line and extracts the port.
 */
export async function startServer(
  filePath: string,
  extraArgs: string[] = [],
): Promise<ServerInstance> {
  ensureBinary();

  const args = ['--port', '0', ...extraArgs, filePath];
  const proc = spawn(BINARY_PATH, args, {
    cwd: PROJECT_ROOT,
    stdio: ['pipe', 'pipe', 'pipe'],
  });

  const { url, port } = await waitForListening(proc);
  return { url, port, process: proc };
}

/**
 * Start the mermaid-preview server reading from stdin.
 * Pipes the provided content into the process stdin.
 */
export async function startServerWithStdin(
  content: string,
  extraArgs: string[] = [],
): Promise<ServerInstance> {
  ensureBinary();

  const args = ['--port', '0', ...extraArgs];
  const proc = spawn(BINARY_PATH, args, {
    cwd: PROJECT_ROOT,
    stdio: ['pipe', 'pipe', 'pipe'],
  });

  // Write content to stdin and close it
  proc.stdin!.write(content);
  proc.stdin!.end();

  const { url, port } = await waitForListening(proc);
  return { url, port, process: proc };
}

/**
 * Wait for the server to print its listening address on stderr.
 * Parses: "mermaid-preview: listening on http://127.0.0.1:XXXXX"
 */
function waitForListening(
  proc: ChildProcess,
): Promise<{ url: string; port: number }> {
  return new Promise((resolve, reject) => {
    let stderrBuf = '';
    const timeout = setTimeout(() => {
      reject(new Error('Server did not start within 10 seconds'));
      proc.kill();
    }, 10_000);

    proc.stderr!.on('data', (chunk: Buffer) => {
      stderrBuf += chunk.toString();
      const match = stderrBuf.match(
        /mermaid-preview: listening on (http:\/\/127\.0\.0\.1:(\d+))/,
      );
      if (match) {
        clearTimeout(timeout);
        resolve({ url: match[1], port: parseInt(match[2], 10) });
      }
    });

    proc.on('error', (err) => {
      clearTimeout(timeout);
      reject(new Error(`Failed to start server: ${err.message}`));
    });

    proc.on('exit', (code) => {
      clearTimeout(timeout);
      if (code !== null && code !== 0) {
        reject(
          new Error(
            `Server exited with code ${code}. stderr: ${stderrBuf}`,
          ),
        );
      }
    });
  });
}

/**
 * Gracefully stop the server process.
 */
export async function stopServer(server: ServerInstance): Promise<void> {
  if (server.process.killed) return;

  return new Promise<void>((resolve) => {
    server.process.on('exit', () => resolve());
    server.process.kill('SIGTERM');

    // Force kill after 3 seconds
    setTimeout(() => {
      if (!server.process.killed) {
        server.process.kill('SIGKILL');
      }
      resolve();
    }, 3_000);
  });
}

/**
 * Resolve a testdata file path relative to the project root.
 */
export function testdataPath(filename: string): string {
  return path.join(PROJECT_ROOT, 'testdata', filename);
}
