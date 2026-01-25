import { describe, it, expect } from "vitest";
import { preprocessSchemaTables } from "./CardBody";

describe("preprocessSchemaTables", () => {
  it("should convert basic schema reference to placeholder", () => {
    const input = "{{schema:1}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1&");
  });

  it("should convert schema reference with slug to placeholder", () => {
    const input = "{{schema:book-review}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:book-review&");
  });

  it("should handle schema_table alias", () => {
    const input = "{{schema_table:1}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1&");
  });

  it("should preserve backward compatible column shorthand", () => {
    const input = "{{schema:1|title,status}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1|title,status&");
  });

  it("should preserve explicit columns syntax", () => {
    const input = "{{schema:1|columns:title,status}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1|columns:title,status&");
  });

  it("should preserve filters syntax", () => {
    const input = "{{schema:1|filter:status=active}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1|filter:status=active&");
  });

  it("should preserve columns and filters together", () => {
    const input = "{{schema:1|columns:title,rating|filter:status=active,rating=gte:4}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1|columns:title,rating|filter:status=active,rating=gte:4&");
  });

  it("should not match spaces in schema ref", () => {
    // Spaces in the ref itself are invalid (IDs and slugs don't have spaces)
    const input = "{{schema: 1 }}";
    const output = preprocessSchemaTables(input);
    // The regex won't match, so output should remain unchanged
    expect(output).toBe(input);
  });

  it("should handle mixed case schema_table", () => {
    const input = "{{Schema_Table:1}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1&");
  });

  it("should handle multiple schema references in one text", () => {
    const input = "Check {{schema:1}} and {{schema:book-review|filter:status=active}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("Check &SCHEMATABLE:1& and &SCHEMATABLE:book-review|filter:status=active&");
  });

  it("should handle schema reference with slug containing hyphens", () => {
    const input = "{{schema:my-book-review}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:my-book-review&");
  });

  it("should handle column names with spaces in columns syntax", () => {
    const input = "{{schema:1|columns:Title,My Field,Status}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1|columns:Title,My Field,Status&");
  });

  it("should not interfere with other markdown content", () => {
    const input = "# Title\n\nSome text {{schema:1}} more text\n\n- list item";
    const output = preprocessSchemaTables(input);
    expect(output).toContain("&SCHEMATABLE:1&");
    expect(output).toContain("# Title");
    expect(output).toContain("Some text");
    expect(output).toContain("- list item");
  });

  it("should handle global insensitive matching", () => {
    const input = "{{schema:1}} and {{SCHEMA:2}} and {{Schema:3}}";
    const output = preprocessSchemaTables(input);
    expect(output).toBe("&SCHEMATABLE:1& and &SCHEMATABLE:2& and &SCHEMATABLE:3&");
  });
});
