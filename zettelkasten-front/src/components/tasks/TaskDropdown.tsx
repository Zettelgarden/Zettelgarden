import React, { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";

interface TaskDropdownProps {
  isOpen: boolean;
  onToggle: (e: React.MouseEvent) => void;
  onClose: () => void;
  display: {
    icon?: string;
    text: string;
    color: string;
  };
  menuClassName?: string;
  children: React.ReactNode;
  triggerRef?: React.RefObject<HTMLElement>;
  usePortal?: boolean;
}

/**
 * Consistent dropdown menu wrapper for Task*Display components.
 * Provides:
 * - Clickable display badge with icon, text, and color styling
 * - Dropdown menu container with consistent styling
 * - Click-outside and stopPropagation handling
 * - Optional portal rendering for z-index stacking context issues
 */
export function TaskDropdown({
  isOpen,
  onToggle,
  onClose,
  display,
  menuClassName = "min-w-[140px]",
  children,
  triggerRef,
  usePortal = false,
}: TaskDropdownProps) {
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number } | null>(null);
  const internalRef = useRef<HTMLSpanElement>(null);
  const effectiveRef = triggerRef || internalRef;

  // Close dropdown when clicking outside (portal mode only)
  useEffect(() => {
    if (!usePortal || !isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (effectiveRef.current && !effectiveRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [usePortal, isOpen, onClose, effectiveRef]);

  // Calculate position when opening with portal
  const handleToggle = (e: React.MouseEvent) => {
    if (usePortal && effectiveRef.current && !isOpen) {
      const rect = effectiveRef.current.getBoundingClientRect();
      setDropdownPosition({ top: rect.bottom + 4, left: rect.left });
    } else {
      setDropdownPosition(null);
    }
    onToggle(e);
  };

  // Keyboard navigation for dropdown menu (portal mode)
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!usePortal || !isOpen) return;
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  };

  const dropdownContent = (
    <div
      className={`${
        usePortal ? "fixed z-50" : "absolute z-20"
      } mt-1 bg-white rounded-md shadow-lg py-1 border border-gray-200 ${menuClassName}`}
      style={
        usePortal && dropdownPosition
          ? { top: `${dropdownPosition.top}px`, left: `${dropdownPosition.left}px` }
          : undefined
      }
      onClick={(e) => {
        e.stopPropagation();
        onClose();
      }}
      onKeyDown={handleKeyDown}
    >
      {children}
    </div>
  );

  return (
    <div className="relative inline-block">
      <span
        ref={triggerRef ? undefined : effectiveRef}
        onClick={handleToggle}
        className="cursor-pointer inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: display.color + "20",
          color: display.color,
          border: `1px solid ${display.color}40`,
        }}
      >
        {display.icon && <span>{display.icon}</span>}
        <span>{display.text}</span>
      </span>

      {isOpen &&
        (usePortal && dropdownPosition
          ? createPortal(dropdownContent, document.body)
          : dropdownContent)}
    </div>
  );
}
