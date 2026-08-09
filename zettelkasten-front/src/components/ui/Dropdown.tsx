import React from 'react';
import { Menu as HeadlessMenu } from '@headlessui/react';

export interface DropdownOption<T> {
  value: T;
  label: React.ReactNode;
}

export interface DropdownProps<T> {
  value: T | undefined;
  options: DropdownOption<T>[];
  onChange: (value: T) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  panelClassName?: string;
}

/**
 * Dropdown — a select-style menu on Headless UI Menu: trigger shows the
 * current value with a chevron, the panel lists options with a check mark
 * on the selected one. Keyboard navigation and focus management are built in.
 */
export function Dropdown<T>({
  value,
  options,
  onChange,
  placeholder = 'Select…',
  disabled = false,
  className = '',
  panelClassName = '',
}: DropdownProps<T>) {
  const selected = options.find((option) => option.value === value);

  return (
    <HeadlessMenu as="div" className={`relative ${className}`}>
      <HeadlessMenu.Button
        type="button"
        disabled={disabled}
        className="flex w-full items-center justify-between rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span className={selected ? 'text-gray-900' : 'text-gray-500'}>
          {selected ? selected.label : placeholder}
        </span>
        <svg
          className="w-4 h-4 ml-2 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </HeadlessMenu.Button>
      <HeadlessMenu.Items
        className={`absolute right-0 z-10 mt-1 w-full origin-top-right bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none ${panelClassName}`}
      >
        {options.map((option) => (
          <HeadlessMenu.Item key={String(option.value)} disabled={disabled}>
            {({ active }) => (
              <button
                type="button"
                onClick={() => onChange(option.value)}
                className={`${
                  active ? 'bg-gray-100' : ''
                } flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 ${
                  option.value === value ? 'font-medium' : ''
                }`}
              >
                <span className="flex-1 text-left">{option.label}</span>
                {option.value === value && (
                  <svg
                    className="w-4 h-4 text-blue-600"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                )}
              </button>
            )}
          </HeadlessMenu.Item>
        ))}
      </HeadlessMenu.Items>
    </HeadlessMenu>
  );
}
