/**
 * MobileStorageAdapter matrix (Zettelgarden-c6l.2): the webview-side adapter
 * must behave exactly like the engine's SqliteStorageAdapter, except its SQL
 * crosses the postMessage bridge to a real SQLite executor (better-sqlite3
 * loopback standing in for op-sqlite on the RN JS thread). This is the same
 * assertion matrix as test/sqlite-adapter.test.ts plus bridge-specific
 * checks (migrate issues the schema, transactions drive BEGIN/COMMIT).
 */

import { describe, expect, it, beforeAll } from "vitest";
import {
  createLoopbackInvoke,
  createSqlExecutor,
  loadMobileStorageAdapter,
} from "./lib/mobileBridge";
import type { StorageAdapter } from "../src/storage";

const { MobileStorageAdapter } = await loadMobileStorageAdapter();

let adapter: StorageAdapter;

beforeAll(async () => {
  const executor = createSqlExecutor(":memory:");
  adapter = new MobileStorageAdapter(createLoopbackInvoke(executor));
  await adapter.whenReady();
});

describe("MobileStorageAdapter (over the bridge)", () => {
  it("mirror CRUD round-trips", async () => {
    await adapter.putRow("cards", {
      collection: "cards",
      rowUuid: "r1",
      version: 3,
      data: { id: 5, title: "hello", body: "world" },
    });
    const row = await adapter.getRow("cards", "r1");
    expect(row).toBeDefined();
    expect(row!.version).toBe(3);
    expect(row!.data.title).toBe("hello");
    expect(row!.data.id).toBe(5);

    await adapter.putRow("cards", {
      collection: "cards",
      rowUuid: "r1",
      version: 4,
      data: { id: 5, title: "updated" },
    });
    expect((await adapter.getRow("cards", "r1"))!.version).toBe(4);

    await adapter.deleteRow("cards", "r1");
    expect(await adapter.getRow("cards", "r1")).toBeUndefined();
  });

  it("collections are isolated", async () => {
    await adapter.putRow("cards", {
      collection: "cards",
      rowUuid: "c1",
      version: 1,
      data: {},
    });
    await adapter.putRow("tasks", {
      collection: "tasks",
      rowUuid: "t1",
      version: 1,
      data: {},
    });
    expect(await adapter.allRows("cards")).toHaveLength(1);
    expect(await adapter.allRows("tasks")).toHaveLength(1);
  });

  it("outbox coalesces by rowUuid, keeping the original base_version", async () => {
    await adapter.enqueue({
      collection: "cards",
      rowUuid: "c1",
      op: "upsert",
      baseVersion: 4,
      data: { title: "a" },
    });
    await adapter.enqueue({
      collection: "cards",
      rowUuid: "c1",
      op: "upsert",
      baseVersion: 4,
      data: { title: "b" },
    });
    await adapter.enqueue({
      collection: "cards",
      rowUuid: "c1",
      op: "upsert",
      baseVersion: 4,
      data: { title: "c" },
    });

    const outbox = await adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.data?.title).toBe("c");
    expect(outbox[0]!.baseVersion).toBe(4);
    expect(await adapter.hasPending("c1")).toBe(true);

    await adapter.dropOutbox("c1");
    expect(await adapter.outbox()).toHaveLength(0);
  });

  it("delete-then-recreate coalesces to an upsert that keeps the original base", async () => {
    await adapter.enqueue({
      collection: "cards",
      rowUuid: "dtr",
      op: "delete",
      baseVersion: 1,
      data: undefined,
    });
    await adapter.enqueue({
      collection: "cards",
      rowUuid: "dtr",
      op: "upsert",
      baseVersion: 0,
      data: { title: "recreated" },
    });

    const outbox = await adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe("upsert");
    expect(outbox[0]!.baseVersion).toBe(1);
  });

  it("cursor and meta persist", async () => {
    expect(await adapter.getCursor()).toBe(0);
    await adapter.setCursor(42);
    expect(await adapter.getCursor()).toBe(42);
    await adapter.setMeta("device_id", "dev-1");
    expect(await adapter.getMeta("device_id")).toBe("dev-1");
  });

  it("transaction rolls back on throw", async () => {
    const before = (await adapter.allRows("cards")).length;
    await expect(
      adapter.transaction(async () => {
        await adapter.putRow("cards", {
          collection: "cards",
          rowUuid: "tx-row",
          version: 1,
          data: {},
        });
        throw new Error("boom");
      }),
    ).rejects.toThrow("boom");
    expect(await adapter.getRow("cards", "tx-row")).toBeUndefined();
    expect(await adapter.allRows("cards")).toHaveLength(before);
  });

  it("transaction commits the write and the outbox enqueue atomically", async () => {
    await adapter.transaction(async () => {
      await adapter.putRow("cards", {
        collection: "cards",
        rowUuid: "atomic",
        version: 1,
        data: {},
      });
      await adapter.enqueue({
        collection: "cards",
        rowUuid: "atomic",
        op: "upsert",
        baseVersion: 0,
        data: {},
      });
    });
    expect(await adapter.getRow("cards", "atomic")).toBeDefined();
    expect(await adapter.hasPending("atomic")).toBe(true);
  });
});

describe("MobileStorageAdapter migrate (bridge surface)", () => {
  it("whenReady creates the mirror schema via sql_exec commands", async () => {
    const executor = createSqlExecutor(":memory:");
    const invokes: string[] = [];
    const invoke = async (cmd: string, args?: Record<string, unknown>) => {
      invokes.push(cmd);
      return createLoopbackInvoke(executor)(cmd, args);
    };
    const a = new MobileStorageAdapter(invoke);
    await a.whenReady();
    expect(invokes[0]).toBe("ping");
    expect(invokes.filter((c) => c === "sql_exec")).toHaveLength(3);
    // Schema exists and is queryable through the bridge.
    const rows = await invoke("sql_query", {
      sql: "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('sync_meta','mirror_rows','sync_outbox') ORDER BY name",
      params: [],
    });
    expect((rows as { name: string }[]).map((r) => r.name).sort()).toEqual([
      "mirror_rows",
      "sync_meta",
      "sync_outbox",
    ]);
    executor.close();
  });
});
