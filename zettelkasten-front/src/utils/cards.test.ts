import { it, expect, test, describe } from "vitest";

import { sampleTasks, sampleCards } from "../tests/data";
import {
  convertCardToPartialCard,
  isCardIdUnique,
  sortCardIds,
  findNextChildId,
  buildCardFromParent,
  buildCardFromTreeNode,
} from "./cards";
import {
  Card,
  defaultPartialCard,
  ProcessedCardWithDescendants,
} from "../models/Card";

test("convert card to partial card", () => {
  let card = sampleCards()[0];
  let result = convertCardToPartialCard(card);
  expect(result["id"]).toBe(1);
});

test("isCardIdUnique", () => {
  expect(isCardIdUnique(sampleCards(), "3")).toBe(true);
  expect(isCardIdUnique(sampleCards(), "1")).toBe(false);
});

test("sort card ids", () => {
  let input = [
    "2/A.3/B",
    "10/A.2/B",
    "B10/B.5",
    "1/A.1/A",
    "3/B.1/C",
    "4/A.5/D",
    "2/A.10/A",
    "5/B.2/B",
    "A2/A.1",
    "3/A.6/A",
    "11/A.1/B",
    "1/B.1/A",
    "A1/A.10",
  ];
  let expectedOutput = [
    "1/A.1/A",
    "1/B.1/A",
    "2/A.3/B",
    "2/A.10/A",
    "3/A.6/A",
    "3/B.1/C",
    "4/A.5/D",
    "5/B.2/B",
    "10/A.2/B",
    "11/A.1/B",
    "A1/A.10",
    "A2/A.1",
    "B10/B.5",
  ];
  expect(sortCardIds(input)).toStrictEqual(expectedOutput);
});

describe("findNextChildId", () => {
  it("handles parent with no children", () => {
    expect(findNextChildId("A.1", [])).toBe("A.1.1");
  });

  it("handles parent with one child", () => {
    expect(findNextChildId("A.1", [{ card_id: "A.1.1", id: 1 } as any])).toBe(
      "A.1.2",
    );
  });

  it("increments existing number", () => {
    expect(findNextChildId("A.1", [{ card_id: "A.1.1", id: 1 } as any])).toBe(
      "A.1.2",
    );
  });

  it("handles multiple existing children", () => {
    const children = [
      { card_id: "A.1.1", id: 1 },
      { card_id: "A.1.2", id: 2 },
      { card_id: "A.1.3", id: 3 },
    ] as any[];
    expect(findNextChildId("A.1", children)).toBe("A.1.4");
  });

  it("handles complex parent ids", () => {
    expect(
      findNextChildId("SP104.6", [{ card_id: "SP104.6.1", id: 1 } as any]),
    ).toBe("SP104.6.2");
  });

  it("handles nested paths", () => {
    expect(
      findNextChildId("SP104.6.1", [{ card_id: "SP104.6.1.1", id: 1 } as any]),
    ).toBe("SP104.6.1.2");
  });

  it("handles real world example", () => {
    expect(findNextChildId("312", [{ card_id: "312.1", id: 1 } as any])).toBe(
      "312.2",
    );
  });

  it("handles big numbers", () => {
    const children = [
      { card_id: "A.1.1", id: 1 },
      { card_id: "A.1.2", id: 2 },
      { card_id: "A.1.3", id: 3 },
      { card_id: "A.1.4", id: 4 },
      { card_id: "A.1.5", id: 5 },
      { card_id: "A.1.6", id: 6 },
      { card_id: "A.1.7", id: 7 },
      { card_id: "A.1.8", id: 8 },
      { card_id: "A.1.9", id: 9 },
      { card_id: "A.1.10", id: 10 },
      { card_id: "A.1.11", id: 11 },
    ] as any[];
    expect(findNextChildId("A.1", children)).toBe("A.1.12");
  });

  it("handles children with non-matching prefixes", () => {
    const children = [
      { card_id: "A.1.1", id: 1 },
      { card_id: "A.2.1", id: 2 }, // Different parent
      { card_id: "A.1.2", id: 3 },
    ] as any[];
    expect(findNextChildId("A.1", children)).toBe("A.1.3");
  });
});

describe("buildCardFromParent", () => {
  it("builds a Card carrying over identifying fields and tags", () => {
    const parent = {
      id: 5,
      card_id: "2",
      user_id: 1,
      title: "Parent",
      parent_id: 1,
      created_at: new Date("2024-01-01"),
      updated_at: new Date("2024-01-02"),
      tags: [{ name: "x", id: 1, color: "#000", user_id: 1 }],
    } as any;

    const card = buildCardFromParent(parent);

    expect(card.id).toBe(5);
    expect(card.card_id).toBe("2");
    expect(card.title).toBe("Parent");
    expect(card.tags).toHaveLength(1);
    // Heavy fields are intentionally blank until the full fetch repopulates them
    expect(card.body).toBe("");
    expect(card.children).toEqual([]);
    expect(card.parent).toEqual(defaultPartialCard);
  });

  it("defaults empty title and missing tags", () => {
    const parent = {
      id: 5,
      card_id: "2",
      user_id: 1,
      title: "",
      parent_id: 1,
      created_at: new Date(0),
      updated_at: new Date(0),
    } as any;

    const card = buildCardFromParent(parent);

    expect(card.title).toBe("");
    expect(card.tags).toEqual([]);
  });
});

describe("buildCardFromTreeNode", () => {
  const parent = {
    id: 1,
    card_id: "1",
    user_id: 1,
    title: "Root",
    parent_id: -1,
    created_at: new Date(0),
    updated_at: new Date(0),
    tags: [],
  } as any;

  it("maps descendant nodes to minimal PartialCards", () => {
    const node: ProcessedCardWithDescendants = {
      id: 10,
      card_id: "1.1",
      user_id: 1,
      title: "Child",
      body: "b",
      link: "l",
      parent_id: 1,
      created_at: new Date(0),
      updated_at: new Date(0),
      depth: 1,
      descendants: [
        {
          id: 11,
          card_id: "1.1.1",
          user_id: 1,
          title: "Grandchild",
          parent_id: 10,
          created_at: new Date(0),
          updated_at: new Date(0),
          depth: 2,
          descendants: [],
        } as any,
      ],
    } as any;

    const card = buildCardFromTreeNode(node, parent);

    expect(card.id).toBe(10);
    expect(card.body).toBe("b");
    expect(card.parent).toBe(parent);
    expect(card.children).toHaveLength(1);
    expect(card.children[0].id).toBe(11);
    expect(card.children[0].tags).toEqual([]);
    expect(card.entities).toEqual([]);
  });

  it("handles a node without descendants", () => {
    const node = {
      id: 10,
      card_id: "1.1",
      user_id: 1,
      title: "Leaf",
      body: "",
      link: "",
      parent_id: 1,
      created_at: new Date(0),
      updated_at: new Date(0),
      depth: 1,
      descendants: [],
    } as any;

    const card = buildCardFromTreeNode(node, parent);

    expect(card.children).toEqual([]);
  });
});
