import React from 'react';
import { Link, useLocation } from 'react-router-dom';

interface SidebarLinkProps {
  to: string;
  children: React.ReactNode;
}

export function SidebarLink({ to, children }: SidebarLinkProps) {
  const location = useLocation();
  const isActive = location.pathname + location.search === to;

  return (
    <Link
      to={to}
      className={`flex items-center px-3 py-2.5 md:px-2 md:py-1 min-h-[44px] md:min-h-0 font-medium rounded-md hover:bg-gray-100 ${
        isActive ? 'bg-gray-100' : ''
      }`}
    >
      {/* Add icon wrapper for first child */}
      {React.Children.map(children, (child, index) => {
        if (index === 0) {
          return (
            <span className="w-6 h-6 flex items-center justify-center flex-shrink-0">
              {child}
            </span>
          );
        }
        return child;
      })}
    </Link>
  );
}
