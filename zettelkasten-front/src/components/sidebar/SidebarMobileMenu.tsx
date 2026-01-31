import React from "react";
import { MenuIcon } from "../../assets/icons/MenuIcon";

interface SidebarMobileMenuProps {
  isSidebarOpen: boolean;
  setIsSidebarOpen: (isOpen: boolean) => void;
}

export function SidebarMobileMenu({ isSidebarOpen, setIsSidebarOpen }: SidebarMobileMenuProps) {
  const handleBackdropKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setIsSidebarOpen(false);
    }
  };

  return (
    <>
      {/* Mobile Menu Button */}
      <button
        className="md:hidden fixed top-4 right-4 z-[60] p-2 bg-white rounded shadow"
        onClick={() => setIsSidebarOpen(!isSidebarOpen)}
        aria-label={isSidebarOpen ? "Close sidebar menu" : "Open sidebar menu"}
        aria-expanded={isSidebarOpen}
      >
        <MenuIcon />
      </button>

      {/* Mobile Backdrop */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 md:hidden z-[45]"
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
