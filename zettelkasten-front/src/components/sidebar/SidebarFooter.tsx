import React, { useState, useRef, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { BookOpenIcon } from '../../assets/icons/BookOpenIcon';
import { SettingsIcon } from '../../assets/icons/SettingsIcon';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { SyncStatusIndicator } from '../SyncStatusIndicator';

interface SidebarFooterProps {
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

interface CollapsibleLinkProps {
  to: string;
  icon: React.ReactNode;
  label: string;
  isCollapsed: boolean;
}

function CollapsibleLink({
  to,
  icon,
  label,
  isCollapsed,
}: CollapsibleLinkProps) {
  const location = useLocation();
  const isActive = location.pathname + location.search === to;
  const [showTooltip, setShowTooltip] = useState(false);
  const linkRef = useRef<HTMLAnchorElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (showTooltip && tooltipRef.current && linkRef.current) {
      const linkRect = linkRef.current.getBoundingClientRect();
      const scrollTop =
        window.pageYOffset || document.documentElement.scrollTop;
      tooltipRef.current.style.top = `${
        linkRect.top + linkRect.height / 2 - 10 + scrollTop
      }px`;
    }
  }, [showTooltip]);

  return (
    <div className="relative">
      <Link
        ref={linkRef}
        to={to}
        className={`
          flex items-center rounded-md transition-colors min-h-[44px]
          ${isActive ? 'bg-gray-100' : 'hover:bg-gray-100'}
          ${isCollapsed ? 'w-12 h-12 justify-center' : 'px-3 py-2'}
        `}
        onMouseEnter={() => isCollapsed && setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        onFocus={() => isCollapsed && setShowTooltip(true)}
        onBlur={() => setShowTooltip(false)}
        aria-label={label}
      >
        <span
          className={
            isCollapsed
              ? ''
              : 'w-6 h-6 flex items-center justify-center flex-shrink-0'
          }
        >
          {icon}
        </span>
      </Link>

      {isCollapsed && showTooltip && (
        <div
          ref={tooltipRef}
          className="fixed left-[4.5rem] px-2 py-1 bg-gray-900 text-white text-xs rounded whitespace-nowrap z-50"
          role="tooltip"
          aria-hidden="true"
        >
          {label}
          <div
            className="absolute top-1/2 -left-1 w-2 h-2 bg-gray-900 transform -translate-y-1/2 rotate-45"
            aria-hidden="true"
          />
        </div>
      )}
    </div>
  );
}

export function SidebarFooter({
  isCollapsed,
  onToggleCollapse,
}: SidebarFooterProps) {
  const [showTooltip, setShowTooltip] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (showTooltip && tooltipRef.current && buttonRef.current) {
      const buttonRect = buttonRef.current.getBoundingClientRect();
      const scrollTop =
        window.pageYOffset || document.documentElement.scrollTop;
      tooltipRef.current.style.top = `${
        buttonRect.top + buttonRect.height / 2 - 10 + scrollTop
      }px`;
    }
  }, [showTooltip]);

  return (
    <div
      className={`p-2 border-t ${
        isCollapsed ? 'flex flex-col items-center gap-2' : ''
      }`}
      style={{
        paddingBottom: 'max(0.5rem, env(safe-area-inset-bottom, 0.5rem))',
      }}
    >
      <div
        className={`flex ${
          isCollapsed ? 'flex-col gap-2' : 'justify-end space-x-4 pr-2'
        }`}
      >
        <SyncStatusIndicator collapsed={isCollapsed} />
        <CollapsibleLink
          to="/app/help"
          icon={<BookOpenIcon />}
          label="Help"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/settings"
          icon={<SettingsIcon />}
          label="Settings"
          isCollapsed={isCollapsed}
        />
        {isCollapsed ? (
          <div className="relative">
            <button
              ref={buttonRef}
              onClick={onToggleCollapse}
              className="w-12 h-12 flex items-center justify-center rounded-md hover:bg-gray-100 transition-colors"
              aria-label="Expand sidebar"
              aria-pressed={isCollapsed}
              onMouseEnter={() => setShowTooltip(true)}
              onMouseLeave={() => setShowTooltip(false)}
              onFocus={() => setShowTooltip(true)}
              onBlur={() => setShowTooltip(false)}
            >
              <ChevronRight
                size={18}
                className="transition-transform duration-300"
              />
            </button>
            {showTooltip && (
              <div
                ref={tooltipRef}
                className="fixed left-[4.5rem] px-2 py-1 bg-gray-900 text-white text-xs rounded whitespace-nowrap z-50"
                role="tooltip"
                aria-hidden="true"
              >
                Expand sidebar
                <div
                  className="absolute top-1/2 -left-1 w-2 h-2 bg-gray-900 transform -translate-y-1/2 rotate-45"
                  aria-hidden="true"
                />
              </div>
            )}
          </div>
        ) : (
          <button
            onClick={onToggleCollapse}
            className="min-h-[44px] px-3 py-2 flex items-center rounded-md hover:bg-gray-100 transition-colors"
            aria-label="Collapse sidebar"
            aria-pressed={isCollapsed}
            title="Collapse sidebar"
          >
            <ChevronLeft
              size={18}
              className="transition-transform duration-300"
            />
          </button>
        )}
      </div>
    </div>
  );
}
