/**
 * Shared priority configuration for tasks
 */

export const PRIORITY_CONFIG = {
  A: { color: "#EF4444", icon: "🔴", label: "High" },
  B: { color: "#F59E0B", icon: "🟠", label: "Medium" },
  C: { color: "#3B82F6", icon: "🔵", label: "Low" },
} as const;

export type PriorityLevel = keyof typeof PRIORITY_CONFIG;

export const PRIORITY_OPTIONS = [
  { value: "A", label: "High", icon: "🔴" },
  { value: "B", label: "Medium", icon: "🟠" },
  { value: "C", label: "Low", icon: "🔵" },
  { value: null, label: "None", icon: "○" },
] as const;
