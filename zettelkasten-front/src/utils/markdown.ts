import TurndownService from 'turndown';

// Create a singleton turndown service
const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
});

// Preserve links when converting
turndownService.addRule('links', {
  filter: 'a',
  replacement: (content: string, node: any) => {
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
  replacement: (content: string, node: any) => {
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
 * Converts HTML content to Markdown format
 * @param html The HTML string to convert
 * @returns The converted Markdown string
 */
export function htmlToMarkdown(html: string): string {
  if (!html) return '';
  return turndownService.turndown(html);
}

/**
 * Converts HTML content to Markdown, with a fallback to plain text
 * @param html The HTML string to convert
 * @returns The converted Markdown string, or plain text if conversion fails
 */
export function safeHtmlToMarkdown(html: string | null | undefined): string {
  if (!html) return '';
  try {
    return htmlToMarkdown(html);
  } catch (error) {
    console.error('Failed to convert HTML to markdown:', error);
    // Fallback: strip HTML tags and return plain text
    return html.replace(/<[^>]*>/g, '').trim();
  }
}
