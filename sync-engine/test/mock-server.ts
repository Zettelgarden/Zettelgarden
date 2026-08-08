/**
 * In-memory mock of the Phase 0b sync API, faithful to the Go backend
 * semantics (services/sync_apply.go): row_uuid idempotency, base_version
 * optimistic concurrency with server-wins LWW, tag name-merge, sync_log feed
 * with monotonic ids, snapshot of non-deleted rows. Used to drive the engine
 * through realistic push/pull cycles without a live server.
 */

import type {
  ChangesResponse,
  Collection,
  PushRequest,
  PushResponse,
  SnapshotResponse,
  SyncTransport,
} from '../src/types';

interface ServerRow {
  rowUuid: string;
  version: number;
  data: Record<string, unknown>;
  isDeleted: boolean;
}

interface LogEntry {
  id: number;
  collection: Collection;
  rowUuid: string;
  op: 'upsert' | 'delete';
  version: number;
}

export class MockServer implements SyncTransport {
  private rows = new Map<string, ServerRow>(); // `${collection}:${rowUuid}`
  private log: LogEntry[] = [];
  private nextId = 1;
  private nextLogId = 1;

  private key(c: Collection, uuid: string): string {
    return `${c}:${uuid}`;
  }

  seed(collection: Collection, rowUuid: string, data: Record<string, unknown>, version = 1): void {
    this.rows.set(this.key(collection, rowUuid), {
      rowUuid,
      version,
      data: { ...data, id: this.nextId++ },
      isDeleted: false,
    });
    // Represent the row as created through a normal flow (log entry), so
    // snapshots carry a meaningful cursor and feeds can deliver later changes.
    this.emit(collection, rowUuid, 'upsert', version);
  }

  /** Simulates the Go server: (id, version) for a row. */
  private lookup(collection: Collection, rowUuid: string): ServerRow | undefined {
    return this.rows.get(this.key(collection, rowUuid));
  }

  private emit(collection: Collection, rowUuid: string, op: 'upsert' | 'delete', version: number): void {
    this.log.push({ id: this.nextLogId++, collection, rowUuid, op, version });
  }

  snapshot(collections: Collection[]): Promise<SnapshotResponse> {
    const out: SnapshotResponse['collections'] = {};
    for (const c of collections) {
      out[c] = [];
      for (const [k, row] of this.rows) {
        if (k.startsWith(`${c}:`) && !row.isDeleted) {
          out[c]!.push({
            row_uuid: row.rowUuid,
            version: row.version,
            op: 'upsert',
            data: row.data,
          });
        }
      }
    }
    return Promise.resolve({
      cursor: this.log.length ? this.log[this.log.length - 1]!.id : 0,
      collections: out,
    });
  }

  changes(since: number): Promise<ChangesResponse> {
    const entries = this.log.filter((e) => e.id > since).slice(0, 500);
    const rows = entries.map((e) => {
      const row = this.lookup(e.collection, e.rowUuid);
      return {
        row_uuid: e.rowUuid,
        version: e.version,
        op: e.op,
        collection: e.collection,
        data: row && !row.isDeleted ? row.data : undefined,
      };
    });
    return Promise.resolve({
      cursor: entries.length ? entries[entries.length - 1]!.id : since,
      rows,
      hasMore: false,
    });
  }

  push(req: PushRequest): Promise<PushResponse> {
    const results: PushResponse['results'] = [];
    // Batch-internal tombstones: a row soft-deleted earlier in THIS batch is
    // resurrected by a same-batch upsert (delete-then-recreate), matching the
    // Go server's pushContext.deletedInBatch.
    const deletedInBatch = new Set<string>();
    for (const ch of req.changes) {
      const batchKey = this.key(ch.collection, ch.row_uuid);
      const existing = this.lookup(ch.collection, ch.row_uuid);
      const resurrecting = deletedInBatch.has(batchKey);
      if (!existing) {
        if (ch.op === 'delete') {
          results.push({ rowUuid: ch.row_uuid, status: 'ignored', serverVersion: 0 });
          continue;
        }
        // Tags merge by name; everything else creates.
        if (ch.collection === 'tags') {
          const byName = [...this.rows.values()].find(
            (r) => r.data.name === ch.data?.name && !r.isDeleted,
          );
          if (byName) {
            const merged = ch.base_version > byName.version;
            if (merged) {
              byName.version = ch.base_version + 1;
              byName.data = { ...ch.data, id: byName.data.id };
              this.emit('tags', byName.rowUuid, 'upsert', byName.version);
            }
            results.push({
              rowUuid: ch.row_uuid,
              status: 'merged',
              serverId: byName.data.id as number,
              serverVersion: byName.version,
              mappedToRowUuid: byName.rowUuid,
              data: byName.data,
            });
            continue;
          }
        }
        const id = this.nextId++;
        this.rows.set(this.key(ch.collection, ch.row_uuid), {
          rowUuid: ch.row_uuid,
          version: 1,
          data: { ...ch.data, id },
          isDeleted: false,
        });
        this.emit(ch.collection, ch.row_uuid, 'upsert', 1);
        results.push({ rowUuid: ch.row_uuid, status: 'applied', serverId: id, serverVersion: 1 });
        continue;
      }

      // Existing row.
      if (ch.base_version === 0 && !resurrecting) {
        // Idempotent retry of an applied create.
        results.push({
          rowUuid: ch.row_uuid,
          status: 'applied',
          serverId: existing.data.id as number,
          serverVersion: existing.version,
        });
        continue;
      }
      if (ch.base_version < existing.version && !resurrecting) {
        results.push({
          rowUuid: ch.row_uuid,
          status: 'conflict',
          serverId: existing.data.id as number,
          serverVersion: existing.version,
          data: existing.data,
        });
        continue;
      }
      if (ch.op === 'delete') {
        existing.isDeleted = true;
        existing.version += 1;
        deletedInBatch.add(batchKey);
        this.emit(ch.collection, ch.row_uuid, 'delete', existing.version);
        results.push({
          rowUuid: ch.row_uuid,
          status: 'applied',
          serverId: existing.data.id as number,
          serverVersion: existing.version,
        });
        continue;
      }
      const newVersion = Math.max(existing.version, ch.base_version) + 1;
      existing.version = newVersion;
      existing.data = { ...ch.data, id: existing.data.id };
      existing.isDeleted = ch.data?.is_deleted === true;
      this.emit(ch.collection, ch.row_uuid, 'upsert', newVersion);
      results.push({
        rowUuid: ch.row_uuid,
        status: 'applied',
        serverId: existing.data.id as number,
        serverVersion: newVersion,
      });
    }
    const cursor = this.log.length ? this.log[this.log.length - 1]!.id : 0;
    return Promise.resolve({
      results,
      cursor,
      lostEdits: results.filter((r) => r.status === 'conflict').length,
    });
  }
}
