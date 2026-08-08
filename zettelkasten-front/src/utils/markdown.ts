import TurndownService from 'turndown';

// Create a singleton turndown service
const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
});

/**
 * Strips script and style tags (and their content) from HTML
 * This is useful for cleaning up HTML emails before converting to markdown
 * @param html The HTML string to clean
 * @returns HTML with script and style tags removed
 */
function stripScriptsAndStyles(html: string): string {
  if (!html) return '';

  // Remove script tags and their content
  let cleaned = html.replace(
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    '',
  );

  // Remove style tags and their content
  cleaned = cleaned.replace(
    /<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi,
    '',
  );

  return cleaned;
}

// Preserve links when converting
turndownService.addRule('links', {
  filter: 'a',
  replacement: (content: string, node: HTMLElement) => {
    const href = node.getAttribute('href');
    if (href) {
      return `[${content}](${href})`;
    }
    return content;
  },
});

// Preserve images when converting
turndownService.addRule('images', {
  filter: 'img',
  replacement: (content: string, node: HTMLElement) => {
    const src = node.getAttribute('src');
    const alt = node.getAttribute('alt') || '';
    if (src) {
      return `![${alt}](${src})`;
    }
    return '';
  },
});

// Preserve bold and italic
turndownService.addRule('strong', {
  filter: ['strong', 'b'],
  replacement: (content: string) => `**${content}**`,
});

turndownService.addRule('em', {
  filter: ['em', 'i'],
  replacement: (content: string) => `*${content}*`,
});

/**
 * Detects if content appears to be HTML (as opposed to markdown or plain text)
 * @param content The content to check
 * @returns true if the content looks like HTML
 */
function isHtmlContent(content: string): boolean {
  // Check for common HTML tags
  const htmlTagPattern =
    /<\s*(?:p|div|span|h[1-6]|a|img|strong|em|b|i|ul|ol|li|blockquote|pre|code|br|hr)\b[^>]*>/i;
  return htmlTagPattern.test(content);
}

/**
 * Detects if content appears to be markdown
 * @param content The content to check
 * @returns true if the content looks like markdown
 */
function isMarkdownContent(content: string): boolean {
  // Check for markdown-specific patterns that aren't HTML
  const markdownPatterns = [
    /^\#{1,6}\s+/m, // Headings: #, ##, etc.
    /^\*\*.*?\*\*/m, // Bold: **text**
    /^\*.*?\*/m, // Italic: *text*
    /^\[.*?\]\(.*?\)/m, // Links: [text](url)
    /^\>.*$/m, // Blockquotes: > quote
    /^\`\`\`/m, // Code blocks: ```
    /^\s*[-*+]\s+/m, // Unordered lists: -, *, +
    /^\s*\d+\.\s+/m, // Ordered lists: 1.
  ];

  // Not markdown if it has HTML tags
  if (isHtmlContent(content)) {
    return false;
  }

  // Check if it has markdown patterns
  return markdownPatterns.some((pattern) => pattern.test(content));
}

/**
 * Converts HTML content to Markdown format
 * @param html The HTML string to convert
 * @returns The converted Markdown string
 */
export function htmlToMarkdown(html: string): string {
  if (!html) return '';
  // Strip scripts and styles before converting
  const cleaned = stripScriptsAndStyles(html);
  return turndownService.turndown(cleaned);
}

/**
 * Converts HTML content to Markdown, with a fallback to plain text
 * Also detects if content is already markdown and returns it as-is
 * @param content The content string to process (may be HTML, markdown, or plain text)
 * @returns The Markdown string
 */
export function safeHtmlToMarkdown(content: string | null | undefined): string {
  if (!content) return '';

  const trimmed = content.trim();

  // If content is already markdown, return as-is
  if (isMarkdownContent(trimmed)) {
    return trimmed;
  }

  // If content looks like HTML, convert it
  if (isHtmlContent(trimmed)) {
    try {
      return htmlToMarkdown(trimmed);
    } catch (error) {
      console.error('Failed to convert HTML to markdown:', error);
    }
  }

  // Otherwise return as plain text
  return trimmed;
}
