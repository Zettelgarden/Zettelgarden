import { Task } from "src/models/Task";
import { toZonedTime, fromZonedTime } from "date-fns-tz";

export function getToday(timezone: string = "UTC"): Date {
  // Get current time
  const now = new Date();

  if (timezone === "UTC") {
    // For UTC, return midnight UTC of the current UTC date
    return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 0, 0, 0, 0));
  }

  // For other timezones, convert to the user's timezone to get today's date components
  const nowInUserTz = toZonedTime(now, timezone);

  // Get the year, month, and date in the user's timezone
  const year = nowInUserTz.getFullYear();
  const month = nowInUserTz.getMonth();
  const date = nowInUserTz.getDate();

  // Return midnight today in the user's timezone (as a UTC Date object)
  return new Date(Date.UTC(year, month, date, 0, 0, 0, 0));
}

export function getTomorrow(timezone: string = "UTC"): Date {
  const today = getToday(timezone);
  return new Date(Date.UTC(
    today.getUTCFullYear(),
    today.getUTCMonth(),
    today.getUTCDate() + 1,
    0, 0, 0, 0
  ));
}

export function getYesterday(timezone: string = "UTC"): Date {
  const today = getToday(timezone);
  return new Date(Date.UTC(
    today.getUTCFullYear(),
    today.getUTCMonth(),
    today.getUTCDate() - 1,
    0, 0, 0, 0
  ));
}

export function getNextWeek(timezone: string = "UTC"): Date {
  const today = getToday(timezone);
  return new Date(Date.UTC(
    today.getUTCFullYear(),
    today.getUTCMonth(),
    today.getUTCDate() + 7,
    0, 0, 0, 0
  ));
}

export function isFriday(timezone: string = "UTC"): boolean {
  const today = getToday(timezone);
  const dayInTimezone = toZonedTime(today, timezone);
  return dayInTimezone.getDay() === 5; // 5 = Friday (0 = Sunday)
}

export function getNextMonday(timezone: string = "UTC"): Date {
  const today = getToday(timezone);
  // Convert to user's timezone to calculate properly
  const todayInTz = toZonedTime(today, timezone);
  const day = todayInTz.getDay();
  const diff = todayInTz.getDate() + (8 - day) % 7;
  const nextMondayInTz = new Date(todayInTz);
  nextMondayInTz.setDate(diff);
  // Convert back to UTC Date object
  return fromZonedTime(nextMondayInTz, timezone);
}
export function compareDates(date1: Date | null, date2: Date | null): boolean {
  if (date1 === null || date2 === null) {
    return false;
  }
  // Compare dates by checking if they fall on the same day
  return (
    date1.getDate() === date2.getDate() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getFullYear() === date2.getFullYear()
  );
}

/**
 * Compare two dates to see if they represent the same day in a specific timezone.
 * This is timezone-aware unlike the legacy compareDates function.
 */
export function compareDatesInTimezone(date1: Date | null, date2: Date | null, timezone: string): boolean {
  if (date1 === null || date2 === null) {
    return false;
  }

  // Compare UTC date components directly
  // Both dates are stored as UTC midnight representing calendar dates
  return (
    date1.getUTCDate() === date2.getUTCDate() &&
    date1.getUTCMonth() === date2.getUTCMonth() &&
    date1.getUTCFullYear() === date2.getUTCFullYear()
  );
}

export function isTodayOrPast(date: Date | null, timezone: string = "UTC"): boolean {
  if (date === null) {
    return false;
  }

  const getTodayResult = getToday(timezone);
  const today = new Date(getTodayResult);
  // Since getToday returns a UTC Date at midnight, normalize both to UTC midnight for comparison
  today.setUTCHours(0, 0, 0, 0);
  const inputDate = new Date(date);
  inputDate.setUTCHours(0, 0, 0, 0);

  return inputDate <= today;
}

export function isPast(date: Date | null, timezone: string = "UTC"): boolean {
  if (date === null) {
    return false;
  }

  const today = new Date(getToday(timezone));
  // Since getToday returns a UTC Date at midnight, normalize both to UTC midnight for comparison
  today.setUTCHours(0, 0, 0, 0);
  const inputDate = new Date(date);
  inputDate.setUTCHours(0, 0, 0, 0);

  return inputDate < today;
}

export function isRecurringTask(task: Task): boolean {
  const recurringPatterns = [
    /every day/i,
    /daily/i,
    /every \d+ days?/i,
    /weekly/i,
    /every week/i,
    /every \d+ weeks?/i,
    /monthly/i,
    /every \d+ months?/i,
  ];

  return recurringPatterns.some((pattern) => pattern.test(task.title));
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString);

  const formattedDate = date.toISOString().split("T")[0];
  return formattedDate;
}
