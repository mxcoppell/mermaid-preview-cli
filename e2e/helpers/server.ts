import { ChildProcess, spawn, execFileSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

export interface ServerInstance {
  url: string;
  port: number;
  process: ChildProcess;
}

const PROJECT_ROOT = path.resolve(__dirname, '..', '..');
const BINARY_PATH = path.join(PROJECT_ROOT, 'mmdp');

/**
 * Build the Go binary if it doesn't already exist.
 */
function ensureBinary(): void {
  if (!fs.existsSync(BINARY_PATH)) {
    execFileSync('go', ['build', '-o', 'mmdp', './cmd/mmdp'], {
      cwd: PROJECT_ROOT,
      stdio: 'pipe',
    });
  }
}

/**
 * Start the mmdp server for a given file.
 * Waits for the "listening on" line and extracts the port.
 */
export async function startServer(
  filePath: string,
  extraArgs: string[] = [],
): Promise<ServerInstance> {
  ensureBinary();

  const args = ['--verbose', '--port', '0', ...extraArgs, filePath];
  const proc = spawn(BINARY_PATH, args, {
    cwd: PROJECT_ROOT,
    stdio: ['pipe', 'pipe', 'pipe'],
  });

  const { url, port } = await waitForListening(proc);
  return { url, port, process: proc };
}

/**
 * Start the mmdp server reading from stdin.
 * Pipes the provided content into the process stdin.
 */
export async function startServerWithStdin(
  content: string,
  extraArgs: string[] = [],
): Promise<ServerInstance> {
  ensureBinary();

  const args = ['--verbose', '--port', '0', ...extraArgs];
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
 * Parses: "mmdp: listening on http://127.0.0.1:XXXXX"
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
        /mmdp: listening on (http:\/\/127\.0\.0\.1:(\d+))/,
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
 *
 * The CLI parent spawns a detached GUI child (`--internal-gui`) and exits
 * immediately, so `server.process` is already dead by the time tests run.
 * The real owner of the HTTP server + webview is the orphaned child process
 * listening on the port. We find it via `lsof` and kill it directly.
 */
export async function stopServer(server: ServerInstance): Promise<void> {
  // Kill the actual GUI process owning the port
  try {
    const pid = execFileSync('lsof', ['-ti', `tcp:${server.port}`], {
      encoding: 'utf-8',
    }).trim();
    if (pid) {
      // May be multiple PIDs (one per line) — kill them all
      for (const p of pid.split('\n')) {
        const n = parseInt(p, 10);
        if (n > 0) {
          try {
            process.kill(n, 'SIGTERM');
          } catch {
            // already exited
          }
        }
      }
    }
  } catch {
    // lsof may fail if process already exited
  }

  // Also clean up the parent ChildProcess handle if still around
  if (!server.process.killed) {
    try {
      server.process.kill('SIGTERM');
    } catch {
      // already exited
    }
  }

  // Brief wait for cleanup
  await new Promise<void>((resolve) => setTimeout(resolve, 200));
}

/**
 * Resolve a testdata file path relative to the project root.
 */
export function testdataPath(filename: string): string {
  return path.join(PROJECT_ROOT, 'testdata', filename);
}
