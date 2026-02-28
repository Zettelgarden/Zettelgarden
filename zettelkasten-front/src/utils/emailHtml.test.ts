// @vitest-environment jsdom
import { describe, expect, it, beforeEach } from "vitest";
import {
  sanitizeEmailHtml,
  postProcessEmailHtml,
  processEmailHtml,
  extractTextFromHtml,
  EMAIL_PURIFY_CONFIG,
} from "./emailHtml";

// Initialize DOMPurify with happy-dom window
beforeEach(() => {
  // DOMPurify needs the window object to be available
  // In happy-dom environment, this is already set up by vitest
});

describe("emailHtml utilities", () => {
  describe("sanitizeEmailHtml", () => {
    it("removes script tags", () => {
      const malicious = '<p>Hello</p><script>alert("XSS")</script>';
      const clean = sanitizeEmailHtml(malicious);
      expect(clean).not.toContain("<script>");
      expect(clean).toContain("Hello");
    });

    it("removes javascript: URLs from links", () => {
      const malicious = '<a href="javascript:alert(\'XSS\')">Click me</a>';
      const clean = sanitizeEmailHtml(malicious);
      expect(clean).not.toContain("javascript:");
    });

    it("preserves safe HTML structure", () => {
      const safe = `
        <h1>Heading</h1>
        <p>Paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
        <ul><li>List item</li></ul>
      `;
      const clean = sanitizeEmailHtml(safe);
      expect(clean).toContain("Heading");
      expect(clean).toContain("bold");
      expect(clean).toContain("italic");
      expect(clean).toContain("List item");
    });

    it("preserves tables", () => {
      const table = '<table><tr><th>Header</th></tr><tr><td>Data</td></tr></table>';
      const clean = sanitizeEmailHtml(table);
      expect(clean).toContain("<table");
      expect(clean).toContain("<th>");
      expect(clean).toContain("<td>");
      expect(clean).toContain("Header");
      expect(clean).toContain("Data");
    });

    it("preserves images with safe attributes", () => {
      const img = '<img src="https://example.com/image.jpg" alt="Test image">';
      const clean = sanitizeEmailHtml(img);
      expect(clean).toContain('src="https://example.com/image.jpg"');
      expect(clean).toContain('alt="Test image"');
    });

    it("removes dangerous iframe tags", () => {
      const malicious = '<iframe src="https://evil.com"></iframe><p>Content</p>';
      const clean = sanitizeEmailHtml(malicious);
      expect(clean).not.toContain("<iframe");
      expect(clean).toContain("Content");
    });

    it("removes data URIs for security", () => {
      const malicious = '<a href="data:text/html,<script>alert(1)</script>">Click</a>';
      const clean = sanitizeEmailHtml(malicious);
      expect(clean).not.toContain("data:");
    });
  });

  describe("postProcessEmailHtml", () => {
    it("adds target blank and rel noopener to links", () => {
      const html = '<a href="https://example.com">Link</a>';
      const processed = postProcessEmailHtml(html);
      expect(processed).toContain('target="_blank"');
      expect(processed).toContain('rel="noopener noreferrer"');
    });

    it("removes fixed width from tables", () => {
      const html = '<table width="1000"><tr><td>Content</td></tr></table>';
      const processed = postProcessEmailHtml(html);
      expect(processed).not.toContain('width="1000"');
      expect(processed).toContain('max-width: 100%');
    });

    it("makes images responsive", () => {
      const html = '<img src="https://example.com/image.jpg" width="1000" height="500">';
      const processed = postProcessEmailHtml(html);
      expect(processed).toContain('max-width: 100%');
      expect(processed).toContain('height: auto');
    });

    it("handles multiple links", () => {
      const html = '<a href="https://one.com">One</a> <a href="https://two.com">Two</a>';
      const processed = postProcessEmailHtml(html);
      const linkCount = (processed.match(/target="_blank"/g) || []).length;
      expect(linkCount).toBe(2);
    });
  });

  describe("processEmailHtml", () => {
    it("sanitizes and post-processes HTML", () => {
      const html = '<p>Hello</p><script>alert(1)</script><a href="https://example.com">Link</a>';
      const processed = processEmailHtml(html);
      expect(processed).not.toContain("<script>");
      expect(processed).toContain("Hello");
      expect(processed).toContain('target="_blank"');
      expect(processed).toContain('rel="noopener noreferrer"');
    });

    it("handles complex email HTML", () => {
      const complexHtml = `
        <h1>Email Subject</h1>
        <p>Dear <strong>User</strong>,</p>
        <p>Please click <a href="https://example.com">here</a> to continue.</p>
        <table>
          <tr><th>Item</th><th>Price</th></tr>
          <tr><td>Widget</td><td>$10</td></tr>
        </table>
        <img src="https://example.com/logo.png" alt="Logo">
      `;
      const processed = processEmailHtml(complexHtml);
      expect(processed).toContain("Email Subject");
      expect(processed).toContain("User");
      expect(processed).toContain("Item");
      expect(processed).toContain("Price");
      expect(processed).toContain('target="_blank"');
    });

    it("returns empty string for empty input", () => {
      expect(processEmailHtml("")).toBe("");
      expect(processEmailHtml("   ")).toBe("");
    });
  });

  describe("extractTextFromHtml", () => {
    it("extracts plain text from HTML", () => {
      const html = '<p>Hello <strong>world</strong>!</p>';
      const text = extractTextFromHtml(html);
      expect(text).toBe("Hello world!");
    });

    it("handles complex HTML structure", () => {
      const html = `
        <h1>Title</h1>
        <ul><li>Item 1</li><li>Item 2</li></ul>
        <p>Paragraph</p>
      `;
      const text = extractTextFromHtml(html);
      expect(text).toContain("Title");
      expect(text).toContain("Item 1");
      expect(text).toContain("Item 2");
      expect(text).toContain("Paragraph");
    });

    it("returns empty string for empty input", () => {
      expect(extractTextFromHtml("")).toBe("");
    });
  });

  describe("EMAIL_PURIFY_CONFIG", () => {
    it("has safe allowed tags", () => {
      expect(EMAIL_PURIFY_CONFIG.ALLOWED_TAGS).toContain('p');
      expect(EMAIL_PURIFY_CONFIG.ALLOWED_TAGS).toContain('a');
      expect(EMAIL_PURIFY_CONFIG.ALLOWED_TAGS).toContain('table');
      expect(EMAIL_PURIFY_CONFIG.ALLOWED_TAGS).not.toContain('script');
      expect(EMAIL_PURIFY_CONFIG.ALLOWED_TAGS).not.toContain('iframe');
    });

    it("has security attributes configured", () => {
      expect(EMAIL_PURIFY_CONFIG.ALLOW_DATA_ATTR).toBe(false);
      expect(EMAIL_PURIFY_CONFIG.ALLOW_UNKNOWN_PROTOCOLS).toBe(false);
      expect(EMAIL_PURIFY_CONFIG.KEEP_CONTENT).toBe(true);
      expect(EMAIL_PURIFY_CONFIG.FORCE_SET_ATTR).toHaveProperty('rel');
    });
  });
});
