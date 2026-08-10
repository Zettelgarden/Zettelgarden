/**
 * Scenario 11 — mobile shell convergence (Zettelgarden-c6l.2): one device is
 * a MobileStorageAdapter over the postMessage bridge (loopback to a real
 * SQLite executor, mirroring the RN shell: webview adapter → bridge → native
 * SQLite); the other is the standard desktop SqliteStorageAdapter. Both share
 * one account; offline edits on the mobile device reconcile on reconnect and
 * every mirror converges — the Phase 3a acceptance shape ("sync converges
 * with a desktop client sharing the same account").
 */

import type { Scenario } from "./context";
import { convergeAndAssert, settle } from "./context";
import { makeDeviceWithStorage, type Device } from "../lib/device";
import {
  createLoopbackInvoke,
  createSqlExecutor,
  loadMobileStorageAdapter,
} from "../../test/lib/mobileBridge";
import { SqliteStorageAdapter } from "../../src/sqlite";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const { MobileStorageAdapter } = await loadMobileStorageAdapter();

export const mobileBridgeScenario: Scenario = {
  name: "11 mobile bridge: MobileStorageAdapter (bridge→SQLite) converges with a desktop client",
  run: async ({ backend }) => {
    const user = await backend.createUser("mobile-bridge");
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "zg-mobile-bridge-"));

    // Mobile device: adapter over the loopback bridge backed by a real
    // SQLite file (the RN shell's op-sqlite executor in production).
    const mobileExecutor = createSqlExecutor(path.join(tmp, "mobile.db"));
    const mobileStorage = new MobileStorageAdapter(
      createLoopbackInvoke(mobileExecutor),
    );
    const mobile = await makeDeviceWithStorage(
      backend.baseUrl,
      user.auth,
      "mobile",
      mobileStorage,
      () => mobileExecutor.close(),
    );

    // Desktop device: the standard engine adapter (Tauri shell shape).
    const desktopStorage = new SqliteStorageAdapter(
      path.join(tmp, "desktop.db"),
    );
    const desktop = await makeDeviceWithStorage(
      backend.baseUrl,
      user.auth,
      "desktop",
      desktopStorage,
      () => desktopStorage.close(),
    );

    const devices: Device[] = [mobile, desktop];
    try {
      await mobile.engine.bootstrap();
      await desktop.engine.bootstrap();
      const baselineCards = (await mobile.engine.query("cards")).length;
      const baselineTags = (await mobile.engine.query("tags")).length;

      // Mobile edits while offline; desktop edits independently.
      mobile.engine.setOnline(false);
      await mobile.engine.mutate("cards", {
        rowUuid: "mobile-card",
        data: {
          title: "written on the phone",
          card_id: "m1",
          body: "offline note",
        },
      });
      await mobile.engine.mutate("tasks", {
        rowUuid: "mobile-task",
        data: { title: "phone task" },
      });
      await mobile.engine.mutate("tags", {
        rowUuid: "mobile-tag",
        data: { name: "mobile-tag", color: "green" },
      });

      desktop.engine.setOnline(false);
      await desktop.engine.mutate("cards", {
        rowUuid: "desktop-card",
        data: {
          title: "written on the desktop",
          card_id: "d1",
          body: "online note",
        },
      });

      // Reconnect both and settle.
      await settle([mobile, desktop], 2);

      const server = await convergeAndAssert(
        "mobile bridge",
        devices,
        user.auth,
        backend.baseUrl,
      );

      const serverCards = server.collections.cards ?? [];
      const serverTasks = server.collections.tasks ?? [];
      const serverTags = server.collections.tags ?? [];
      if (
        serverCards.length !== baselineCards + 2 ||
        serverTasks.length !== 1 ||
        serverTags.length !== baselineTags + 1
      ) {
        throw new Error(
          `mobile bridge: expected ${baselineCards + 2} cards / 1 task / ${baselineTags + 1} tags on the server, got ${serverCards.length} / ${serverTasks.length} / ${serverTags.length}`,
        );
      }
      // The mobile-originated rows made it server-side under their uuids.
      const serverUuids = new Set(
        serverCards.map((c: { row_uuid: string }) => c.row_uuid),
      );
      if (!serverUuids.has("mobile-card") || !serverUuids.has("desktop-card")) {
        throw new Error(
          "mobile bridge: both devices' cards missing from the server",
        );
      }
    } finally {
      for (const d of devices) d.close();
      fs.rmSync(tmp, { recursive: true, force: true });
    }
  },
};
