import { LucideIcon } from 'lucide-react';

export interface PanelTheme {
  bg: string;
  bgLight: string;
  text: string;
  textMuted: string;
  border: string;
  hoverBg: string;
}

export const PANEL_THEMES: Record<'blue' | 'green', PanelTheme> = {
  blue: {
    bg: 'bg-blue-50',
    bgLight: 'bg-blue-100',
    text: 'text-blue-600',
    textMuted: 'text-blue-700',
    border: 'border-blue-200',
    hoverBg: 'hover:bg-blue-200',
  },
  green: {
    bg: 'bg-green-50',
    bgLight: 'bg-green-100',
    text: 'text-green-600',
    textMuted: 'text-green-700',
    border: 'border-green-200',
    hoverBg: 'hover:bg-green-200',
  },
} as const;
