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
    return this.request<ChangesResponse>('GET', `/api/sync/changes?since=${since}`).then(
      normalizeChanges,
    );
  }

  push(req: PushRequest): Promise<PushResponse> {
    return this.request<PushResponse>('POST', '/api/sync/push', req).then(normalizePush);
  }
}

/**
 * The Go backend emits snake_case JSON (`has_more`, `row_uuid`…); the engine
 * contract is camelCase for feed/push envelopes (row payloads stay verbatim —
 * `data` is raw column maps). Normalize the wire shape here, once, so the
 * engine and its mock keep one type language. Found by the Phase 1b live
 * harness (Zettelgarden-xre): responses parsed with undefined `row_uuid` left
 * the outbox undrained and versions unadopted.
 */
function normalizeChanges(resp: ChangesResponse): ChangesResponse {
  const raw = resp as unknown as {
    has_more?: boolean;
    hasMore?: boolean;
  };
  return {
    cursor: resp.cursor,
    rows: resp.rows ?? [],
    hasMore: raw.has_more ?? raw.hasMore ?? false,
    reset: resp.reset,
  };
}

function normalizePush(resp: PushResponse): PushResponse {
  const raw = resp as unknown as {
    results?: Array<{
      row_uuid?: string;
      rowUuid?: string;
      status: string;
      server_id?: number;
      serverId?: number;
      server_version?: number;
      serverVersion?: number;
      mapped_to_row_uuid?: string;
      mappedToRowUuid?: string;
      data?: Record<string, unknown>;
    }>;
    cursor: number;
    lost_edits?: number;
    lostEdits?: number;
  };
  return {
    results: (raw.results ?? []).map((r) => ({
      rowUuid: r.row_uuid ?? r.rowUuid,
      status: r.status,
      serverId: r.server_id ?? r.serverId,
      serverVersion: r.server_version ?? r.serverVersion ?? 0,
      mappedToRowUuid: r.mapped_to_row_uuid ?? r.mappedToRowUuid,
      data: r.data,
    })),
    cursor: raw.cursor,
    lostEdits: raw.lost_edits ?? raw.lostEdits ?? 0,
  };
}
