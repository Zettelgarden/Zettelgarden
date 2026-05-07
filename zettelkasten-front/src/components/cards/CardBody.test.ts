import { describe, it, expect } from "vitest";
import { preprocessSchemaTables, preprocessWikiLinks } from "./CardBody";

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

describe("preprocessWikiLinks", () => {
  it("should convert basic wiki-link to markdown link", () => {
    const input = "See [[42]] for details";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [42](#card:42) for details");
  });

  it("should convert wiki-link with display text", () => {
    const input = "See [[42|Meeting Notes]] for details";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [Meeting Notes](#card:42) for details");
  });

  it("should not touch markdown links", () => {
    const input = "[click here](https://example.com)";
    const output = preprocessWikiLinks(input);
    expect(output).toBe(input);
  });

  it("should not touch single brackets", () => {
    const input = "[some regular text]";
    const output = preprocessWikiLinks(input);
    expect(output).toBe(input);
  });

  it("should handle child card IDs with dots", () => {
    const input = "Related to [[1.3]]";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("Related to [1.3](#card:1.3)");
  });

  it("should handle multiple wiki-links in one body", () => {
    const input = "See [[a]] and [[b|Second]] and [[c]]";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [a](#card:a) and [Second](#card:b) and [c](#card:c)");
  });

  it("should handle wiki-link with hyphens and underscores", () => {
    const input = "See [[my-note_v2]]";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [my-note_v2](#card:my-note_v2)");
  });

  it("should handle mixed wiki-links and markdown", () => {
    const input = "[[42]] and [click](https://example.com)";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("[42](#card:42) and [click](https://example.com)");
  });

  it("should handle wiki-link with trailing pipe [[card|title|]]", () => {
    const input = "See [[42|Meeting Notes|]] for details";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [Meeting Notes](#card:42) for details");
  });

  it("should handle wiki-link with empty display text and trailing pipe", () => {
    const input = "See [[42||]]";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [42](#card:42)");
  });

  it("should preserve * as display text for dynamic title resolution", () => {
    const input = "See [[42|*|]] for details";
    const output = preprocessWikiLinks(input);
    expect(output).toBe("See [*](#card:42) for details");
  });
});
