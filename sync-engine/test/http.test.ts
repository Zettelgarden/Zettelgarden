import { describe, it, expect } from 'vitest';
import { HttpTransport } from '../src/http';
import type { PushRequest } from '../src/types';

/**
 * Transport wire-shape tests (Zettelgarden-xre). The live Go backend emits
 * snake_case JSON (`row_uuid`, `server_id`, `mapped_to_row_uuid`, `has_more`,
 * `lost_edits`) while the engine contract is camelCase; the harness caught
 * pushes applying server-side with the client's outbox never draining because
 * `result.rowUuid` parsed as undefined. These lock the normalization down
 * against both the Go shape and the camelCase unit-test mock.
 */

function fakeFetch(response: unknown): typeof fetch {
  return (async () => {
    return {
      ok: true,
      status: 200,
      json: async () => response,
    } as Response;
  }) as typeof fetch;
}

const GO_SNAKE_PUSH = {
  results: [
    { row_uuid: 'tag-a', status: 'merged', server_id: 6, server_version: 1, mapped_to_row_uuid: 'survivor', data: { name: 'work' } },
    { row_uuid: 'card-1', status: 'applied', server_id: 2, server_version: 1 },
  ],
  cursor: 11,
  lost_edits: 1,
};

describe('HttpTransport wire normalization', () => {
  it('normalizes the Go backend snake_case push response to the engine contract', async () => {
    const transport = new HttpTransport({
      baseUrl: 'http://server',
      token: () => 'Bearer t',
      fetchImpl: fakeFetch(GO_SNAKE_PUSH),
    });
    const resp = await transport.push({
      changes: [],
      device_id: 'dev-a',
      cursor: 0,
    } as unknown as PushRequest);

    expect(resp.lostEdits).toBe(1);
    expect(resp.results[0]).toMatchObject({
      rowUuid: 'tag-a',
      status: 'merged',
      serverId: 6,
      serverVersion: 1,
      mappedToRowUuid: 'survivor',
    });
    expect(resp.results[1]).toMatchObject({ rowUuid: 'card-1', status: 'applied', serverId: 2 });
  });

  it('normalizes has_more on the changes feed', async () => {
    const transport = new HttpTransport({
      baseUrl: 'http://server',
      token: () => 'Bearer t',
      fetchImpl: fakeFetch({ cursor: 5, rows: [], has_more: true }),
    });
    const resp = await transport.changes(0);
    expect(resp.hasMore).toBe(true);
  });

  it('passes through the camelCase mock-server shape unchanged', async () => {
    const transport = new HttpTransport({
      baseUrl: 'http://server',
      token: () => 'Bearer t',
      fetchImpl: fakeFetch({
        results: [{ rowUuid: 'tag-b', status: 'merged', serverId: 6, serverVersion: 1, mappedToRowUuid: 'survivor' }],
        cursor: 1,
        lostEdits: 0,
      }),
    });
    const resp = await transport.push({ changes: [], device_id: 'dev-b' });
    expect(resp.results[0]).toMatchObject({
      rowUuid: 'tag-b',
      status: 'merged',
      mappedToRowUuid: 'survivor',
    });
  });
});
