import React from "react";
import { SidebarLink } from "../SidebarLink";
import { EntityIcon } from "../../assets/icons/EntityIcon";
import { FactsIcon } from "../../assets/icons/FactsIcon";

interface SecondaryNavigationLinksProps {
  hasSubscription: boolean;
}

export function SecondaryNavigationLinks({ hasSubscription }: SecondaryNavigationLinksProps) {
  return (
    <div className="p-2">
      <ul className="space-y-1">
        <SidebarLink to="/app/entities">
          <EntityIcon />
          <span className="px-2 flex-grow">Entities</span>
          {!hasSubscription && (
            <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">PRO</span>
          )}
        </SidebarLink>
        <SidebarLink to="/app/facts">
          <FactsIcon />
          <span className="px-2 flex-grow">Facts</span>
          {!hasSubscription && (
            <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">PRO</span>
          )}
        </SidebarLink>
      </ul>
    </div>
  );
}