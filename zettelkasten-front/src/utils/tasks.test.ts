import { expect, test } from "vitest";
import { removeTagsFromTitle, parseTags, filterTasks, parseTaskQuery } from "./tasks";
import { sampleTaskData } from "../tests/data";
import { Task } from "../models/Task";

test("remove tags from title", () => {
  let title = "This is a #test title with #tags";
  expect(removeTagsFromTitle(title)).toBe("This is a  title with");
});

test("remove no tags from title", () => {
  const title = "This is a title without tags";
  expect(removeTagsFromTitle(title)).toBe("This is a title without tags");
});

test("parse tags from title with multiple tags", () => {
  const title = "This is a #test title with #multiple #tags";
  expect(parseTags(title)).toEqual(["#test", "#multiple", "#tags"]);
});

test("parse tags from title with no tags", () => {
  const title = "This title has no tags";
  expect(parseTags(title)).toEqual([]);
});

test("parse tags from empty string", () => {
  const title = "";
  expect(parseTags(title)).toEqual([]);
});

test("parse tags from title with only tags", () => {
  const title = "#tag1 #tag2 #tag3";
  expect(parseTags(title)).toEqual(["#tag1", "#tag2", "#tag3"]);
});

test("parse tags from title with tags and punctuation", () => {
  const title = "This is a title with #tags, and punctuation!";
  expect(parseTags(title)).toEqual(["#tags,"]);
});

test("parse tags from title with mixed content", () => {
  const title = "Some text #tag1 more text #tag2";
  expect(parseTags(title)).toEqual(["#tag1", "#tag2"]);
});

test("parse tags from title with consecutive tags", () => {
  const title = "Some text #tag1#tag2 more text";
  expect(parseTags(title)).toEqual(["#tag1#tag2"]);
});

test("skip parsing tags from middle of words", () => {
  const title = "Some text#tag1#tag2 more text";
  expect(parseTags(title)).toEqual([]);
});

test("filter tasks by tags", () => {
  const results = filterTasks(sampleTaskData, "#work session");
  expect(results.length).toEqual(1);
});

test("filter tasks by negated tags", () => {
  const tasks: Task[] = [
    {
      id: 1,
      title: "Task 1",
      tags: [{ name: "work", id: 1, color: "#ff0000", user_id: 1 }],
      card_pk: 1,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    },
    {
      id: 2,
      title: "Task 2",
      tags: [{ name: "personal", id: 2, color: "#00ff00", user_id: 1 }],
      card_pk: 2,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    },
    {
      id: 3,
      title: "Task 3",
      tags: [
        { name: "work", id: 1, color: "#ff0000", user_id: 1 },
        { name: "personal", id: 2, color: "#00ff00", user_id: 1 }
      ],
      card_pk: 3,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    }
  ];

  const results1 = filterTasks(tasks, "!#work");
  expect(results1.length).toEqual(1);
  expect(results1[0].id).toEqual(2);

  const results2 = filterTasks(tasks, "#work !#personal");
  expect(results2.length).toEqual(1);
  expect(results2[0].id).toEqual(1);
});

test("filter tasks by negated text", () => {
  const tasks: Task[] = [
    {
      id: 1,
      title: "Meeting with team",
      tags: [],
      card_pk: 1,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    },
    {
      id: 2,
      title: "Write documentation",
      tags: [],
      card_pk: 2,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    },
    {
      id: 3,
      title: "Team lunch meeting",
      tags: [],
      card_pk: 3,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
    }
  ];

  const results1 = filterTasks(tasks, "!meeting");
  expect(results1.length).toEqual(1);
  expect(results1[0].id).toEqual(2);

  const results2 = filterTasks(tasks, "team !lunch");
  expect(results2.length).toEqual(1);
  expect(results2[0].id).toEqual(1);
});

test("parseTaskQuery extracts date:none", () => {
  const result = parseTaskQuery("date:none");
  expect(result.dateView).toEqual("no_date");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery extracts date:no_date", () => {
  const result = parseTaskQuery("date:no_date");
  expect(result.dateView).toEqual("no_date");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery extracts date:today", () => {
  const result = parseTaskQuery("date:today");
  expect(result.dateView).toEqual("today");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery extracts date:overdue", () => {
  const result = parseTaskQuery("date:overdue");
  expect(result.dateView).toEqual("overdue");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery extracts date:this_week", () => {
  const result = parseTaskQuery("date:this_week");
  expect(result.dateView).toEqual("this_week");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery extracts date:all", () => {
  const result = parseTaskQuery("date:all");
  expect(result.dateView).toEqual("all");
  expect(result.searchTerms).toEqual([]);
});

test("parseTaskQuery handles date filter with search terms", () => {
  const result = parseTaskQuery("date:none #work urgent");
  expect(result.dateView).toEqual("no_date");
  expect(result.searchTerms).toEqual(["#work", "urgent"]);
});
