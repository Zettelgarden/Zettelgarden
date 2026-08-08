import React from 'react';
import { MenuIcon } from '../../assets/icons/MenuIcon';

interface SidebarMobileMenuProps {
  isSidebarOpen: boolean;
  setIsSidebarOpen: (isOpen: boolean) => void;
}

export function SidebarMobileMenu({
  isSidebarOpen,
  setIsSidebarOpen,
}: SidebarMobileMenuProps) {
  const handleBackdropKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setIsSidebarOpen(false);
    }
  };

  return (
    <>
      {/* Mobile Menu Button with safe area support */}
      <button
        className="md:hidden fixed top-4 right-4 z-[60] p-2 min-w-[44px] min-h-[44px] bg-white rounded shadow safe-top-fixed"
        style={{ top: `max(1rem, env(safe-area-inset-top, 0px))` }}
        onClick={() => setIsSidebarOpen(!isSidebarOpen)}
        aria-label={isSidebarOpen ? 'Close sidebar menu' : 'Open sidebar menu'}
        aria-expanded={isSidebarOpen}
      >
        <MenuIcon />
      </button>

      {/* Mobile Backdrop with safe area support */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 md:hidden z-[45] safe-all"
          onClick={() => setIsSidebarOpen(false)}
          onKeyDown={handleBackdropKeyDown}
          tabIndex={0}
          role="button"
          aria-label="Close sidebar menu"
        />
      )}
    </>
  );
}
