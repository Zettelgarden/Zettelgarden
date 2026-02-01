import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { isToday, isPast } from "./CalendarEvent";

describe("CalendarEvent timezone functions", () => {
  // Mock current time as Jan 15, 2025 at 20:00 UTC (8 PM UTC)
  // This is Jan 15, 2025 at 15:00 (3 PM) in America/New_York (EST, UTC-5)
  // This is Jan 16, 2025 at 05:00 (5 AM) in Asia/Tokyo (JST, UTC+9)
  const mockNow = new Date(Date.UTC(2025, 0, 15, 20, 0, 0));

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(mockNow);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("isToday", () => {
    it("should correctly identify today in UTC timezone", () => {
      const todayUtc = new Date(Date.UTC(2025, 0, 15, 12, 0, 0));
      expect(isToday(todayUtc, "UTC")).toBe(true);
    });

    it("should correctly identify yesterday as not today in UTC timezone", () => {
      const yesterdayUtc = new Date(Date.UTC(2025, 0, 14, 12, 0, 0));
      expect(isToday(yesterdayUtc, "UTC")).toBe(false);
    });

    it("should correctly identify today in America/New_York timezone", () => {
      // Jan 15, 2025 at 20:00 UTC = Jan 15, 2025 at 15:00 EST (3 PM)
      const todayInNy = new Date(Date.UTC(2025, 0, 15, 20, 0, 0));
      expect(isToday(todayInNy, "America/New_York")).toBe(true);
    });

    it("should correctly identify a date that is tomorrow in UTC but today in America/New_York", () => {
      // Jan 16, 2025 at 02:00 UTC = Jan 15, 2025 at 21:00 EST (9 PM previous day)
      // So in NY timezone, this is still "today" (Jan 15)
      const tomorrowUtcButTodayNy = new Date(Date.UTC(2025, 0, 16, 2, 0, 0));
      expect(isToday(tomorrowUtcButTodayNy, "America/New_York")).toBe(true);
    });

    it("should correctly identify today in Asia/Tokyo timezone", () => {
      // Mock time: Jan 15, 2025 at 20:00 UTC
      // In Tokyo (UTC+9): Jan 16, 2025 at 05:00 JST
      // So "today" in Tokyo is Jan 16, 2025
      // Jan 16 at 00:00 UTC = Jan 16 at 09:00 JST (today in Tokyo)
      const todayInTokyo = new Date(Date.UTC(2025, 0, 16, 0, 0, 0));
      expect(isToday(todayInTokyo, "Asia/Tokyo")).toBe(true);
    });

    it("should correctly identify a date that is today in UTC but yesterday in Asia/Tokyo", () => {
      // Mock time: Jan 15, 2025 at 20:00 UTC
      // In Tokyo (UTC+9): Jan 16, 2025 at 05:00 JST
      // Jan 15 at 14:00 UTC = Jan 15 at 23:00 JST (yesterday in Tokyo)
      // So this should NOT be "today" in Tokyo
      const todayUtcButYesterdayTokyo = new Date(Date.UTC(2025, 0, 15, 14, 0, 0));
      expect(isToday(todayUtcButYesterdayTokyo, "Asia/Tokyo")).toBe(false);
    });
  });

  describe("isPast", () => {
    it("should correctly identify yesterday as past in UTC timezone", () => {
      const yesterdayUtc = new Date(Date.UTC(2025, 0, 14, 12, 0, 0));
      expect(isPast(yesterdayUtc, "UTC")).toBe(true);
    });

    it("should correctly identify today as not past in UTC timezone", () => {
      const todayUtc = new Date(Date.UTC(2025, 0, 15, 12, 0, 0));
      expect(isPast(todayUtc, "UTC")).toBe(false);
    });

    it("should correctly identify tomorrow as not past in UTC timezone", () => {
      const tomorrowUtc = new Date(Date.UTC(2025, 0, 16, 12, 0, 0));
      expect(isPast(tomorrowUtc, "UTC")).toBe(false);
    });

    it("should correctly identify a date that is today in UTC but past in America/New_York", () => {
      // Jan 15, 2025 at 04:00 UTC = Jan 15, 2025 at 23:00 EST (11 PM on Jan 14 in NY)
      // So in NY timezone, this date is in the past (it's yesterday)
      // Wait, let me recalculate: 04:00 UTC = 04:00 - 5 = 23:00 on Jan 14 EST
      // Current time is 20:00 UTC = 15:00 EST on Jan 15
      // So Jan 15 at 04:00 UTC = Jan 14 at 23:00 EST, which is in the past
      const todayUtcButPastNy = new Date(Date.UTC(2025, 0, 15, 4, 0, 0));
      expect(isPast(todayUtcButPastNy, "America/New_York")).toBe(true);
    });

    it("should correctly identify a date that is tomorrow UTC but today in America/New_York as not past", () => {
      // Jan 16, 2025 at 02:00 UTC = Jan 15, 2025 at 21:00 EST (still today in NY)
      // Current time is Jan 15 at 20:00 UTC = Jan 15 at 15:00 EST
      // So Jan 16 at 02:00 UTC = Jan 15 at 21:00 EST is in the future relative to 15:00 EST
      const tomorrowUtcButTodayNy = new Date(Date.UTC(2025, 0, 16, 2, 0, 0));
      expect(isPast(tomorrowUtcButTodayNy, "America/New_York")).toBe(false);
    });
  });
});
