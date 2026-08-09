/**
 * Convergence verification: the heart of the harness. After a scenario's final
 * settle-sync, every device mirror must be byte-identical (same row_uuids,
 * versions, and data) to the server snapshot, and to each other. Any
 * divergence throws with a readable diff — the harness then exits non-zero.
 */

import { COLLECTIONS, type Collection, type MirrorRow, type SnapshotResponse } from '../../src/types';
import { HttpTransport } from '../../src/http';
import type { Device } from './device';

export interface DeviceMirror {
  id: string;
  rows: Record<Collection, Map<string, MirrorRow>>;
}

/** Build a normalized per-collection uuid -> row map from a device mirror. */
export async function deviceMirror(device: Device): Promise<DeviceMirror> {
  const rows = {} as Record<Collection, Map<string, MirrorRow>>;
  for (const c of COLLECTIONS) {
    rows[c] = new Map<string, MirrorRow>();
    for (const row of await device.engine.query(c)) {
      rows[c]!.set(row.rowUuid, row);
    }
  }
  return { id: device.id, rows };
}

/** Fetch the server's authoritative snapshot for the harness account. */
export async function serverSnapshot(baseUrl: string, auth: string): Promise<SnapshotResponse> {
  const transport = new HttpTransport({ baseUrl, token: () => auth });
  return transport.snapshot(COLLECTIONS);
}

export function serverMirror(snap: SnapshotResponse): DeviceMirror {
  const rows = {} as Record<Collection, Map<string, MirrorRow>>;
  for (const c of COLLECTIONS) {
    rows[c] = new Map<string, MirrorRow>();
    for (const row of snap.collections[c] ?? []) {
      rows[c]!.set(row.row_uuid, {
        collection: c,
        rowUuid: row.row_uuid,
        version: row.version,
        data: row.data ?? {},
      });
    }
  }
  return { id: 'server', rows };
}

/**
 * Assert that the server and every device hold identical content:
 * same row_uuids per collection, same version per row, same data per row
 * (semantic equality — key order and number formatting ignored).
 */
export async function assertConvergence(label: string, server: SnapshotResponse, devices: Device[]): Promise<void> {
  const expected = serverMirror(server);
  const mirrors = [expected, ...(await Promise.all(devices.map(deviceMirror)))];
  const problems: string[] = [];

  for (const device of devices) {
    if (await device.engine.pendingChanges() > 0) {
      problems.push(`${label}: ${device.id} still has ${await device.engine.pendingChanges()} pending outbox change(s) after settle`);
    }
  }

  for (const c of COLLECTIONS) {
    const reference = expected.rows[c]!;
    for (const mirror of mirrors.slice(1)) {
      const actual = mirror.rows[c]!;
      // 1. same uuid set
      const aUuids = [...actual.keys()].sort();
      const rUuids = [...reference.keys()].sort();
      if (JSON.stringify(aUuids) !== JSON.stringify(rUuids)) {
        const onlyActual = aUuids.filter((u) => !reference.has(u));
        const onlyServer = rUuids.filter((u) => !actual.has(u));
        problems.push(
          `${label}: ${c} uuid set differs on ${mirror.id}: only-on-device=[${onlyActual.join(',')}] only-on-server=[${onlyServer.join(',')}]`,
        );
        continue;
      }
      // 2. per-row version + data
      for (const [uuid, refRow] of reference) {
        const actRow = actual.get(uuid);
        if (!actRow) continue; // already flagged above
        if (actRow.version !== refRow.version) {
          problems.push(`${label}: ${c} ${uuid} version ${actRow.version} on ${mirror.id} != server ${refRow.version}`);
          continue;
        }
        if (!dataEqual(actRow.data, refRow.data)) {
          problems.push(
            `${label}: ${c} ${uuid} data differs on ${mirror.id}:\n  device: ${JSON.stringify(canonical(actRow.data))}\n  server: ${JSON.stringify(canonical(refRow.data))}`,
          );
        }
      }
    }
  }

  if (problems.length > 0) {
    throw new Error(`CONVERGENCE FAILED (${label})\n${problems.join('\n')}`);
  }
}

/** True when two row payloads are semantically identical (ignores key order). */
export function dataEqual(a: Record<string, unknown>, b: Record<string, unknown>): boolean {
  return JSON.stringify(canonical(a)) === JSON.stringify(canonical(b));
}

/** Recursively sorts object keys so JSON comparison ignores key order. */
export function canonical(v: unknown): unknown {
  if (Array.isArray(v)) return v.map(canonical);
  if (v && typeof v === 'object') {
    const out: Record<string, unknown> = {};
    for (const k of Object.keys(v as Record<string, unknown>).sort()) {
      out[k] = canonical((v as Record<string, unknown>)[k]);
    }
    return out;
  }
  return v;
}
