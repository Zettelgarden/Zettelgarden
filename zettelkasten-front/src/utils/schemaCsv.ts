import { Card } from '../models/Card';
import { FieldDefinition } from '../models/Schema';

/**
 * Serialize a single value as a CSV cell.
 *
 * - null/undefined become an empty cell
 * - arrays (multi-select) are joined with ", "
 * - values containing a comma, quote, or newline are wrapped in double
 *   quotes with embedded quotes doubled (RFC 4180)
 */
export function formatCsvValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  const str = Array.isArray(value)
    ? (value as unknown[]).join(', ')
    : String(value);
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
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
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
