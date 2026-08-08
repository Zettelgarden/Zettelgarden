import { Card } from '../models/Card';
import { FieldDefinition } from '../models/Schema';

/**
 * Serialize a single value as a CSV cell.
 *
 * - null/undefined become an empty cell
 * - arrays (multi-select) are joined with ", "
 * - values containing a comma, quote, or newline are wrapped in double
 *   quotes with embedded quotes doubled (RFC 4180)
 * - values starting with a spreadsheet-formula metacharacter (=, +, -, @,
 *   tab) are neutralized with a leading apostrophe so Excel/LibreOffice
 *   treat them as text, not formulas (OWASP CSV Injection)
 */
export function formatCsvValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  const str = Array.isArray(value)
    ? (value as unknown[]).join(', ')
    : String(value);
  const guarded = /^[=+\-@\t]/.test(str) ? `'${str}` : str;
  if (/[",\n\r]/.test(guarded)) {
    return `"${guarded.replace(/"/g, '""')}"`;
  }
  return guarded;
}

/**
 * Serialize the given cards as CSV with columns: card_id, title, then one
 * column per schema field (in field order). Multi-select values are joined
 * with ", " and link_to_card values are exported as the raw card id.
 */
export function schemaCardsToCsv(
  cards: Card[],
  fields: FieldDefinition[],
): string {
  const columns = ['card_id', 'title', ...fields.map((f) => f.name)];
  const header = columns.map(formatCsvValue).join(',');

  const rows = cards.map((card) => {
    const cells = columns.map((column) => {
      if (column === 'card_id') return card.card_id;
      if (column === 'title') return card.title;
      return card.structured_data?.[column];
    });
    return cells.map(formatCsvValue).join(',');
  });

  return [header, ...rows].join('\n');
}

/**
 * Trigger a browser download of `csv` content as `filename`.
 */
export function downloadCsv(filename: string, csv: string): void {
  // UTF-8 BOM: without it Excel sniffs the file as ANSI and renders
  // non-ASCII titles/values as mojibake.
  const blob = new Blob(['\uFEFF' + csv], {
    type: 'text/csv;charset=utf-8',
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  // Defer revoke so the browser has finished reading the blob URL before it
  // is freed (avoids rare dropped downloads in some engines).
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
