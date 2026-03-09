/**
 * Shared constants for habit-related components
 */

/** Available emoji icons for habits */
export const HABIT_ICONS = ['💊', '🏃', '📚', '💧', '🧘', '💪', '🎯', '✅'] as const;

/** Available colors for habits (Tailwind-compatible hex values) */
export const HABIT_COLORS = [
  '#10b981', // emerald-500
  '#3b82f6', // blue-500
  '#f59e0b', // amber-500
  '#ef4444', // red-500
  '#8b5cf6', // violet-500
  '#ec4899', // pink-500
] as const;

/** Type for habit icon values */
export type HabitIcon = (typeof HABIT_ICONS)[number];

/** Type for habit color values */
export type HabitColor = (typeof HABIT_COLORS)[number];
