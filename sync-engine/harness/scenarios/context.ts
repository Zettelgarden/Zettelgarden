/**
 * Shared scenario plumbing: fresh account per scenario, two live devices,
 * settle-sync, convergence assertion, teardown.
 */

import type { SyncSummary } from '../../src/types';
import type { HarnessBackend } from '../lib/backend';
import { makeDevice, type Device } from '../lib/device';
import { assertConvergence, serverSnapshot } from '../lib/verify';

export interface ScenarioContext {
  backend: HarnessBackend;
}

export interface Scenario {
  name: string;
  run(ctx: ScenarioContext): Promise<void>;
}

/**
 * Creates a fresh account + the requested devices, runs the body, then
 * asserts full convergence (server == every device mirror, outboxes empty).
 */
export async function withDevices(
  backend: HarnessBackend,
  tag: string,
  ids: string[],
  fn: (devices: Device[], auth: string, baseUrl: string) => Promise<void>,
): Promise<void> {
  const user = await backend.createUser(tag);
  const devices: Device[] = [];
  try {
    for (const id of ids) {
      devices.push(await makeDevice(backend.baseUrl, user.auth, id, backend.deviceDbPath(id)));
    }
    await fn(devices, user.auth, backend.baseUrl);
  } finally {
    for (const d of devices) d.close();
  }
}

/**
 * Converges both devices: each rounds of sync() pulls remote changes then
 * pushes its own, so after `rounds` passes every echo and cross-device change
 * has been applied everywhere. Idempotent — extra rounds are harmless.
 * Returns every per-device per-round summary so callers can assert on the
 * round that actually pushed (e.g. which device lost an LWW race).
 */
export async function settle(
  devices: Device[],
  rounds = 2,
): Promise<Map<string, SyncSummary[]>> {
  const summaries = new Map<string, SyncSummary[]>();
  for (let round = 0; round < rounds; round++) {
    for (const d of devices) {
      d.engine.setOnline(true);
      const s = await d.engine.sync();
      const list = summaries.get(d.id) ?? [];
      list.push(s);
      summaries.set(d.id, list);
    }
  }
  return summaries;
}

/**
 * First summary (per device) whose push actually drained the outbox — an
 * applied row counts as `pushed`, a stale-base LWW loss as `conflicts` (and
 * its lostEdits), so either means the outbox was flushed that round.
 */
export function pushSummary(
  summaries: Map<string, SyncSummary[]>,
  deviceId: string,
): SyncSummary | undefined {
  return (summaries.get(deviceId) ?? []).find((s) => s.pushed + s.conflicts > 0);
}

/** Fetch the server snapshot and assert convergence, returning the snapshot. */
export async function convergeAndAssert(
  label: string,
  devices: Device[],
  auth: string,
  baseUrl: string,
) {
  const server = await serverSnapshot(baseUrl, auth);
  assertConvergence(label, server, devices);
  return server;
}
