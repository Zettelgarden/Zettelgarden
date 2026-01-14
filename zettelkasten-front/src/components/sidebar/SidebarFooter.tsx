import React from "react";
import { SidebarLink } from "../SidebarLink";
import { BookOpenIcon } from "../../assets/icons/BookOpenIcon";
import { SettingsIcon } from "../../assets/icons/SettingsIcon";

export function SidebarFooter() {
  return (
    <div className="p-2 border-t">
      <div className="flex justify-end space-x-4 pr-2">
        <SidebarLink to="/app/help">
          <BookOpenIcon />
        </SidebarLink>
        <SidebarLink to="/app/settings">
          <SettingsIcon />
        </SidebarLink>
      </div>
    </div>
  );
}