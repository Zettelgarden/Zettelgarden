// zettelkasten-front/src/components/cards/ViewMobileAccordion.tsx
import React, { useState } from "react";

interface ViewMobileAccordionProps {
  title: string;
  icon?: React.ReactNode;
  defaultExpanded?: boolean;
  rightElement?: React.ReactNode;
  children: React.ReactNode;
}

export function ViewMobileAccordion({
  title,
  icon,
  defaultExpanded = false,
  rightElement,
  children,
}: ViewMobileAccordionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <div className="border-b border-gray-200">
      <button
        className="w-full sticky top-0 bg-gray-50 px-4 py-3 flex items-center justify-between text-left hover:bg-gray-100 transition-colors z-10"
        onClick={() => setIsExpanded(!isExpanded)}
        aria-expanded={isExpanded}
      >
        <div className="flex items-center gap-2">
          <span className="text-gray-400 text-sm">
            {isExpanded ? "▼" : "►"}
          </span>
          {icon && <span className="text-gray-500">{icon}</span>}
          <span className="font-medium text-gray-900">{title}</span>
        </div>
        {rightElement && (
          <div onClick={(e) => e.stopPropagation()}>
            {rightElement}
          </div>
        )}
      </button>
      {isExpanded && (
        <div className="px-4 py-3 bg-white">
          {children}
        </div>
      )}
    </div>
  );
}
