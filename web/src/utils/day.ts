/**
 * Calendar days of the food diary, written as YYYY-MM-DD in the person's own
 * time zone. A day is never a UTC timestamp, so eating at 23:30 lands on the
 * day the person was living, not the next one in Greenwich.
 */
import type { DayKey } from "@/types/domain";

const pad = (n: number) => String(n).padStart(2, "0");

/** The day a local date falls on. */
export function dayOf(date: Date): DayKey {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

export function today(): DayKey {
  return dayOf(new Date());
}

/** Noon of the day, so formatting and arithmetic never cross a DST edge. */
export function dateOf(day: DayKey): Date {
  const [year, month, date] = day.split("-").map(Number);
  return new Date(year!, month! - 1, date!, 12);
}

export function addDays(day: DayKey, days: number): DayKey {
  const date = dateOf(day);
  date.setDate(date.getDate() + days);
  return dayOf(date);
}

export function isDayKey(value: unknown): value is DayKey {
  return typeof value === "string" && /^\d{4}-\d{2}-\d{2}$/.test(value) && dayOf(dateOf(value)) === value;
}
