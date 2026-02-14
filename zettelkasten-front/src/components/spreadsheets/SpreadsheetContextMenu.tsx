import React, { useEffect, useRef } from 'react';

export interface ContextMenuPosition {
  x: number;
  y: number;
}

export interface ContextMenuAction {
  label: string;
  action: () => void;
  disabled?: boolean;
}

interface SpreadsheetContextMenuProps {
  position: ContextMenuPosition | null;
  actions: ContextMenuAction[];
  onClose: () => void;
}

export function SpreadsheetContextMenu({ position, actions, onClose }: SpreadsheetContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (position) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [position, onClose]);

  if (!position) return null;

  const menuStyle: React.CSSProperties = {
    position: 'fixed',
    left: position.x,
    top: position.y,
    zIndex: 1000,
  };

  return (
    <div
      ref={menuRef}
      className="absolute bg-white border border-gray-300 rounded shadow-lg py-1 min-w-[160px]"
      style={menuStyle}
    >
      {actions.map((action, index) => (
        <button
          key={index}
          onClick={() => {
            if (!action.disabled) {
              action.action();
              onClose();
            }
          }}
          disabled={action.disabled}
          className={`
            w-full text-left px-4 py-2 text-sm
            ${action.disabled
              ? 'text-gray-400 cursor-not-allowed'
              : 'text-gray-700 hover:bg-gray-100 cursor-pointer'
            }
          `}
        >
          {action.label}
        </button>
      ))}
    </div>
  );
}
