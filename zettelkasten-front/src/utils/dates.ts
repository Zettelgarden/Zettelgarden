import { Task } from "src/models/Task";

export function getToday(timezone: string = "UTC"): Date {
  const now = new Date();
  // For timezone support, we'll create a new Date at midnight UTC
  const midnight = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0));
  return midnight;
}

export function getTomorrow(timezone: string = "UTC"): Date {
  const today = new Date(getToday(timezone));
  today.setDate(today.getDate() + 1);
  return today;
}

export function getYesterday(timezone: string = "UTC"): Date {
  const today = new Date(getToday(timezone));
  today.setDate(today.getDate() - 1);
  return today;
}

export function getNextWeek(timezone: string = "UTC"): Date {
  const today = new Date(getToday(timezone));
  today.setDate(today.getDate() + 7);
  return today;
}

export function isFriday(timezone: string = "UTC"): boolean {
  const today = new Date();
  return today.getDay() === 5; // 5 = Friday (0 = Sunday)
}

export function getNextMonday(timezone: string = "UTC"): Date {
  const today = new Date();
  const day = today.getDay();
  const diff = today.getDate() + (8 - day) % 7;
  const nextMonday = new Date(today);
  nextMonday.setDate(diff);
  return nextMonday;
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

export function isTodayOrPast(date: Date | null, timezone: string = "UTC"): boolean {
  if (date === null) {
    return false;
  }

  const today = new Date(getToday(timezone));
  today.setHours(0, 0, 0, 0);
  const inputDate = new Date(date);
  inputDate.setHours(0, 0, 0, 0);

  return inputDate <= today;
}

export function isPast(date: Date | null, timezone: string = "UTC"): boolean {
  if (date === null) {
    return false;
  }

  const today = new Date(getToday(timezone));
  today.setHours(0, 0, 0, 0);
  const inputDate = new Date(date);
  inputDate.setHours(0, 0, 0, 0);

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
