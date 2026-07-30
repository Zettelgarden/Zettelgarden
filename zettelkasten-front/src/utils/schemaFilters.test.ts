import { describe, it, expect } from "vitest";
import {
  parseFilterValue,
  matchesFilter,
  parseFiltersString,
  applyFiltersToCard
} from "./schemaFilters";

describe("parseFilterValue", () => {
  it("should parse equality (default operator)", () => {
    expect(parseFilterValue("active")).toEqual({ operator: "eq", value: "active" });
    expect(parseFilterValue("pending")).toEqual({ operator: "eq", value: "pending" });
  });

  it("should parse greater than or equal", () => {
    expect(parseFilterValue("gte:5")).toEqual({ operator: "gte", value: "5" });
    expect(parseFilterValue("gte:10")).toEqual({ operator: "gte", value: "10" });
  });

  it("should parse less than or equal", () => {
    expect(parseFilterValue("lte:3")).toEqual({ operator: "lte", value: "3" });
    expect(parseFilterValue("lte:100")).toEqual({ operator: "lte", value: "100" });
  });

  it("should parse greater than", () => {
    expect(parseFilterValue("gt:5")).toEqual({ operator: "gt", value: "5" });
  });

  it("should parse less than", () => {
    expect(parseFilterValue("lt:10")).toEqual({ operator: "lt", value: "10" });
  });

  it("should parse not equals", () => {
    expect(parseFilterValue("ne:closed")).toEqual({ operator: "ne", value: "closed" });
  });

  it("should handle values with colons in them", () => {
    expect(parseFilterValue("gte:2024-01-01")).toEqual({ operator: "gte", value: "2024-01-01" });
  });
});

describe("matchesFilter", () => {
  describe("equality (default)", () => {
    it("should match string values exactly", () => {
      expect(matchesFilter("active", "active")).toBe(true);
      expect(matchesFilter("active", "inactive")).toBe(false);
    });

    it("should be case-insensitive", () => {
      expect(matchesFilter("Active", "active")).toBe(true);
      expect(matchesFilter("ACTIVE", "active")).toBe(true);
      expect(matchesFilter("active", "ACTIVE")).toBe(true);
    });

    it("should match numeric values as strings", () => {
      expect(matchesFilter("5", "5")).toBe(true);
      expect(matchesFilter(5, "5")).toBe(true);
      expect(matchesFilter("5", "10")).toBe(false);
    });

    it("should return false for null/undefined/empty", () => {
      expect(matchesFilter(null, "active")).toBe(false);
      expect(matchesFilter(undefined, "active")).toBe(false);
      expect(matchesFilter("", "active")).toBe(false);
    });
  });

  describe("not equals (ne)", () => {
    it("should return true when values don't match", () => {
      expect(matchesFilter("active", "ne:inactive")).toBe(true);
      expect(matchesFilter("active", "ne:closed")).toBe(true);
    });

    it("should return false when values match", () => {
      expect(matchesFilter("active", "ne:active")).toBe(false);
      expect(matchesFilter("Active", "ne:active")).toBe(false); // case-insensitive
    });

    it("should return false for null/undefined/empty", () => {
      expect(matchesFilter(null, "ne:active")).toBe(false);
      expect(matchesFilter(undefined, "ne:active")).toBe(false);
      expect(matchesFilter("", "ne:active")).toBe(false);
    });
  });

  describe("greater than (gt)", () => {
    it("should match when card value is greater", () => {
      expect(matchesFilter(5, "gt:3")).toBe(true);
      expect(matchesFilter(10, "gt:5")).toBe(true);
      expect(matchesFilter("10", "gt:5")).toBe(true);
    });

    it("should not match when card value is equal or less", () => {
      expect(matchesFilter(5, "gt:5")).toBe(false);
      expect(matchesFilter(3, "gt:5")).toBe(false);
    });

    it("should handle numeric strings", () => {
      expect(matchesFilter("7.5", "gt:5")).toBe(true);
      expect(matchesFilter("5", "gt:5")).toBe(false);
    });

    it("should return false for non-numeric values", () => {
      expect(matchesFilter("high", "gt:5")).toBe(false);
      expect(matchesFilter(null, "gt:5")).toBe(false);
    });
  });

  describe("greater than or equal (gte)", () => {
    it("should match when card value is greater or equal", () => {
      expect(matchesFilter(5, "gte:5")).toBe(true);
      expect(matchesFilter(10, "gte:5")).toBe(true);
      expect(matchesFilter("10", "gte:5")).toBe(true);
    });

    it("should not match when card value is less", () => {
      expect(matchesFilter(3, "gte:5")).toBe(false);
    });
  });

  describe("less than (lt)", () => {
    it("should match when card value is less", () => {
      expect(matchesFilter(3, "lt:5")).toBe(true);
      expect(matchesFilter(5, "lt:10")).toBe(true);
    });

    it("should not match when card value is equal or greater", () => {
      expect(matchesFilter(5, "lt:5")).toBe(false);
      expect(matchesFilter(10, "lt:5")).toBe(false);
    });
  });

  describe("less than or equal (lte)", () => {
    it("should match when card value is less or equal", () => {
      expect(matchesFilter(5, "lte:5")).toBe(true);
      expect(matchesFilter(3, "lte:5")).toBe(true);
    });

    it("should not match when card value is greater", () => {
      expect(matchesFilter(10, "lte:5")).toBe(false);
    });
  });

  describe("date comparisons", () => {
    it("should compare ISO dates chronologically (not as the leading year)", () => {
      expect(matchesFilter("2026-01-15", "gt:2026-01-01")).toBe(true);
      expect(matchesFilter("2026-01-01", "gt:2026-01-01")).toBe(false);
    });

    it("should support gte on dates", () => {
      expect(matchesFilter("2026-01-01", "gte:2026-01-01")).toBe(true);
      expect(matchesFilter("2025-12-31", "gte:2026-01-01")).toBe(false);
    });

    it("should support lt/lte on dates (before / on-or-before)", () => {
      expect(matchesFilter("2025-12-31", "lt:2026-01-01")).toBe(true);
      expect(matchesFilter("2026-01-15", "lt:2026-01-01")).toBe(false);
      expect(matchesFilter("2026-01-01", "lte:2026-01-01")).toBe(true);
    });

    it("should still use numeric comparison for bare numbers", () => {
      expect(matchesFilter(5, "gt:3")).toBe(true);
      expect(matchesFilter("7.5", "lt:5")).toBe(false);
    });

    it("should return false when the card value is not a date/number", () => {
      expect(matchesFilter("not-a-date", "gt:2026-01-01")).toBe(false);
    });
  });
});

describe("parseFiltersString", () => {
  it("should parse single filter", () => {
    expect(parseFiltersString("status=active")).toEqual({ status: "active" });
  });

  it("should parse multiple filters", () => {
    expect(parseFiltersString("status=active,priority=high"))
      .toEqual({ status: "active", priority: "high" });
  });

  it("should parse filters with spaces", () => {
    expect(parseFiltersString("status = active , priority = high"))
      .toEqual({ status: "active", priority: "high" });
  });

  it("should parse filters with operators", () => {
    expect(parseFiltersString("rating=gte:4,status=active"))
      .toEqual({ rating: "gte:4", status: "active" });
  });

  it("should handle empty string", () => {
    expect(parseFiltersString("")).toEqual({});
  });
});

describe("applyFiltersToCard", () => {
  const mockCard = {
    title: "Test Card",
    structured_data: {
      status: "active",
      priority: "high",
      rating: 5
    }
  };

  it("should match single filter on structured data", () => {
    expect(applyFiltersToCard(mockCard, { status: "active" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { status: "inactive" })).toBe(false);
  });

  it("should match filter on title field", () => {
    expect(applyFiltersToCard(mockCard, { title: "Test Card" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { title: "Other" })).toBe(false);
  });

  it("should match multiple filters (AND logic)", () => {
    expect(applyFiltersToCard(mockCard, {
      status: "active",
      priority: "high"
    })).toBe(true);

    expect(applyFiltersToCard(mockCard, {
      status: "active",
      priority: "low"
    })).toBe(false);
  });

  it("should work with numeric comparison operators", () => {
    expect(applyFiltersToCard(mockCard, { rating: "gte:5" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { rating: "gt:4" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { rating: "lte:5" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { rating: "lt:6" })).toBe(true);
    expect(applyFiltersToCard(mockCard, { rating: "gt:5" })).toBe(false);
  });

  it("should apply date comparison operators to ISO date fields", () => {
    const cardWithDate = {
      title: "Task",
      structured_data: { due_date: "2026-01-15" }
    };
    expect(applyFiltersToCard(cardWithDate, { due_date: "gt:2026-01-01" })).toBe(true);
    expect(applyFiltersToCard(cardWithDate, { due_date: "lt:2026-01-01" })).toBe(false);
    expect(applyFiltersToCard(cardWithDate, { due_date: "lte:2026-01-15" })).toBe(true);
  });

  it("should return true for empty filters", () => {
    expect(applyFiltersToCard(mockCard, {})).toBe(true);
  });

  it("should handle missing fields", () => {
    expect(applyFiltersToCard(mockCard, { missing: "value" })).toBe(false);
  });

  it("should handle cards without structured_data", () => {
    const cardWithoutData = { title: "Simple Card" };
    expect(applyFiltersToCard(cardWithoutData, { title: "Simple Card" })).toBe(true);
    expect(applyFiltersToCard(cardWithoutData, { status: "active" })).toBe(false);
  });
});
