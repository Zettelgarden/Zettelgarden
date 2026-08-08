import { describe, it, expect, beforeEach, vi } from "vitest";
import { apiClient } from "./client";
import type { AppSettings } from "./settings";

vi.mock("./client", () => ({
  apiClient: { get: vi.fn() },
  getData: vi.fn(async (p: Promise<{ data: unknown }>) => (await p).data),
}));

describe("getSettings", () => {
  beforeEach(() => {
    // Reset module registry so the module-level settingsPromise cache (page-
    // lifetime by design) starts fresh per test.
    vi.resetModules();
    vi.clearAllMocks();
  });

  it("parses raw string settings into typed AppSettings", async () => {
    const { getSettings } = await import("./settings");
    vi.mocked(apiClient.get).mockResolvedValue({
      data: {
        site_name: "My Notes",
        signups_enabled: "true",
        mail_enabled: "false",
        email_auto_validate: "true",
        support_email: "help@example.com",
      },
      response: {} as Response,
    });

    const settings: AppSettings = await getSettings();
    expect(settings).toEqual({
      siteName: "My Notes",
      signupsEnabled: true,
      mailEnabled: false,
      emailAutoValidate: true,
      supportEmail: "help@example.com",
    });
    expect(vi.mocked(apiClient.get)).toHaveBeenCalledWith("/settings");
  });

  it("falls back to defaults when the fetch fails", async () => {
    const { getSettings } = await import("./settings");
    vi.mocked(apiClient.get).mockRejectedValue(new Error("network"));

    const settings: AppSettings = await getSettings();
    expect(settings).toEqual({
      siteName: "Zettelgarden",
      signupsEnabled: true,
      mailEnabled: true,
      emailAutoValidate: true,
      supportEmail: "",
    });
  });
});
