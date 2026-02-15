import React, { useState } from "react";
import { useIsDesktop } from "../../hooks/useWindowSize";
import { PANEL_THEMES, PanelTheme } from "./panelThemes";

export interface SidePanelLayoutProps {
  children: React.ReactNode;
  panelContent: React.ReactNode;
  theme: keyof typeof PANEL_THEMES;
  title: string;
  subtitle?: string;
  onClose: () => void;
  icon?: React.ReactNode;
}

/**
 * Generic split-view layout component with a collapsible side panel.
 * Used for pinned cards, chat panels, and other side panel features.
 *
 * Desktop: Side panel is always visible on the right
 * Mobile: Side panel collapses to bottom, expandable with toggle button
 */
export const SidePanelLayout: React.FC<SidePanelLayoutProps> = ({
  children,
  panelContent,
  theme,
  title,
  subtitle,
  onClose,
  icon,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDesktop = useIsDesktop(1024);
  const colors = PANEL_THEMES[theme];

  return (
    <div className="flex flex-col lg:flex-row h-full">
      {/* Main Content Pane - Left side on desktop, top on mobile */}
      <div className={`
        w-full lg:w-1/2
        border-b lg:border-b-0 lg:border-r border-gray-200
        overflow-y-auto
        ${isExpanded ? 'h-1/3 md:h-1/2 lg:h-full' : 'flex-1 lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full">
          {children}
        </div>
      </div>

      {/* Side Panel - Right side on desktop, collapsible bottom on mobile */}
      <div className={`
        w-full lg:w-1/2
        ${isExpanded ? 'h-2/3 md:h-1/2' : 'h-auto lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className={`h-full ${colors.bg} flex flex-col`}>
          {/* Desktop Header */}
          <div className={`hidden lg:flex ${colors.bgLight} px-3 py-2 border-b ${colors.border} items-center justify-between`}>
            <div className="flex items-center gap-2">
              {icon && <div className={`${colors.text}`}>{icon}</div>}
              <span className={`text-xs font-semibold uppercase tracking-wide ${colors.textMuted}`}>
                {title}
              </span>
              {subtitle && (
                <span className={`${colors.text} text-sm font-medium truncate max-w-md`}>
                  {subtitle}
                </span>
              )}
            </div>
            <button
              onClick={onClose}
              className={`${colors.text} ${colors.hoverBg} px-2 py-1 rounded text-sm flex items-center gap-1`}
              title="Close panel"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              Close
            </button>
          </div>

          {/* Mobile collapse/expand button */}
          <div className={`lg:hidden ${colors.bgLight} p-2 border-b ${colors.border}`}>
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className={`flex items-center justify-between w-full ${colors.textMuted}`}
            >
              <div className="flex items-center gap-2">
                {icon && <div className={`${colors.text}`}>{icon}</div>}
                <span className="text-xs font-medium uppercase tracking-wide">
                  {title}
                </span>
                {subtitle && (
                  <span className="text-sm font-medium truncate">
                    {subtitle}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={(e) => { e.stopPropagation(); onClose(); }}
                  className={`${colors.text} ${colors.hoverBg} p-1`}
                  title="Close panel"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
                <svg
                  className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </button>
          </div>

          {/* Panel content - conditionally shown on mobile */}
          {(isExpanded || isDesktop) && (
            <div className="flex-1 min-h-0">
              {panelContent}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
