/**
 * Live-backend lifecycle for the two-DB convergence harness (Zettelgarden-xre).
 *
 * Builds the Go backend binary (if stale), spawns it against a throwaway
 * SQLite file in a temp dir, waits for the HTTP server to accept requests,
 * and exposes createUser() so each scenario gets a fresh, isolated account.
 * All scenario traffic goes through the real REST + sync API over HTTP — no
 * mocks, no in-process handlers.
 */

import { execSync, spawn, type ChildProcess } from 'node:child_process';
import * as fs from 'node:fs';
import * as net from 'node:net';
import * as os from 'node:os';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const HARNESS_DIR = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const SYNC_ENGINE_DIR = path.dirname(HARNESS_DIR);
const GO_BACKEND_DIR = path.resolve(SYNC_ENGINE_DIR, '..', 'go-backend');
const CACHE_DIR = path.join(HARNESS_DIR, '.cache');
const BIN_NAME = process.platform === 'win32' ? 'zg-backend.exe' : 'zg-backend';
const BIN_PATH = path.join(CACHE_DIR, BIN_NAME);

/** A freshly-registered harness account (one account == one sync user). */
export interface HarnessUser {
  email: string;
  /** Authorization header value, e.g. `Bearer <jwt>`. */
  auth: string;
  jwt: string;
}

export class HarnessBackend {
  readonly baseUrl: string;
  private proc: ChildProcess | null = null;
  private tmpDir: string;
  private port: number;
  private userCounter = 0;
  private dbCounter = 0;
  private logTail: string[] = [];

  private constructor(baseUrl: string, port: number, tmpDir: string) {
    this.baseUrl = baseUrl;
    this.port = port;
    this.tmpDir = tmpDir;
  }

  // ---- lifecycle -----------------------------------------------------------

  static async start(): Promise<HarnessBackend> {
    const bin = ensureBackendBinary();
    // The free-port probe closes its socket before the backend binds, so the
    // port can be stolen in between (TOCTOU); retry with a fresh port if the
    // child dies during boot rather than failing the whole run.
    let lastError: Error | undefined;
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        return await HarnessBackend.tryStart(bin);
      } catch (err) {
        lastError = err instanceof Error ? err : new Error(String(err));
      }
    }
    throw lastError ?? new Error('backend failed to start');
  }

  private static async tryStart(bin: string): Promise<HarnessBackend> {
    const port = await findFreePort();
    const baseUrl = `http://127.0.0.1:${port}`;
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'zg-harness-'));
    const backend = new HarnessBackend(baseUrl, port, tmpDir);

    // Whitelisted env (NOT ...process.env): the harness backend must boot
    // identically regardless of what the developer's shell exports — a stray
    // MAIL_ENABLED/ZETTEL_* would otherwise change settings seeding and
    // silently alter scenario behavior. Mirrors go-backend/tests/conftest.go
    // defaults; dev mode tolerates placeholders; Typesense is unreachable and
    // degrades to SQL search; no SMTP_HOST -> mail off.
    const env: NodeJS.ProcessEnv = {
      PATH: process.env.PATH ?? '',
      HOME: process.env.HOME,
      TMPDIR: process.env.TMPDIR,
      ZETTEL_DEV: 'true',
      ZETTEL_PORT: String(port),
      ZETTEL_URL: baseUrl,
      SQLITE_PATH: path.join(tmpDir, 'server.db'),
      SECRET_KEY: 'harness-secret-key-for-jwt-signing-32-chars-min',
      STORAGE_DIR: path.join(tmpDir, 'files'),
      ZETTEL_BACKEND_LOG_LOCATION: path.join(tmpDir, 'backend.log'),
      TYPESENSE_HOST: 'http://127.0.0.1:8108',
      TYPESENSE_PASSWORD: 'harness-typesense-password',
      TYPESENSE_COLLECTION: 'zettelgarden_harness',
      ZETTEL_LLM_KEY: 'harness-llm-key',
      ZETTEL_LLM_ENDPOINT: 'https://api.z.ai/api/coding/paas/v4',
      ZETTEL_LLM_DEFAULT_MODEL: 'glm-5.1',
      ZETTEL_LLM_SUMMARIZE_MODEL: 'glm-5.1',
      STRIPE_SECRET_KEY: 'harness-stripe-secret',
      STRIPE_PUBLISHABLE_KEY: 'harness-stripe-publishable',
      STRIPE_WEBHOOK_SECRET: 'harness-stripe-webhook-secret',
      STRIPE_MONTH_PRICE: 'price_monthly_test_id',
      STRIPE_YEAR_PRICE: 'price_yearly_test_id',
      STRIPE_BILLING_URL: 'https://billing.example.test',
      GITHUB_AUTH_ENABLED: 'false',
      ZETTEL_RUN_CHUNKING_EMBEDDING: 'false',
    };

    backend.proc = spawn(bin, [], { env, cwd: GO_BACKEND_DIR, stdio: ['ignore', 'pipe', 'pipe'] });
    backend.proc.stdout?.on('data', (d: Buffer) => backend.capture(d));
    backend.proc.stderr?.on('data', (d: Buffer) => backend.capture(d));
    backend.proc.on('exit', (code, signal) => {
      backend.logTail.push(`[proc exit code=${code} signal=${signal}]`);
    });

    try {
      await backend.waitReady();
      return backend;
    } catch (err) {
      // Never leak the child or the temp dir on a failed boot.
      await backend.stop().catch(() => undefined);
      throw err;
    }
  }

  private capture(chunk: Buffer): void {
    const lines = chunk.toString().split('\n');
    for (const line of lines) if (line.trim()) this.logTail.push(line);
    if (this.logTail.length > 2000) this.logTail.splice(0, this.logTail.length - 2000);
  }

  /** Polls a public endpoint until the server answers (or the child dies). */
  private async waitReady(timeoutMs = 90_000): Promise<void> {
    const deadline = Date.now() + timeoutMs;
    for (;;) {
      if (this.proc?.exitCode !== null && this.proc?.exitCode !== undefined) {
        throw new Error(`backend exited during boot (code=${this.proc.exitCode})\n${this.logTail.slice(-60).join('\n')}`);
      }
      try {
        const res = await fetch(`${this.baseUrl}/api/settings`, { signal: AbortSignal.timeout(1500) });
        if (res.ok) return;
      } catch {
        // not up yet
      }
      if (Date.now() > deadline) {
        throw new Error(`backend did not become ready in ${timeoutMs}ms\n${this.logTail.slice(-60).join('\n')}`);
      }
      await sleep(250);
    }
  }

  async stop(): Promise<void> {
    const proc = this.proc;
    this.proc = null;
    if (proc && proc.exitCode === null) {
      proc.kill('SIGTERM');
      const exited = await Promise.race([
        new Promise<boolean>((resolve) => proc.once('exit', () => resolve(true))),
        sleep(8000).then(() => false),
      ]);
      if (!exited) proc.kill('SIGKILL');
    }
    await sleep(200);
    fs.rmSync(this.tmpDir, { recursive: true, force: true });
  }

  /** Last log lines for failure diagnostics. */
  logs(): string[] {
    return this.logTail.slice(-80);
  }

  /** A fresh temp file path for a device DB (parent dir exists). */
  deviceDbPath(tag: string): string {
    return path.join(this.tmpDir, `${tag}-${++this.dbCounter}.db`);
  }

  // ---- accounts ------------------------------------------------------------

  /** Registers a fresh account and logs in, returning the bearer auth. */
  async createUser(tag: string): Promise<HarnessUser> {
    const email = `harness-${tag}-${++this.userCounter}@zettelgarden.test`;
    const password = 'harness-pass-123!';
    const register = await fetch(`${this.baseUrl}/api/users`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: `${tag}-${this.userCounter}`,
        email,
        password,
        confirmPassword: password,
      }),
    });
    if (!register.ok) {
      throw new Error(`register ${email}: HTTP ${register.status} ${await register.text()}`);
    }
    const login = await fetch(`${this.baseUrl}/api/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    if (!login.ok) {
      throw new Error(`login ${email}: HTTP ${login.status} ${await login.text()}`);
    }
    const body = (await login.json()) as { access_token?: string; message?: string };
    if (!body.access_token) {
      throw new Error(`login ${email}: no access_token in response: ${JSON.stringify(body)}`);
    }
    return { email, auth: `Bearer ${body.access_token}`, jwt: body.access_token };
  }
}

// ---- helpers ---------------------------------------------------------------

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Reserves a free TCP port on loopback. */
function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.once('error', reject);
    srv.listen(0, '127.0.0.1', () => {
      const port = (srv.address() as net.AddressInfo).port;
      srv.close(() => resolve(port));
    });
  });
}

/**
 * Builds the backend binary once per harness run (cached in .cache/), then
 * reuses it while no Go source is newer. `go build` is incremental, so the
 * rebuild path is cheap.
 */
function ensureBackendBinary(): string {
  fs.mkdirSync(CACHE_DIR, { recursive: true });
  if (fs.existsSync(BIN_PATH) && !goSourcesNewerThan(BIN_PATH)) {
    return BIN_PATH;
  }
  execSync(`go build -o ${JSON.stringify(BIN_PATH)} .`, {
    cwd: GO_BACKEND_DIR,
    stdio: ['ignore', 'inherit', 'inherit'],
  });
  return BIN_PATH;
}

function goSourcesNewerThan(bin: string): boolean {
  try {
    // Newest Go source or module file by mtime (GNU find printf).
    const listing = execSync(
      `find . \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -printf '%T@ %p\\n' | sort -rn | head -1`,
      { cwd: GO_BACKEND_DIR, encoding: 'utf8' },
    ).trim();
    if (listing === '') return true; // no sources: be safe, rebuild
    const newestMtimeMs = Number(listing.split(/\s+/)[0]) * 1000;
    return newestMtimeMs > fs.statSync(bin).mtimeMs;
  } catch {
    return true; // be safe: rebuild
  }
}
