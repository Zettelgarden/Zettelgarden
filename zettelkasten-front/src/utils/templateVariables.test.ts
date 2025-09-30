import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { processTemplateVariables } from './templateVariables';

describe('processTemplateVariables', () => {
  beforeEach(() => {
    // Mock Date to get consistent results
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-03-15T14:30:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns original text when no variables present', () => {
    const text = 'Just plain text';
    expect(processTemplateVariables(text)).toBe('Just plain text');
  });

  it('replaces $date with current date', () => {
    const text = 'Today is $date';
    expect(processTemplateVariables(text)).toBe('Today is 2024-03-15');
  });

  it('replaces $time with current time', () => {
    const text = 'Time is $time';
    expect(processTemplateVariables(text)).toContain('14:30');
  });

  it('replaces $datetime with date and time', () => {
    const text = 'Now: $datetime';
    expect(processTemplateVariables(text)).toContain('2024-03-15');
  });

  it('replaces $day with day number', () => {
    const text = 'Day: $day';
    expect(processTemplateVariables(text)).toBe('Day: 15');
  });

  it('replaces $month with month number', () => {
    const text = 'Month: $month';
    expect(processTemplateVariables(text)).toBe('Month: 3');
  });

  it('replaces $year with year', () => {
    const text = 'Year: $year';
    expect(processTemplateVariables(text)).toBe('Year: 2024');
  });

  it('replaces $weekday with day name', () => {
    const text = 'Today is $weekday';
    expect(processTemplateVariables(text)).toBe('Today is Friday');
  });

  it('replaces multiple variables in one text', () => {
    const text = 'Date: $date, Time: $time, Year: $year';
    const result = processTemplateVariables(text);
    expect(result).toContain('2024-03-15');
    expect(result).toContain('14:30');
    expect(result).toContain('2024');
  });

  it('replaces multiple occurrences of the same variable', () => {
    const text = '$date and $date again';
    expect(processTemplateVariables(text)).toBe('2024-03-15 and 2024-03-15 again');
  });

  it('handles empty string', () => {
    expect(processTemplateVariables('')).toBe('');
  });

  it('handles text with no matches', () => {
    const text = 'No $fake or $invalid variables here';
    expect(processTemplateVariables(text)).toBe('No $fake or $invalid variables here');
  });
});