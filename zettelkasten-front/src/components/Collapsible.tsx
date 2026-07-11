import React, { useState } from "react";

interface CollapsibleProps {
  title: string;
  /** Optional count shown as a muted badge next to the title. */
  count?: number;
  /** Whether the section starts expanded. Defaults to true. */
  defaultOpen?: boolean;
  /** Optional element rendered on the right of the header. Clicks here do not toggle. */
  rightElement?: React.ReactNode;
  /** Optional id suffix to keep aria attributes unique. */
  idSuffix?: string;
  children: React.ReactNode;
}

/**
 * A calm, Obsidian-style collapsible section. Renders as a header row with a
 * rotating chevron; children are shown/hidden based on the open state.
 * Uses a <button> for accessibility and stops propagation on the right
 * element so actions placed there don't toggle the section.
 */
export function Collapsible({
  title,
  count,
  defaultOpen = true,
  rightElement,
  idSuffix,
  children,
}: CollapsibleProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const panelId = idSuffix ? `collapsible-${idSuffix}` : undefined;

  return (
    <div>
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          aria-expanded={isOpen}
          aria-controls={panelId}
          className="flex items-center gap-1.5 text-sm font-medium text-gray-700 hover:text-gray-900 transition-colors py-1"
        >
          <svg
            className={`h-3.5 w-3.5 text-gray-400 transition-transform duration-150 ${
              isOpen ? "rotate-90" : ""
            }`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5l7 7-7 7"
            />
          </svg>
          {title}
          {count !== undefined && count > 0 && (
            <span className="text-xs text-gray-400 font-normal">{count}</span>
          )}
        </button>
        {rightElement && (
          <div onClick={(e) => e.stopPropagation()}>{rightElement}</div>
        )}
      </div>
      {isOpen && (
        <div id={panelId} className="mt-2 ml-5">
          {children}
        </div>
      )}
    </div>
  );
}
