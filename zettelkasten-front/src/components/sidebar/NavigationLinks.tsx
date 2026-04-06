import React, { useState, useRef, useEffect } from "react";
import { Link, useLocation } from "react-router-dom";
import { SearchIcon } from "../../assets/icons/SearchIcon";
import { TasksIcon } from "../../assets/icons/TasksIcon";
import { CalendarIcon } from "../../assets/icons/CalendarIcon";
import { ChatIcon } from "../../assets/icons/ChatIcon";
import { RssIcon } from "../../assets/icons/RssIcon";
import { EmailIcon } from "../../assets/icons/EmailIcon";
import { EntityIcon } from "../../assets/icons/EntityIcon";
import { HabitsIcon } from "../../assets/icons/HabitsIcon";
import { FileIcon } from "../../assets/icons/FileIcon";

interface NavigationLinksProps {
  todayTasksCount: number;
  unreadRssCount: number;
  unreadEmailCount: number;
  hasSubscription: boolean;
  isCollapsed: boolean;
}

interface CollapsibleLinkProps {
  to: string;
  icon: React.ReactNode;
  label: string;
  isCollapsed: boolean;
  badge?: React.ReactNode;
  badgeCount?: number;
  isPro?: boolean;
  hasSubscription?: boolean;
}

function CollapsibleLink({
  to,
  icon,
  label,
  isCollapsed,
  badge,
  badgeCount = 0,
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
      const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
      tooltipRef.current.style.top = `${linkRect.top + linkRect.height / 2 - 10 + scrollTop}px`;
    }
  }, [showTooltip]);

  const showDotIndicator = isCollapsed && (badgeCount > 0 || (isPro && !hasSubscription));

  return (
    <li>
      <div className="relative">
        <Link
          ref={linkRef}
          to={to}
          className={`
            flex items-center relative rounded-md transition-colors
            ${isActive ? "bg-gray-100" : "hover:bg-gray-100"}
            ${isCollapsed ? "w-12 h-12 mx-auto justify-center" : "w-full px-3 py-2.5 md:px-2 md:py-1 min-h-[44px] md:min-h-0"}
          `}
          onMouseEnter={() => isCollapsed && setShowTooltip(true)}
          onMouseLeave={() => setShowTooltip(false)}
          onFocus={() => isCollapsed && setShowTooltip(true)}
          onBlur={() => setShowTooltip(false)}
          aria-label={label}
        >
          <span className={isCollapsed ? "" : "w-6 h-6 flex items-center justify-center flex-shrink-0"}>
            {icon}
          </span>

          {!isCollapsed && (
            <span className="px-2 flex-grow">{label}</span>
          )}

          {!isCollapsed && badge}

          {showDotIndicator && (
            <span
              className={`absolute top-1 right-1 w-2 h-2 rounded-full ${
                isPro && !hasSubscription ? "bg-purple-500" : "bg-blue-500"
              }`}
              aria-label={
                isPro && !hasSubscription ? "PRO feature" : `${badgeCount} items`
              }
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

export function NavigationLinks({ todayTasksCount, unreadRssCount, unreadEmailCount, hasSubscription, isCollapsed }: NavigationLinksProps) {
  const SectionHeader = ({ children }: { children: React.ReactNode }) => {
    if (isCollapsed) return null;
    return (
      <li className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wide">
        {children}
      </li>
    );
  };

  return (
    <div className={`p-2 ${isCollapsed ? "px-1" : ""}`}>
      <ul className="space-y-1">
        {/* Group 1: Search, Chat, RSS */}
        <SectionHeader>Knowledge</SectionHeader>
        <CollapsibleLink
          to="/app/search?recent=true"
          icon={<SearchIcon />}
          label="Search"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/entities"
          icon={<EntityIcon />}
          label="Entities"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/chat"
          icon={<ChatIcon />}
          label="Chat"
          isCollapsed={isCollapsed}
          isPro={true}
          hasSubscription={hasSubscription}
          badge={
            !hasSubscription && (
              <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-3 py-1.5 md:px-2 md:py-0.5 rounded-full min-h-[32px] md:min-h-0 flex items-center">
                PRO
              </span>
            )
          }
        />
        <CollapsibleLink
          to="/app/rss"
          icon={<RssIcon />}
          label="RSS"
          isCollapsed={isCollapsed}
          badgeCount={unreadRssCount}
          badge={
            unreadRssCount > 0 && (
              <span className="px-3 py-1.5 md:px-2 md:py-1 text-xs bg-blue-100 rounded-full min-h-[32px] md:min-h-0 flex items-center">
                {unreadRssCount}
              </span>
            )
          }
        />

        {/* Group 2: Email, Tasks, Calendar, Habits */}
        <SectionHeader>Organization</SectionHeader>
        <CollapsibleLink
          to="/app/emails"
          icon={<EmailIcon />}
          label="Email"
          isCollapsed={isCollapsed}
          badgeCount={unreadEmailCount}
          badge={
            unreadEmailCount > 0 && (
              <span className="px-3 py-1.5 md:px-2 md:py-1 text-xs bg-blue-100 rounded-full min-h-[32px] md:min-h-0 flex items-center">
                {unreadEmailCount}
              </span>
            )
          }
        />
        <CollapsibleLink
          to="/app/tasks"
          icon={<TasksIcon />}
          label="Tasks"
          isCollapsed={isCollapsed}
          badgeCount={todayTasksCount}
          badge={
            <span className="px-3 py-1.5 md:px-2 md:py-1 text-xs bg-blue-100 rounded-full min-h-[32px] md:min-h-0 flex items-center">
              {todayTasksCount}
            </span>
          }
        />
        <CollapsibleLink
          to="/app/calendar"
          icon={<CalendarIcon />}
          label="Calendar"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/habits"
          icon={<HabitsIcon />}
          label="Habits"
          isCollapsed={isCollapsed}
        />
        <CollapsibleLink
          to="/app/files"
          icon={<FileIcon />}
          label="Files"
          isCollapsed={isCollapsed}
        />


      </ul>
    </div>
  );
}