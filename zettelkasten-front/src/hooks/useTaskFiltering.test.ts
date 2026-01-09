import { renderHook } from '@testing-library/react';
import { expect, test, describe } from "vitest";
import { useTaskFiltering } from "./useTaskFiltering";
import { Task } from "../models/Task";

// Helper to create a minimal task object
const createTask = (overrides: Partial<Task>): Task => ({
  id: 1,
  title: "Test task",
  tags: [],
  card_pk: 0,
  user_id: 1,
  scheduled_date: new Date("2026-01-08"),
  due_date: null,
  status: 'todo' as const,
  is_complete: false,
  created_at: new Date("2026-01-08"),
  updated_at: new Date("2026-01-08"),
  completed_at: null,
  is_deleted: false,
  priority: null,
  card: null,
  reminder_time: null,
  reminder_sent: false,
  ...overrides
});

describe("useTaskFiltering - sorting", () => {
  test("sorts by title alphabetically (ascending)", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Zebra task" }),
      createTask({ id: 2, title: "Apple task" }),
      createTask({ id: 3, title: "Mango task" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "title",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const titles = result.current.tasksToDisplay.map(t => t.title);
    expect(titles).toEqual(["Apple task", "Mango task", "Zebra task"]);
  });

  test("sorts by title alphabetically (descending)", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Zebra task" }),
      createTask({ id: 2, title: "Apple task" }),
      createTask({ id: 3, title: "Mango task" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "title",
        sortDirection: "desc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const titles = result.current.tasksToDisplay.map(t => t.title);
    expect(titles).toEqual(["Zebra task", "Mango task", "Apple task"]);
  });

  test("sorts by title with tags stripped before comparison", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "#urgent Zebra task" }),
      createTask({ id: 2, title: "Apple task #work" }),
      createTask({ id: 3, title: "#important Mango task #urgent" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "title",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const titles = result.current.tasksToDisplay.map(t => t.title);
    // Should sort as: "Apple task", "Mango task", "Zebra task" (without tags)
    expect(titles).toEqual(["Apple task #work", "#important Mango task #urgent", "#urgent Zebra task"]);
  });

  test("sorts by priority", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Task 1", priority: "C" }),
      createTask({ id: 2, title: "Task 2", priority: "A" }),
      createTask({ id: 3, title: "Task 3", priority: "B" }),
      createTask({ id: 4, title: "Task 4", priority: null }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "priority",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const priorities = result.current.tasksToDisplay.map(t => t.priority);
    // nulls should be last
    expect(priorities).toEqual(["A", "B", "C", null]);
  });

  test("sorts by scheduled_date", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Task 1", scheduled_date: new Date("2026-01-15") }),
      createTask({ id: 2, title: "Task 2", scheduled_date: new Date("2026-01-10") }),
      createTask({ id: 3, title: "Task 3", scheduled_date: null }),
      createTask({ id: 4, title: "Task 4", scheduled_date: new Date("2026-01-05") }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "scheduled_date",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const dates = result.current.tasksToDisplay.map(t => t.scheduled_date?.toISOString() ?? null);
    expect(dates).toEqual([
      new Date("2026-01-05").toISOString(),
      new Date("2026-01-10").toISOString(),
      new Date("2026-01-15").toISOString(),
      null
    ]);
  });

  test("sorts by id", () => {
    const tasks: Task[] = [
      createTask({ id: 3, title: "Task 3" }),
      createTask({ id: 1, title: "Task 1" }),
      createTask({ id: 2, title: "Task 2" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "id",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const ids = result.current.tasksToDisplay.map(t => t.id);
    expect(ids).toEqual([1, 2, 3]);
  });

  test("sorts by updated_at", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Task 1", updated_at: new Date("2026-01-15") }),
      createTask({ id: 2, title: "Task 2", updated_at: new Date("2026-01-10") }),
      createTask({ id: 3, title: "Task 3", updated_at: new Date("2026-01-20") }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "updated_at",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const updatedDates = result.current.tasksToDisplay.map(t => t.updated_at.toISOString());
    expect(updatedDates).toEqual([
      new Date("2026-01-10").toISOString(),
      new Date("2026-01-15").toISOString(),
      new Date("2026-01-20").toISOString(),
    ]);
  });

  test("applies sorting to kanban view", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Zebra task", status: "todo" }),
      createTask({ id: 2, title: "Apple task", status: "todo" }),
      createTask({ id: 3, title: "Mango task", status: "todo" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "title",
        sortDirection: "asc",
        viewMode: "kanban",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const titles = result.current.tasksToDisplay.map(t => t.title);
    // Sorting should apply to kanban view too
    expect(titles).toEqual(["Apple task", "Mango task", "Zebra task"]);
  });

  test("applies sorting to matrix view", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Zebra task" }),
      createTask({ id: 2, title: "Apple task" }),
      createTask({ id: 3, title: "Mango task" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "title",
        sortDirection: "asc",
        viewMode: "matrix",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    const titles = result.current.tasksToDisplay.map(t => t.title);
    // Sorting should apply to matrix view too
    expect(titles).toEqual(["Apple task", "Mango task", "Zebra task"]);
  });
});

describe("useTaskFiltering - filtering", () => {
  test("filters by date view - today", () => {
    // Use actual date functions to ensure proper filtering
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);
    const tomorrow = new Date();
    tomorrow.setDate(today.getDate() + 1);

    const tasks: Task[] = [
      createTask({ id: 1, title: "Today task", scheduled_date: today }),
      createTask({ id: 2, title: "Yesterday task", scheduled_date: yesterday }),
      createTask({ id: 3, title: "Tomorrow task", scheduled_date: tomorrow }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "today",
        showCompleted: false,
        filterString: "",
        sortField: "id",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    // Should include today and overdue (yesterday), but not tomorrow
    expect(result.current.tasksToDisplay.length).toBe(2);
    const titles = result.current.tasksToDisplay.map(t => t.title);
    expect(titles).toContain("Today task");
    expect(titles).toContain("Yesterday task");
    expect(titles).not.toContain("Tomorrow task");
  });

  test("filters by search string", () => {
    const tasks: Task[] = [
      createTask({ id: 1, title: "Buy groceries" }),
      createTask({ id: 2, title: "Write code" }),
      createTask({ id: 3, title: "Buy coffee" }),
    ];

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "buy",
        sortField: "id",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 25,
      })
    );

    expect(result.current.tasksToDisplay.length).toBe(2);
    const titles = result.current.tasksToDisplay.map(t => t.title);
    expect(titles).toEqual(["Buy groceries", "Buy coffee"]);
  });
});

describe("useTaskFiltering - pagination", () => {
  test("paginates tasks in list view", () => {
    const tasks: Task[] = Array.from({ length: 30 }, (_, i) =>
      createTask({ id: i + 1, title: `Task ${i + 1}` })
    );

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "id",
        sortDirection: "asc",
        viewMode: "list",
        currentPage: 1,
        itemsPerPage: 10,
      })
    );

    expect(result.current.tasksToDisplay.length).toBe(30);
    expect(result.current.paginatedTasks.length).toBe(10);
    expect(result.current.totalPages).toBe(3);
    expect(result.current.paginatedTasks[0].id).toBe(1);
    expect(result.current.paginatedTasks[9].id).toBe(10);
  });

  test("does not paginate kanban view", () => {
    const tasks: Task[] = Array.from({ length: 30 }, (_, i) =>
      createTask({ id: i + 1, title: `Task ${i + 1}` })
    );

    const { result } = renderHook(() =>
      useTaskFiltering({
        tasks,
        dateView: "all",
        showCompleted: true,
        filterString: "",
        sortField: "id",
        sortDirection: "asc",
        viewMode: "kanban",
        currentPage: 1,
        itemsPerPage: 10,
      })
    );

    // Kanban view should not paginate
    expect(result.current.paginatedTasks.length).toBe(30);
    expect(result.current.totalPages).toBe(1);
  });
});
