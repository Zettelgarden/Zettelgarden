import React from "react";
import { SidebarLink } from "../SidebarLink";
import { EntityIcon } from "../../assets/icons/EntityIcon";
import { FactsIcon } from "../../assets/icons/FactsIcon";
import { MemoryIcon } from "../../assets/icons/MemoryIcon";
import { ChartIcon } from "../../assets/icons/ChartIcon";
import { SchemaIcon } from "../../assets/icons/SchemaIcon";

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
        <SidebarLink to="/app/memory">
          <MemoryIcon />
          <span className="px-2 flex-grow">Memory</span>
        </SidebarLink>
        <SidebarLink to="/app/schemas">
          <SchemaIcon />
          <span className="px-2 flex-grow">Schemas</span>
        </SidebarLink>
        <SidebarLink to="/app/stats">
          <ChartIcon />
          <span className="px-2 flex-grow">Stats</span>
        </SidebarLink>
      </ul>
    </div>
  );
}