import React from 'react';
import { Menu as HeadlessMenu } from '@headlessui/react';

export interface MenuProps {
  /** Trigger content rendered inside the menu button */
  button: React.ReactNode;
  /** Extra classes for the trigger button */
  buttonClassName?: string;
  /** Which side the panel drops toward (default: right) */
  align?: 'left' | 'right';
  /** Extra classes for the items panel */
  panelClassName?: string;
  /** Menu items (use MenuItem or raw HeadlessMenu.Item) */
  children: React.ReactNode;
}

/**
 * Menu — the shared dropdown-menu shell on Headless UI Menu.
 *
 * Provides the trigger button, panel positioning (right/left), and the
 * standard panel look; keyboard navigation and focus management come from
 * Headless UI. Use <MenuItem> for plain action items or a raw
 * HeadlessMenu.Item for custom item content.
 */
export function Menu({
  button,
  buttonClassName = '',
  align = 'right',
  panelClassName = '',
  children,
}: MenuProps) {
  const alignClasses =
    align === 'right' ? 'right-0 origin-top-right' : 'left-0 origin-top-left';

  return (
    <HeadlessMenu as="div" className="relative flex-shrink-0">
      <HeadlessMenu.Button
        type="button"
        className={`p-1.5 md:p-1 rounded hover:bg-gray-100 transition-colors min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${buttonClassName}`}
      >
        {button}
      </HeadlessMenu.Button>
      <HeadlessMenu.Items
        className={`absolute ${alignClasses} z-10 mt-1 bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none ${panelClassName}`}
      >
        {children}
      </HeadlessMenu.Items>
    </HeadlessMenu>
  );
}

export interface MenuItemProps {
  onClick?: () => void;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
}

/**
 * MenuItem — a standard full-width action item with the 44px touch target,
 * active-state highlight and keyboard navigation (via Headless UI).
 */
export function MenuItem({
  onClick,
  disabled = false,
  className = '',
  children,
}: MenuItemProps) {
  return (
    <HeadlessMenu.Item disabled={disabled}>
      {({ active }) => (
        <button
          type="button"
          onClick={onClick}
          disabled={disabled}
          className={`${
            active ? 'bg-gray-100' : ''
          } flex w-full items-center px-4 py-3 min-h-[44px] text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
        >
          {children}
        </button>
      )}
    </HeadlessMenu.Item>
  );
}

/** Raw Headless UI Menu.Item (render-prop) for custom item content */
export const MenuRawItem = HeadlessMenu.Item;
