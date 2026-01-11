import { renderHook, waitFor, act } from '@testing-library/react';
import { expect, test, describe, vi, beforeEach, Mock } from "vitest";
import { useEditedTask, useTaskSaving, useTaskLoading, useTaskDialog } from "./useTaskDialog";
import { Task, TaskAuditEvent } from "../models/Task";
import { TaskStatus } from "../models/TaskStatus";
import { fetchTask, saveExistingTask, fetchTaskAuditEvents } from "../api/tasks";

// Mock contexts
const mockUseTaskContext = vi.fn();
const mockUseStatus = vi.fn();

vi.mock("../contexts/TaskContext", () => ({
  useTaskContext: () => mockUseTaskContext(),
}));

vi.mock("../contexts/StatusContext", () => ({
  useStatus: () => mockUseStatus(),
}));

// Mock API functions
vi.mock("../api/tasks", () => ({
  fetchTask: vi.fn(),
  saveExistingTask: vi.fn(),
  fetchTaskAuditEvents: vi.fn(),
}));

const { fetchTask: mockFetchTask, saveExistingTask: mockSaveExistingTask, fetchTaskAuditEvents: mockFetchTaskAuditEvents } = await import("../api/tasks");

// Test data helpers
const createTestTask = (overrides: Partial<Task> = {}): Task => ({
  id: 1,
  card_pk: 0,
  user_id: 1,
  scheduled_date: new Date("2026-01-10"),
  due_date: null,
  created_at: new Date("2026-01-10"),
  updated_at: new Date("2026-01-10"),
  completed_at: null,
  title: "Test task",
  description: "Test description",
  priority: "A",
  status: 'todo',
  is_complete: false,
  is_deleted: false,
  reminder_time: null,
  reminder_sent: false,
  card: null,
  tags: [],
  blocked_by: [],
  blocks: [],
  ...overrides
});

const createTestAuditEvents = (): TaskAuditEvent[] => [
  {
    id: 1,
    user_id: 1,
    entity_id: 1,
    entity_type: "task",
    action: "create",
    details: { change_type: "create", changes: {} },
    created_at: new Date("2026-01-10"),
  }
];

const createTestStatus = (overrides: Partial<TaskStatus> = {}): TaskStatus => ({
  id: 1,
  user_id: 1,
  name: "todo",
  display_name: "To Do",
  color: "#6B7280",
  icon: "⭕",
  position: 0,
  is_default: true,
  is_complete_state: false,
  created_at: new Date("2026-01-10"),
  updated_at: new Date("2026-01-10"),
  ...overrides
});

describe("useEditedTask", () => {
  test("initializes with null task", () => {
    const { result } = renderHook(() => useEditedTask(null));

    expect(result.current.editedTask).toBeNull();
  });

  test("initializes with provided task", () => {
    const initialTask = createTestTask();
    const { result } = renderHook(() => useEditedTask(initialTask));

    expect(result.current.editedTask).toEqual(initialTask);
  });

  test("updates editedTask when initial task changes", () => {
    const initialTask = createTestTask({ title: "Initial" });
    const { result, rerender } = renderHook(
      ({ task }) => useEditedTask(task),
      { initialProps: { task: initialTask } }
    );

    expect(result.current.editedTask?.title).toBe("Initial");

    const updatedTask = createTestTask({ title: "Updated" });
    rerender({ task: updatedTask });

    expect(result.current.editedTask?.title).toBe("Updated");
  });

  test("updateEditedTask applies partial updates", async () => {
    const initialTask = createTestTask();
    const { result } = renderHook(() => useEditedTask(initialTask));

    // Wait for initial state to be set
    await waitFor(() => {
      expect(result.current.editedTask).toBeTruthy();
    });

    act(() => {
      result.current.updateEditedTask({ title: "Updated title" });
    });

    await waitFor(() => {
      expect(result.current.editedTask?.title).toBe("Updated title");
    });
    expect(result.current.editedTask?.description).toBe("Test description");
  });

  test("does not update when task is null", () => {
    const { result } = renderHook(() => useEditedTask(null));

    result.current.updateEditedTask({ title: "New title" });

    expect(result.current.editedTask).toBeNull();
  });
});

describe("useTaskSaving", () => {
  beforeEach(() => {
    mockUseTaskContext.mockReturnValue({
      setRefreshTasks: vi.fn(),
    });
    vi.clearAllMocks();
  });

  test("initial state", () => {
    const { result } = renderHook(() => useTaskSaving());

    expect(result.current.isSaving).toBe(false);
    expect(result.current.saveError).toBeNull();
  });

  test("successful save", async () => {
    const savedTask = createTestTask({ title: "Saved" });
    const mockSetRefreshTasks = vi.fn();
    mockUseTaskContext.mockReturnValue({
      setRefreshTasks: mockSetRefreshTasks,
    });
    (mockSaveExistingTask as Mock).mockResolvedValue(savedTask);

    const { result } = renderHook(() => useTaskSaving());
    const saveResult = await result.current.saveTask(createTestTask());

    expect(result.current.isSaving).toBe(false);
    expect(result.current.saveError).toBeNull();
    expect(saveResult).toEqual({ success: true, task: savedTask });
    expect(mockSetRefreshTasks).toHaveBeenCalledWith(true);
  });

  test("failed save", async () => {
    const mockSetRefreshTasks = vi.fn();
    mockUseTaskContext.mockReturnValue({
      setRefreshTasks: mockSetRefreshTasks,
    });
    (mockSaveExistingTask as Mock).mockRejectedValue(new Error("Save failed"));

    const { result } = renderHook(() => useTaskSaving());
    const saveResult = await result.current.saveTask(createTestTask());

    await waitFor(() => {
      expect(result.current.isSaving).toBe(false);
    });
    await waitFor(() => {
      expect(result.current.saveError).toBe("Save failed");
    });
    expect(saveResult).toEqual({ success: false, error: "Save failed" });
    expect(mockSetRefreshTasks).not.toHaveBeenCalled();
  });

  test("API error response", async () => {
    const mockSetRefreshTasks = vi.fn();
    mockUseTaskContext.mockReturnValue({
      setRefreshTasks: mockSetRefreshTasks,
    });
    (mockSaveExistingTask as Mock).mockResolvedValue({ error: "API error" });

    const { result } = renderHook(() => useTaskSaving());
    const saveResult = await result.current.saveTask(createTestTask());

    await waitFor(() => {
      expect(result.current.isSaving).toBe(false);
    });
    await waitFor(() => {
      expect(result.current.saveError).toBe("Failed to save task");
    });
    expect(saveResult).toEqual({ success: false, error: "Failed to save task" });
  });

  test("clearSaveError resets error", async () => {
    (mockSaveExistingTask as Mock).mockRejectedValue(new Error("Test error"));

    const { result } = renderHook(() => useTaskSaving());
    await result.current.saveTask(createTestTask());

    await waitFor(() => {
      expect(result.current.saveError).toBe("Test error");
    });

    result.current.clearSaveError();
    await waitFor(() => {
      expect(result.current.saveError).toBeNull();
    });
  });
});

describe("useTaskLoading", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("initial state when not open or no taskId", () => {
    const { result } = renderHook(() => useTaskLoading(null, false));

    expect(result.current.loadedTask).toBeNull();
    expect(result.current.auditEvents).toEqual([]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.loadingError).toBeNull();
  });

  test("loads task and audit events when open with taskId", async () => {
    const task = createTestTask();
    const auditEvents = createTestAuditEvents();

    (mockFetchTask as Mock).mockResolvedValue(task);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue(auditEvents);

    const { result } = renderHook(() => useTaskLoading(1, true));

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.loadedTask).toEqual(task);
    expect(result.current.auditEvents).toEqual(auditEvents);
    expect(result.current.loadingError).toBeNull();
  });

  test("handles load error", async () => {
    (mockFetchTask as Mock).mockRejectedValue(new Error("Load failed"));

    const { result } = renderHook(() => useTaskLoading(1, true));

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.loadedTask).toBeNull();
    expect(result.current.auditEvents).toEqual([]);
    expect(result.current.loadingError).toBe("Load failed");
  });

  test("resets state when dialog closes", async () => {
    const task = createTestTask();
    (mockFetchTask as Mock).mockResolvedValue(task);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue([]);

    const { result, rerender } = renderHook(
      ({ taskId, isOpen }) => useTaskLoading(taskId, isOpen),
      { initialProps: { taskId: 1 as number | null, isOpen: true } }
    );

    await waitFor(() => {
      expect(result.current.loadedTask).toEqual(task);
    });

    // Close dialog
    rerender({ taskId: null as any, isOpen: false });
    expect(result.current.loadedTask).toBeNull();
    expect(result.current.auditEvents).toEqual([]);
    expect(result.current.loadingError).toBeNull();
  });

  test("reloads when taskId changes", async () => {
    const task1 = createTestTask({ id: 1, title: "Task 1" });
    const task2 = createTestTask({ id: 2, title: "Task 2" });

    (mockFetchTask as Mock)
      .mockResolvedValueOnce(task1)
      .mockResolvedValueOnce(task2);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue([]);

    const { result, rerender } = renderHook(
      ({ taskId, isOpen }) => useTaskLoading(taskId, isOpen),
      { initialProps: { taskId: 1, isOpen: true } }
    );

    await waitFor(() => {
      expect(result.current.loadedTask?.title).toBe("Task 1");
    });

    rerender({ taskId: 2, isOpen: true });

    await waitFor(() => {
      expect(result.current.loadedTask?.title).toBe("Task 2");
    });
  });
});

describe("useTaskDialog", () => {
  beforeEach(() => {
    const mockSetRefreshTasks = vi.fn();
    mockUseTaskContext.mockReturnValue({
      setRefreshTasks: mockSetRefreshTasks,
    });

    const defaultStatus = createTestStatus({ name: "todo", is_default: true, is_complete_state: false });
    const completeStatus = createTestStatus({ name: "done", is_default: false, is_complete_state: true });

    mockUseStatus.mockReturnValue({
      getDefaultStatus: () => defaultStatus,
      getCompleteStatus: () => completeStatus,
    });

    vi.clearAllMocks();
  });

  test("combines all hook functionality", async () => {
    const task = createTestTask({ status: "todo", is_complete: false });
    const savedTask = createTestTask({ status: "done", is_complete: true });
    const auditEvents = createTestAuditEvents();

    (mockFetchTask as Mock).mockResolvedValue(task);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue(auditEvents);
    (mockSaveExistingTask as Mock).mockResolvedValue(savedTask);

    const { result } = renderHook(() => useTaskDialog(1, true));

    // Wait for loading to complete
    await waitFor(() => {
      expect(result.current.task).toEqual(task);
    });

    expect(result.current.auditEvents).toEqual(auditEvents);
    expect(result.current.editedTask).toEqual(task);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.isSaving).toBe(false);
    expect(result.current.saveError).toBeNull();

    // Test toggle complete functionality
    const toggleResult = await result.current.toggleComplete();
    expect(toggleResult.success).toBe(true);
    expect(result.current.isSaving).toBe(false);
  });

  test("saveEditedTask function works", async () => {
    const task = createTestTask();
    const savedTask = createTestTask({ title: "Updated" });

    (mockFetchTask as Mock).mockResolvedValue(task);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue([]);
    (mockSaveExistingTask as Mock).mockResolvedValue(savedTask);

    const { result } = renderHook(() => useTaskDialog(1, true));

    await waitFor(() => {
      expect(result.current.editedTask).toEqual(task);
    });

    // Update the task
    result.current.updateEditedTask({ title: "Updated" });

    // Save it
    const saveResult = await result.current.saveEditedTask();
    expect(saveResult.success).toBe(true);
    expect(saveResult.task).toEqual(savedTask);
  });

  test("handles missing status configuration", async () => {
    mockUseStatus.mockReturnValue({
      getDefaultStatus: () => undefined,
      getCompleteStatus: () => undefined,
    });

    const task = createTestTask();
    (mockFetchTask as Mock).mockResolvedValue(task);
    (mockFetchTaskAuditEvents as Mock).mockResolvedValue([]);

    const { result } = renderHook(() => useTaskDialog(1, true));

    await waitFor(() => {
      expect(result.current.task).toEqual(task);
    });

    // Toggle complete should fail gracefully
    const toggleResult = await result.current.toggleComplete();
    expect(toggleResult.success).toBe(false);
    expect(toggleResult.error).toContain("Could not find");
  });
});