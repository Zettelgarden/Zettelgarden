import { useEffect, useState } from 'react';

/**
 * Hook to manage dropdown menu state with click-outside detection.
 * Used consistently across Task*Display components.
 */
export function useTaskDropdown(initialOpen = false) {
  const [isOpen, setIsOpen] = useState<boolean>(initialOpen);

  const toggle = (e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation();
    }
    setIsOpen((prev) => !prev);
  };

  const open = (e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation();
    }
    setIsOpen(true);
  };

  const close = () => {
    setIsOpen(false);
  };

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setIsOpen(false);
    if (isOpen) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [isOpen]);

  return {
    isOpen,
    toggle,
    open,
    close,
    setIsOpen,
  };
}
