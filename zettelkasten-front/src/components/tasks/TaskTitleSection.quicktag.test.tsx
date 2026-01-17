// @vitest-environment happy-dom

import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import { TagProvider } from "../../contexts/TagContext";
import { emptyTask } from "../../models/Task";
import type { Tag } from "../../models/Tags";
import { TaskTitleSection } from "./TaskTitleSection";

const testTags: Tag[] = [
  { id: 1, name: "work", color: "#000", user_id: 1 },
  { id: 2, name: "project-alpha", color: "#000", user_id: 1 },
];

function renderTitleSection(props: {
  mode: "create" | "edit";
  initialTitle: string;
}) {
  function Harness() {
    const [task, setTask] = React.useState({ ...emptyTask, title: props.initialTitle });

    return (
      <TagProvider testing testTags={testTags}>
        <TaskTitleSection
          task={task}
          setTask={setTask}
          mode={props.mode}
          isEditingTitle={true}
          setIsEditingTitle={() => undefined}
          showRecurringMenu={false}
          setShowRecurringMenu={() => undefined}
          onTitleSubmit={() => undefined}
          saveOnChange={true}
          onSaveTitle={vi.fn(async () => undefined)}
        />
      </TagProvider>
    );
  }

  render(<Harness />);

  return {
    input: screen.getByPlaceholderText("Enter task title") as HTMLInputElement,
  };
}

describe("TaskTitleSection quick-tag integration", () => {
  it("opens the popover when '#' is typed and closes it on Escape", () => {
    const { input } = renderTitleSection({ mode: "create", initialTitle: "" });

    fireEvent.change(input, { target: { value: "do #", selectionStart: 4 } });

    expect(screen.getByText("#work")).toBeInTheDocument();

    fireEvent.keyDown(input, { key: "Escape" });

    expect(screen.queryByText("#work")).not.toBeInTheDocument();
    expect(screen.queryByText("#project-alpha")).not.toBeInTheDocument();
  });

  it("filters suggestions as the user types", () => {
    const { input } = renderTitleSection({ mode: "create", initialTitle: "" });

    fireEvent.change(input, { target: { value: "do #", selectionStart: 4 } });
    expect(screen.getByText("#work")).toBeInTheDocument();
    expect(screen.getByText("#project-alpha")).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "do #pro", selectionStart: 6 } });

    expect(screen.getByText("#project-alpha")).toBeInTheDocument();
    expect(screen.queryByText("#work")).not.toBeInTheDocument();
  });

  it("shows suggestions and inserts selected tag (create mode)", () => {
    const { input } = renderTitleSection({ mode: "create", initialTitle: "" });

    fireEvent.change(input, { target: { value: "do #", selectionStart: 4 } });

    expect(screen.getByText("#work")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByText("#work"));

    expect(input.value).toBe("do #work ");
    expect(screen.queryByText("#project-alpha")).not.toBeInTheDocument();
  });

  it("prevents inserting a duplicate tag", () => {
    const { input } = renderTitleSection({ mode: "create", initialTitle: "do #work" });

    fireEvent.change(input, { target: { value: "do #work #", selectionStart: 10 } });

    expect(screen.getByText("#work")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByText("#work"));

    // Selection should close the popover but keep the title unchanged.
    expect(input.value).toBe("do #work #");
    expect(screen.queryByText("#work")).not.toBeInTheDocument();
  });

  it("works while editing in edit mode", () => {
    const { input } = renderTitleSection({ mode: "edit", initialTitle: "" });

    fireEvent.change(input, { target: { value: "#pro", selectionStart: 4 } });

    expect(screen.getByText("#project-alpha")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByText("#project-alpha"));

    expect(input.value).toBe("#project-alpha ");
  });
});
