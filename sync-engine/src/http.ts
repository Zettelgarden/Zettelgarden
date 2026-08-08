/**
 * HTTP SyncTransport backed by fetch(), pointed at the Phase 0b sync API.
 * The auth token is injected by the shell (keychain on desktop, WebView
 * injection on mobile) via the headers callback.
 */

import type {
  ChangesResponse,
  Collection,
  PushRequest,
  PushResponse,
  SnapshotResponse,
  SyncTransport,
} from './types';

export interface HttpTransportOptions {
  baseUrl: string;
  /** Called per request; return the auth header value (e.g. `Bearer <jwt>`). */
  token: () => string | null;
  fetchImpl?: typeof fetch;
}

export class HttpTransport implements SyncTransport {
  private baseUrl: string;
  private token: () => string | null;
  private fetchImpl: typeof fetch;

  constructor(opts: HttpTransportOptions) {
    this.baseUrl = opts.baseUrl.replace(/\/$/, '');
    this.token = opts.token;
    this.fetchImpl = opts.fetchImpl ?? fetch;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const token = this.token();
    const headers: Record<string, string> = {};
    if (token) headers['Authorization'] = token;
    let payload: string | undefined;
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      payload = JSON.stringify(body);
    }
    const res = await this.fetchImpl(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: payload,
    });
    if (!res.ok) {
      const text = await res.text().catch(() => '');
      throw new Error(`sync ${method} ${path}: HTTP ${res.status} ${text.slice(0, 200)}`);
    }
    return (await res.json()) as T;
  }

  snapshot(collections: Collection[]): Promise<SnapshotResponse> {
    return this.request<SnapshotResponse>(
      'GET',
      `/api/sync/snapshot?collections=${collections.join(',')}`,
    );
  }

  changes(since: number): Promise<ChangesResponse> {
    return this.request<ChangesResponse>('GET', `/api/sync/changes?since=${since}`);
  }

  push(req: PushRequest): Promise<PushResponse> {
    return this.request<PushResponse>('POST', '/api/sync/push', req);
  }
}
