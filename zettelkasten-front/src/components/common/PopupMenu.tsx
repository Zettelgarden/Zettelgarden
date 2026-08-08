import React from 'react';

interface MenuOption {
  label: string;
  onClick: () => void;
  className?: string;
}

interface PopupMenuProps {
  options: MenuOption[];
  isOpen: boolean;
  className?: string;
}

export function PopupMenu({ options, isOpen, className = '' }: PopupMenuProps) {
  if (!isOpen) return null;

  return (
    <div
      className={`absolute right-0 top-12 bg-white border border-gray-200 rounded-md shadow-lg z-10 w-48 ${className}`}
    >
      {options.map((option, index) => (
        <button
          key={index}
          onClick={option.onClick}
          className={`block w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 ${
            option.className || ''
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
