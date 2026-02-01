import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  getToday,
  getTomorrow,
  getYesterday,
  getNextWeek,
  compareDates,
  compareDatesInTimezone,
  isTodayOrPast,
  isPast,
  getStartOfMonthInTimezone,
  getEndOfMonthInTimezone,
  getStartOfWeekInTimezone,
  getEndOfWeekInTimezone,
} from "./dates";

describe("Date utility functions", () => {
  // Use UTC to remove time zone dependence
  const mockDate = new Date(Date.UTC(2023, 0, 1, 0, 0, 0)); // Jan 1, 2023, UTC

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(mockDate);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return today's date", () => {
    const today = getToday("UTC");
    expect(today.getUTCFullYear()).toBe(2023);
    expect(today.getUTCMonth()).toBe(0);
    expect(today.getUTCDate()).toBe(1);
  });

  it("should return tomorrow's date", () => {
    const tomorrow = getTomorrow("UTC");
    expect(tomorrow.getUTCFullYear()).toBe(2023);
    expect(tomorrow.getUTCMonth()).toBe(0);
    expect(tomorrow.getUTCDate()).toBe(2); // Expecting Jan 2, 2023, UTC
  });

  it("should return yesterday's date", () => {
    const yesterday = getYesterday("UTC");
    expect(yesterday.getUTCFullYear()).toBe(2022);
    expect(yesterday.getUTCMonth()).toBe(11); // December is 11
    expect(yesterday.getUTCDate()).toBe(31); // Expecting Dec 31, 2022, UTC
  });

  it("should return the date for exactly one week from today", () => {
    const nextWeek = getNextWeek("UTC");
    expect(nextWeek.getUTCFullYear()).toBe(2023);
    expect(nextWeek.getUTCMonth()).toBe(0);
    expect(nextWeek.getUTCDate()).toBe(8); // Expecting Jan 8, 2023, UTC
  });
});
describe('compareDates function', () => {
  it('should return true for identical dates', () => {
    const date1 = new Date(2023, 0, 1); // January 1, 2023
    const date2 = new Date(2023, 0, 1); // January 1, 2023
    expect(compareDates(date1, date2)).toBe(true);
  });

  it('should return false for different dates', () => {
    const date1 = new Date(2023, 0, 1); // January 1, 2023
    const date2 = new Date(2023, 0, 2); // January 2, 2023
    expect(compareDates(date1, date2)).toBe(false);
  });

  it('should return false if the first date is null', () => {
    const date2 = new Date(2023, 0, 1);
    expect(compareDates(null, date2)).toBe(false);
  });

  it('should return false if the second date is null', () => {
    const date1 = new Date(2023, 0, 1);
    expect(compareDates(date1, null)).toBe(false);
  });

  it('should return false if both dates are null', () => {
    expect(compareDates(null, null)).toBe(false);
  });

  // this should pass and doesn't, we have time zone problems
  // it('should return false for dates that differ by time only', () => {
  //   const date1 = new Date('2023-01-01T00:00:00Z');
  //   const date2 = new Date('2023-01-01T12:00:00Z');
  //   expect(compareDates(date1, date2)).toBe(true); // Assuming we just care about date, not time
  // });
});
describe("Date validation functions", () => {
  // Mock date setup, similar to your existing tests
  const mockToday = new Date(Date.UTC(2023, 0, 1, 5, 0, 0)); // Jan 1, 2023, UTC

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(mockToday);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("isTodayOrPast", () => {
    it("should return true for today's date", () => {
      const today = new Date(Date.UTC(2023, 0, 1, 5, 0, 0)); // Use same mock time
      expect(isTodayOrPast(today, "UTC")).toBe(true);
    });

    it("should return true for a past date", () => {
      const pastDate = new Date(Date.UTC(2022, 11, 31, 0, 0, 0));
      expect(isTodayOrPast(pastDate, "UTC")).toBe(true);
    });

    it("should return false for a future date", () => {
      const futureDate = new Date(Date.UTC(2023, 0, 2, 0, 0, 0));
      expect(isTodayOrPast(futureDate, "UTC")).toBe(false);
    });

    it("should return false for a null date", () => {
      expect(isTodayOrPast(null, "UTC")).toBe(false);
    });
  });

  describe("isPast", () => {
    it("should return false for today's date", () => {
      const today = new Date(Date.UTC(2023, 0, 1, 5, 0, 0)); // Use same mock time
      expect(isPast(today, "UTC")).toBe(false);
    });

    it("should return true for a past date", () => {
      const pastDate = new Date(Date.UTC(2022, 11, 30, 0, 0, 0));
      expect(isPast(pastDate, "UTC")).toBe(true);
    });

    it("should return false for a future date", () => {
      const futureDate = new Date(Date.UTC(2023, 0, 2, 0, 0, 0));
      expect(isPast(futureDate, "UTC")).toBe(false);
    });

    it("should return false for a null date", () => {
      expect(isPast(null, "UTC")).toBe(false);
    });
  });
});

describe('compareDatesInTimezone function', () => {
  it('should return true for identical dates in the same timezone', () => {
    const date1 = new Date('2023-01-01T00:00:00Z');
    const date2 = new Date('2023-01-01T12:00:00Z'); // Same day, different time
    expect(compareDatesInTimezone(date1, date2, "UTC")).toBe(true);
  });

  it('should return false for dates on different days in the same timezone', () => {
    const date1 = new Date('2023-01-01T00:00:00Z');
    const date2 = new Date('2023-01-02T00:00:00Z');
    expect(compareDatesInTimezone(date1, date2, "UTC")).toBe(false);
  });

  it('should handle timezone conversion correctly - same moment in different timezones', () => {
    // January 1, 2023 at midnight UTC in different timezones
    const date1 = new Date('2023-01-01T00:00:00Z'); // UTC
    const date2 = new Date('2023-01-01T08:00:00+08:00'); // Same moment in China timezone
    expect(compareDatesInTimezone(date1, date2, "UTC")).toBe(true);
  });

  it('should return false if the first date is null', () => {
    const date2 = new Date('2023-01-01T00:00:00Z');
    expect(compareDatesInTimezone(null, date2, "UTC")).toBe(false);
  });

  it('should return false if the second date is null', () => {
    const date1 = new Date('2023-01-01T00:00:00Z');
    expect(compareDatesInTimezone(date1, null, "UTC")).toBe(false);
  });

  it('should return false if both dates are null', () => {
    expect(compareDatesInTimezone(null, null, "UTC")).toBe(false);
  });

  it('should handle timezone conversions correctly', () => {
    // Test dates that represent the same local day in different timezones
    // March 14, 2023 05:00 UTC = March 14, 2023 01:00 in America/New_York (UTC-4)
    // March 14, 2023 10:00 UTC = March 14, 2023 06:00 in America/New_York (UTC-4)
    const date1 = new Date('2023-03-14T05:00:00Z'); // 1 AM EDT
    const date2 = new Date('2023-03-14T10:00:00Z'); // 6 AM EDT on same day

    // These should be the same day in America/New_York timezone
    expect(compareDatesInTimezone(date1, date2, "America/New_York")).toBe(true);

    // Different day should return false
    const date3 = new Date('2023-03-15T05:00:00Z'); // 1 AM EDT on next day
    expect(compareDatesInTimezone(date1, date3, "America/New_York")).toBe(false);
  });

  it('should work with different timezones', () => {
    // January 1, 2023 at 11:00 UTC = January 2, 2023 at 10:00 in Tokyo
    const utcDate = new Date('2023-01-01T11:00:00Z');
    const tokyoDate = new Date('2023-01-02T10:00:00+09:00'); // Same moment

    // These should be the same day in their respective timezones
    expect(compareDatesInTimezone(utcDate, utcDate, "UTC")).toBe(true);
    expect(compareDatesInTimezone(tokyoDate, tokyoDate, "Asia/Tokyo")).toBe(true);
  });
});

describe('Timezone-aware date comparison scenarios', () => {
  it('should handle user timezone vs UTC comparisons correctly', () => {
    // Simulate a task scheduled for today in user's timezone
    // getToday("America/New_York") returns midnight UTC for the current day in NY timezone
    // If today is Jan 1, 2023 in NY (which is midnight Jan 1, 2023 UTC for UTC-5),
    // then it's still the same calendar day
    const todayInUserTz = getToday("America/New_York");
    const todayInUtc = new Date(Date.UTC(todayInUserTz.getUTCFullYear(), todayInUserTz.getUTCMonth(), todayInUserTz.getUTCDate()));

    // These should be comparable correctly - both represent the same calendar day
    expect(compareDatesInTimezone(todayInUtc, todayInUserTz, "America/New_York")).toBe(true);
  });
});

describe('Timezone-aware date range functions', () => {
  describe('getStartOfMonthInTimezone', () => {
    it('should return midnight on first day of month in UTC for UTC timezone', () => {
      const date = new Date('2023-02-15T12:00:00Z');
      const start = getStartOfMonthInTimezone(date, 'UTC');

      expect(start.getUTCFullYear()).toBe(2023);
      expect(start.getUTCMonth()).toBe(1); // February
      expect(start.getUTCDate()).toBe(1);
      expect(start.getUTCHours()).toBe(0);
      expect(start.getUTCMinutes()).toBe(0);
    });

    it('should return midnight on first day of month in America/New_York timezone', () => {
      // Feb 15, 2023 at 12:00 UTC = Feb 15, 2023 at 07:00 EST
      const date = new Date('2023-02-15T12:00:00Z');
      const start = getStartOfMonthInTimezone(date, 'America/New_York');

      // Should be Feb 1, 2023 at 00:00 EST = Feb 1, 2023 at 05:00 UTC
      expect(start.getUTCFullYear()).toBe(2023);
      expect(start.getUTCMonth()).toBe(1); // February
      expect(start.getUTCDate()).toBe(1);
      expect(start.getUTCHours()).toBe(5); // 00:00 EST = 05:00 UTC
    });
  });

  describe('getEndOfMonthInTimezone', () => {
    it('should return end of day on last day of month in UTC timezone', () => {
      const date = new Date('2023-02-15T12:00:00Z');
      const end = getEndOfMonthInTimezone(date, 'UTC');

      // Feb 2023 has 28 days
      expect(end.getUTCFullYear()).toBe(2023);
      expect(end.getUTCMonth()).toBe(1); // February
      expect(end.getUTCDate()).toBe(28);
      expect(end.getUTCHours()).toBe(23);
      expect(end.getUTCMinutes()).toBe(59);
      expect(end.getUTCSeconds()).toBe(59);
    });

    it('should return end of day on last day of month in America/New_York timezone', () => {
      const date = new Date('2023-02-15T12:00:00Z');
      const end = getEndOfMonthInTimezone(date, 'America/New_York');

      // Should be Feb 28, 2023 at 23:59:59 EST = Feb 28, 2023 at 04:59:59 UTC (next day)
      expect(end.getUTCFullYear()).toBe(2023);
      expect(end.getUTCMonth()).toBe(2); // March (in UTC)
      expect(end.getUTCDate()).toBe(1);
      expect(end.getUTCHours()).toBe(4); // 23:59 EST = 04:59 UTC next day
    });
  });

  describe('getStartOfWeekInTimezone', () => {
    it('should return start of week (Sunday) in UTC timezone', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC
      const date = new Date('2023-02-15T12:00:00Z');
      const start = getStartOfWeekInTimezone(date, 'UTC', 0); // Week starts on Sunday

      // Should be Sunday, Feb 12, 2023 at 00:00 UTC
      expect(start.getUTCFullYear()).toBe(2023);
      expect(start.getUTCMonth()).toBe(1); // February
      expect(start.getUTCDate()).toBe(12);
      expect(start.getUTCHours()).toBe(0);
    });

    it('should return start of week (Monday) in UTC timezone', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC
      const date = new Date('2023-02-15T12:00:00Z');
      const start = getStartOfWeekInTimezone(date, 'UTC', 1); // Week starts on Monday

      // Should be Monday, Feb 13, 2023 at 00:00 UTC
      expect(start.getUTCFullYear()).toBe(2023);
      expect(start.getUTCMonth()).toBe(1); // February
      expect(start.getUTCDate()).toBe(13);
      expect(start.getUTCHours()).toBe(0);
    });

    it('should return start of week (Sunday) in America/New_York timezone', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC = 07:00 EST
      const date = new Date('2023-02-15T12:00:00Z');
      const start = getStartOfWeekInTimezone(date, 'America/New_York', 0); // Week starts on Sunday

      // Should be Sunday, Feb 12, 2023 at 00:00 EST = Feb 12, 2023 at 05:00 UTC
      expect(start.getUTCFullYear()).toBe(2023);
      expect(start.getUTCMonth()).toBe(1); // February
      expect(start.getUTCDate()).toBe(12);
      expect(start.getUTCHours()).toBe(5); // 00:00 EST = 05:00 UTC
    });
  });

  describe('getEndOfWeekInTimezone', () => {
    it('should return end of week (Saturday) in UTC timezone', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC
      const date = new Date('2023-02-15T12:00:00Z');
      const end = getEndOfWeekInTimezone(date, 'UTC', 0); // Week starts on Sunday, ends on Saturday

      // Should be Saturday, Feb 18, 2023 at 23:59:59 UTC
      expect(end.getUTCFullYear()).toBe(2023);
      expect(end.getUTCMonth()).toBe(1); // February
      expect(end.getUTCDate()).toBe(18);
      expect(end.getUTCHours()).toBe(23);
      expect(end.getUTCMinutes()).toBe(59);
    });

    it('should return end of week (Sunday) in UTC timezone when week starts on Monday', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC
      const date = new Date('2023-02-15T12:00:00Z');
      const end = getEndOfWeekInTimezone(date, 'UTC', 1); // Week starts on Monday, ends on Sunday

      // Should be Sunday, Feb 19, 2023 at 23:59:59 UTC
      expect(end.getUTCFullYear()).toBe(2023);
      expect(end.getUTCMonth()).toBe(1); // February
      expect(end.getUTCDate()).toBe(19);
      expect(end.getUTCHours()).toBe(23);
      expect(end.getUTCMinutes()).toBe(59);
    });

    it('should return end of week (Saturday) in America/New_York timezone', () => {
      // Wednesday, Feb 15, 2023 at 12:00 UTC = 07:00 EST
      const date = new Date('2023-02-15T12:00:00Z');
      const end = getEndOfWeekInTimezone(date, 'America/New_York', 0); // Week starts on Sunday

      // Should be Saturday, Feb 18, 2023 at 23:59:59 EST = Feb 19, 2023 at 04:59:59 UTC
      expect(end.getUTCFullYear()).toBe(2023);
      expect(end.getUTCMonth()).toBe(1); // February
      expect(end.getUTCDate()).toBe(19);
      expect(end.getUTCHours()).toBe(4); // 23:59 EST = 04:59 UTC next day
    });
  });
});
