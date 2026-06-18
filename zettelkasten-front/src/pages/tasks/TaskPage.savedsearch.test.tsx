// @vitest-environment happy-dom

import React from "react";
import { cleanup, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, vi, expect, beforeEach, afterEach } from "vitest";
import type { Mock } from "vitest";
import { TaskPage } from "./TaskPage";
import { renderWithProviders } from "../../tests/utils";
import { ToastProvider } from "../../components/toast/ToastContext";
import * as api from "../../api/taskSavedSearches";

// The menu hits these via useTaskSavedSearches; mock the whole module.
vi.mock("../../api/taskSavedSearches", () => ({
  fetchTaskSavedSearches: vi.fn(),
  createTaskSavedSearch: vi.fn(),
  updateTaskSavedSearch: vi.fn(),
  deleteTaskSavedSearch: vi.fn(),
}));

describe("TaskPage saved-search save (integration)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.fetchTaskSavedSearches as Mock).mockResolvedValue([]);
    (api.createTaskSavedSearch as Mock).mockResolvedValue({ id: 99 });
    // happy-dom reports a desktop width, so TaskPage renders the desktop layout.
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 1280,
    });
  });

  afterEach(() => cleanup());

  it("saves the filter text currently in the filter box", async () => {
    renderWithProviders(
      <ToastProvider>
        <TaskPage />
      </ToastProvider>,
    );

    // Type a real filter into the actual toolbar input.
    const filterInput = await screen.findByPlaceholderText(
      "Filter... try #tag or status:todo",
    );
    fireEvent.change(filterInput, { target: { value: "#work date:today" } });

    // Open the saved-searches menu and save.
    fireEvent.click(screen.getByTitle("Saved searches"));
    fireEvent.click(await screen.findByText("Save current search"));
    const nameInput = await screen.findByPlaceholderText("Search name");
    fireEvent.change(nameInput, { target: { value: "Work today" } });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(api.createTaskSavedSearch).toHaveBeenCalled();
    });

    const sent = (api.createTaskSavedSearch as Mock).mock.calls[0][0];
    // The bug repro: filter_string must match what's in the box, not "".
    expect(sent.filter_string).toBe("#work date:today");
    expect(sent.name).toBe("Work today");
  });
});
