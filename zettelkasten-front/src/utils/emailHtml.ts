import DOMPurify from "dompurify";

// Initialize DOMPurify hooks once
let isInitialized = false;

function initializePurify() {
  if (isInitialized) return;

  if (typeof window !== "undefined") {
    DOMPurify.addHook("uponSanitizeAttribute", (node, data) => {
      // Additional security hook for attributes
      if (data.attrName === "href" && data.attrValue?.toLowerCase().startsWith("javascript:")) {
        data.attrValue = "";
      }
    });
    isInitialized = true;
  }
}

// Initialize on import
initializePurify();

/**
 * DOMPurify configuration for sanitizing email HTML content.
 *
 * This configuration:
 * - Allows common email HTML tags and attributes
 * - Forces security attributes on links (rel="noopener noreferrer")
 * - Blocks dangerous content like scripts, data URIs, etc.
 * - Preserves safe content and formatting
 */
export const EMAIL_PURIFY_CONFIG = {
  // Allow tags commonly used in emails
  ALLOWED_TAGS: [
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'p', 'br', 'hr',
    'strong', 'b', 'em', 'i', 'u', 's', 'strike',
    'ul', 'ol', 'li',
    'blockquote', 'pre', 'code',
    'a', 'img',
    'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td',
    'div', 'span',
    'font', 'center',
    'sub', 'sup',
  ],
  // Allow safe attributes
  ALLOWED_ATTR: [
    'href', 'title', 'target', 'rel',
    'src', 'alt', 'width', 'height',
    'class', 'style', 'id',
    'colspan', 'rowspan',
    'cellpadding', 'cellspacing', 'border',
    'align', 'valign',
    'bgcolor', 'color',
    'face', 'size',
  ],
  // Add security attributes to links
  ADD_ATTR: ['target'],
  // Force rel="noopener noreferrer" on all links for security
  FORCE_SET_ATTR: {
    rel: 'noopener noreferrer'
  },
  // Disallow data URIs (security risk)
  ALLOW_DATA_ATTR: false,
  // Disallow unknown protocols (e.g., javascript:)
  ALLOW_UNKNOWN_PROTOCOLS: false,
  // Keep safe HTML entities and text content
  KEEP_CONTENT: true,
};

/**
 * Sanitizes email HTML content to prevent XSS attacks while
 * preserving safe formatting and structure.
 *
 * @param html - The raw HTML content from an email
 * @returns Sanitized HTML safe for rendering
 */
export function sanitizeEmailHtml(html: string): string {
  if (!html) return "";

  // Ensure DOMPurify is initialized
  initializePurify();

  // Workaround for test environment issue with script tags:
  // In some test environments (happy-dom), script tags can cause DOMPurify
  // to return empty string. We pre-remove script tags as a defense-in-depth
  // measure to avoid this issue. This is safe since script tags are not
  // in ALLOWED_TAGS and would be removed anyway.
  const htmlWithoutScripts = html.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');

  return DOMPurify.sanitize(htmlWithoutScripts, EMAIL_PURIFY_CONFIG);
}

/**
 * Post-processes sanitized HTML to ensure responsive behavior
 * and proper link security attributes.
 *
 * @param sanitizedHtml - Already sanitized HTML
 * @returns Processed HTML with responsive fixes
 */
export function postProcessEmailHtml(sanitizedHtml: string): string {
  if (!sanitizedHtml) return "";

  const parser = new DOMParser();
  const doc = parser.parseFromString(sanitizedHtml, "text/html");

  // Ensure all links have target="_blank" and rel="noopener noreferrer"
  const links = doc.querySelectorAll("a");
  links.forEach((link) => {
    link.setAttribute("target", "_blank");
    link.setAttribute("rel", "noopener noreferrer");
  });

  // Fix table overflow issues
  const tables = doc.querySelectorAll("table");
  tables.forEach((table) => {
    // Remove fixed widths that cause overflow
    table.removeAttribute("width");
    // Add responsive inline styles
    table.style.maxWidth = "100%";
    table.style.overflow = "hidden";
  });

  // Make images responsive
  const images = doc.querySelectorAll("img");
  images.forEach((img) => {
    img.style.maxWidth = "100%";
    img.style.height = "auto";
    img.style.display = "block";
  });

  // Fix nested divs that might have problematic inline styles
  const divs = doc.querySelectorAll("div[style*='font-size']");
  divs.forEach((div) => {
    // Remove absolute font sizes that break responsiveness
    (div as HTMLElement).style.fontSize = "inherit";
  });

  return doc.body.innerHTML;
}

/**
 * Processes raw email HTML through sanitization and post-processing.
 *
 * This is the main function to use for rendering email HTML content.
 * It:
 * 1. Sanitizes the HTML to remove malicious content
 * 2. Post-processes for responsive behavior
 * 3. Ensures all links open safely in new tabs
 *
 * @param rawHtml - The raw HTML content from an email
 * @returns Fully processed, safe, and responsive HTML
 */
export function processEmailHtml(rawHtml: string): string {
  if (!rawHtml) return "";

  const sanitized = sanitizeEmailHtml(rawHtml);
  return postProcessEmailHtml(sanitized);
}

/**
 * Extracts plain text from HTML as a fallback.
 *
 * @param html - HTML content
 * @returns Plain text representation
 */
export function extractTextFromHtml(html: string): string {
  if (!html) return "";

  const doc = new DOMParser().parseFromString(html, "text/html");
  return doc.body.textContent || "";
}
