import React from "react";
import { SidebarLink } from "../SidebarLink";
import { SearchIcon } from "../../assets/icons/SearchIcon";
import { TasksIcon } from "../../assets/icons/TasksIcon";
import { ChatIcon } from "../../assets/icons/ChatIcon";

interface NavigationLinksProps {
  todayTasksCount: number;
  hasSubscription: boolean;
}

export function NavigationLinks({ todayTasksCount, hasSubscription }: NavigationLinksProps) {
  return (
    <div className="p-2">
      <ul className="space-y-1">
        <SidebarLink to="/app/search?recent=true">
          <SearchIcon />
          <span className="px-2 flex-grow">Search</span>
        </SidebarLink>

        <SidebarLink to="/app/tasks">
          <TasksIcon />
          <span className="px-2 flex-grow">Tasks</span>
          <span className="px-3 py-1.5 md:px-2 md:py-1 text-xs bg-blue-100 rounded-full min-h-[32px] md:min-h-0 flex items-center">
            {todayTasksCount}
          </span>
        </SidebarLink>

        <SidebarLink to="/app/chat">
          <ChatIcon />
          <span className="px-2 flex-grow">Chat</span>
          {!hasSubscription && (
            <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-3 py-1.5 md:px-2 md:py-0.5 rounded-full min-h-[32px] md:min-h-0 flex items-center">PRO</span>
          )}
        </SidebarLink>
      </ul>
    </div>
  );
}