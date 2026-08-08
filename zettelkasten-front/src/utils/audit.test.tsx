/**
 * Tests for audit formatting helpers (src/utils/audit.tsx)
 */

import { describe, it, expect } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import {
  formatFieldName,
  generateChangeSummary,
  groupEventsByDate,
  renderInlineDiff,
  renderLineDiff,
  renderAuditDiff,
  parseAuditEvent,
  getEventIcon,
  formatDate,
  AuditChange,
} from './audit';

describe('formatFieldName', () => {
  it('capitalizes the first letter', () => {
    expect(formatFieldName('title')).toBe('Title');
  });

  it('splits camelCase into words', () => {
    expect(formatFieldName('scheduledDate')).toBe('Scheduled Date');
  });

  it('handles already-capitalized input', () => {
    expect(formatFieldName('CardID')).toBe('Card I D');
  });
});

describe('generateChangeSummary', () => {
  const change = (field: string): AuditChange => ({
    field,
    from: 'old',
    to: 'new',
  });

  it('returns "Card created" for a create event with no changes', () => {
    expect(generateChangeSummary([], 'create')).toBe('Card created');
  });

  it('returns "Card deleted" for a delete event with no changes', () => {
    expect(generateChangeSummary([], 'DELETE')).toBe('Card deleted');
  });

  it('returns "No changes" for other events with no changes', () => {
    expect(generateChangeSummary([], 'update')).toBe('No changes');
    expect(generateChangeSummary([], 'unknown')).toBe('No changes');
  });

  it('summarizes a single changed field', () => {
    expect(generateChangeSummary([change('title')], 'update')).toBe(
      'Updated title',
    );
  });

  it('summarizes two changed fields with "and"', () => {
    expect(
      generateChangeSummary([change('title'), change('body')], 'update'),
    ).toBe('Updated title and body');
  });

  it('summarizes three or more fields with a count', () => {
    const changes = [change('a'), change('b'), change('c')];
    expect(generateChangeSummary(changes, 'update')).toBe('Updated 3 fields');
  });
});

describe('groupEventsByDate', () => {
  const dayMs = 24 * 60 * 60 * 1000;
  const now = new Date();

  function eventWithDate(daysAgo: number) {
    return {
      id: daysAgo,
      created_at: new Date(now.getTime() - daysAgo * dayMs),
    };
  }

  it('groups events into Today, Yesterday, This Week, and Older', () => {
    const groups = groupEventsByDate([
      eventWithDate(0), // today
      eventWithDate(1), // yesterday
      eventWithDate(3), // this week
      eventWithDate(30), // older
    ]);

    expect(groups.Today).toHaveLength(1);
    expect(groups.Yesterday).toHaveLength(1);
    expect(groups['This Week']).toHaveLength(1);
    expect(groups.Older).toHaveLength(1);
  });

  it('returns empty groups for no events', () => {
    const groups = groupEventsByDate([]);
    expect(groups.Today).toEqual([]);
    expect(groups.Yesterday).toEqual([]);
    expect(groups['This Week']).toEqual([]);
    expect(groups.Older).toEqual([]);
  });
});

describe('renderInlineDiff', () => {
  it('renders from and to values with truncation', () => {
    const html = renderToStaticMarkup(
      renderInlineDiff('old text', 'new text', 100) as any,
    );
    expect(html).toContain('old text');
    expect(html).toContain('new text');
  });

  it('truncates long values with an ellipsis', () => {
    const long = 'x'.repeat(500);
    const html = renderToStaticMarkup(
      renderInlineDiff(long, 'short', 50) as any,
    );
    expect(html).toContain('...');
    expect(html).not.toContain('x'.repeat(100));
  });

  it('renders "(empty)" for missing values', () => {
    const html = renderToStaticMarkup(
      renderInlineDiff('', null as any, 100) as any,
    );
    expect(html).toContain('(empty)');
  });
});

describe('renderLineDiff', () => {
  it('renders added and removed lines with +/- prefixes', () => {
    const html = renderToStaticMarkup(
      renderLineDiff('line one\nremoved line', 'line one\nadded line') as any,
    );
    expect(html).toContain('removed line');
    expect(html).toContain('added line');
    // context before the change is not rendered (trailing-context only)
    expect(html).not.toContain('line one');
  });

  it('renders an empty state for empty inputs without crashing', () => {
    const html = renderToStaticMarkup(renderLineDiff('', '') as any);
    expect(typeof html).toBe('string');
  });

  it('truncates output beyond maxLines', () => {
    // 100 removed + 100 added lines -> far beyond maxLines
    const a = Array.from({ length: 100 }, (_, i) => `a${i}`).join('\n');
    const b = Array.from({ length: 100 }, (_, i) => `b${i}`).join('\n');
    const html = renderToStaticMarkup(
      renderLineDiff(a, b, { maxLines: 10 }) as any,
    );
    expect(html).toContain('more lines');
  });
});

describe('renderAuditDiff', () => {
  it('uses line diff for the body field', () => {
    const html = renderToStaticMarkup(
      renderAuditDiff({ field: 'body', from: 'a\nb', to: 'a\nc' }) as any,
    );
    expect(html).toContain('Body');
    expect(html).toContain('b');
    expect(html).toContain('c');
  });

  it('shows "No visible changes" for identical body values', () => {
    const html = renderToStaticMarkup(
      renderAuditDiff({ field: 'body', from: 'same', to: 'same' }) as any,
    );
    expect(html).toContain('No visible changes');
  });

  it('renders short string fields with inline styling', () => {
    const html = renderToStaticMarkup(
      renderAuditDiff({ field: 'title', from: 'Old', to: 'New' }) as any,
    );
    expect(html).toContain('Title');
    expect(html).toContain('Old');
    expect(html).toContain('New');
  });

  it('renders non-string values as JSON', () => {
    const html = renderToStaticMarkup(
      renderAuditDiff({ field: 'count', from: 1, to: 2 }) as any,
    );
    expect(html).toContain('1');
    expect(html).toContain('2');
  });
});

describe('parseAuditEvent', () => {
  it('parses flat field changes', () => {
    const event = {
      details: {
        changes: {
          title: { from: 'Old', to: 'New' },
        },
      },
    };
    expect(parseAuditEvent(event)).toEqual([
      { field: 'title', from: 'Old', to: 'New' },
    ]);
  });

  it('parses nested field changes with dotted field names', () => {
    const event = {
      details: {
        changes: {
          metadata: {
            priority: { from: 1, to: 2 },
          },
        },
      },
    };
    expect(parseAuditEvent(event)).toEqual([
      { field: 'metadata.priority', from: 1, to: 2 },
    ]);
  });

  it('returns an empty array when there are no changes', () => {
    expect(parseAuditEvent({ details: {} })).toEqual([]);
    expect(parseAuditEvent({ details: { changes: null } })).toEqual([]);
    expect(parseAuditEvent({})).toEqual([]);
  });
});

describe('getEventIcon', () => {
  it('returns update/create/delete icons', () => {
    expect(renderToStaticMarkup(getEventIcon('update') as any)).toContain(
      'svg',
    );
    expect(renderToStaticMarkup(getEventIcon('CREATE') as any)).toContain(
      'svg',
    );
    expect(renderToStaticMarkup(getEventIcon('Delete') as any)).toContain(
      'svg',
    );
  });

  it('returns a default icon for unknown event types', () => {
    expect(renderToStaticMarkup(getEventIcon('mystery') as any)).toContain(
      'svg',
    );
  });
});

describe('formatDate', () => {
  it('formats a Date object', () => {
    const date = new Date('2024-03-15T10:30:00Z');
    const out = formatDate(date);
    expect(out).toContain('2024');
    expect(out).toContain('Mar');
  });

  it('formats an ISO string', () => {
    const out = formatDate('2024-03-15T10:30:00Z');
    expect(out).toContain('2024');
    expect(out).toContain('Mar');
  });
});
