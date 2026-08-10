import React, { useState, useRef, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { SchemaIcon } from '../../assets/icons/SchemaIcon';
import { FactsIcon } from '../../assets/icons/FactsIcon';
import { GraphIcon } from '../../assets/icons/GraphIcon';

interface SecondaryNavigationLinksProps {
  hasSubscription: boolean;
  isCollapsed: boolean;
}

interface CollapsibleLinkProps {
  to: string;
  icon: React.ReactNode;
  label: string;
  isCollapsed: boolean;
  isPro?: boolean;
  hasSubscription?: boolean;
}

function CollapsibleLink({
  to,
  icon,
  label,
  isCollapsed,
  isPro = false,
  hasSubscription = true,
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

  const showDotIndicator = isCollapsed && isPro && !hasSubscription;

  return (
    <li>
      <div className="relative">
        <Link
          ref={linkRef}
          to={to}
          className={`
            flex items-center relative rounded-md transition-colors
            ${isActive ? 'bg-gray-100' : 'hover:bg-gray-100'}
            ${
              isCollapsed
                ? 'w-12 h-12 mx-auto justify-center'
                : 'w-full px-3 py-2.5 md:px-2 md:py-1 min-h-[44px] md:min-h-0'
            }
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

          {!isCollapsed && <span className="px-2 flex-grow">{label}</span>}

          {!isCollapsed && isPro && !hasSubscription && (
            <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">
              PRO
            </span>
          )}

          {showDotIndicator && (
            <span
              className="absolute top-1 right-1 w-2 h-2 rounded-full bg-purple-500"
              aria-label="PRO feature"
            />
          )}
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
    </li>
  );
}

export function SecondaryNavigationLinks({
  hasSubscription,
  isCollapsed,
}: SecondaryNavigationLinksProps) {
  return (
    <div className={`p-2 ${isCollapsed ? 'px-1' : ''}`}>
      <ul className="space-y-1">
        <CollapsibleLink
          to="/app/schemas"
          icon={<SchemaIcon />}
          label="Schemas"
          isCollapsed={isCollapsed}
          isPro={true}
          hasSubscription={hasSubscription}
        />
        <CollapsibleLink
          to="/app/graph"
          icon={<GraphIcon />}
          label="Graph"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/facts"
          icon={<FactsIcon />}
          label="Facts"
          isCollapsed={isCollapsed}
          isPro={true}
          hasSubscription={hasSubscription}
        />
      </ul>
    </div>
  );
}
