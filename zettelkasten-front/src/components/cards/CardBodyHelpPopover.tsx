import React from "react";
import { Popover } from "@headlessui/react";

interface HelpFeature {
  name: string;
  description: string;
  syntax: string;
  example: string;
}

const features: HelpFeature[] = [
  {
    name: "Task Queries",
    description: "Embed a dynamic list of tasks based on filters",
    syntax: "{{tasks: <query>}}",
    example: "{{tasks: status:pending}}\n{{tasks: due:today priority:high}}"
  },
  {
    name: "Schema Tables",
    description: "Display a table from a schema definition",
    syntax: "{{schema: <ref>}}",
    example: "{{schema: book-review}}\n{{schema: 1|columns:title,status}}"
  },
  {
    name: "Spreadsheets",
    description: "Embed an editable spreadsheet in the card",
    syntax: "{{spreadsheet: <name>}}",
    example: "{{spreadsheet: project-tracking}}"
  },
  {
    name: "Card Links",
    description: "Link to other cards by their ID",
    syntax: "[<CardID>]",
    example: "[my-note]\n[20250201-reading-list]"
  }
];

const templateVariables: HelpFeature[] = [
  {
    name: "Date Variables",
    description: "Insert date/time when using card templates",
    syntax: "$<variable>",
    example: "$date      → 2025-02-01\n$time      → 14:30\n$datetime  → 2025-02-01 14:30\n$weekday   → Friday"
  }
];

export function CardBodyHelpPopover() {
  return (
    <Popover className="relative inline-block ml-2">
      <Popover.Button className="inline-flex items-center justify-center rounded-full bg-blue-100 text-blue-600 w-5 h-5 text-xs font-medium hover:bg-blue-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1" title="Body syntax help">
        <span className="sr-only">Show body syntax help</span>
        ?
      </Popover.Button>

      <Popover.Panel className="absolute z-[60] left-full ml-2 top-0 w-80 max-w-md bg-white rounded-lg shadow-lg border border-gray-200 p-4">
        <div className="text-sm">
          <h3 className="font-semibold text-gray-900 mb-3">Card Body Features</h3>

          <div className="space-y-4">
            {features.map((feature) => (
              <div key={feature.name} className="border-b border-gray-100 pb-3 last:border-0 last:pb-0">
                <div className="font-medium text-gray-800 mb-1">{feature.name}</div>
                <div className="text-gray-600 text-xs mb-2">{feature.description}</div>
                <div className="bg-gray-50 rounded px-2 py-1.5 font-mono text-xs text-gray-700">
                  <div className="text-blue-600 mb-1">{feature.syntax}</div>
                  <div className="text-gray-600 whitespace-pre-wrap">{feature.example}</div>
                </div>
              </div>
            ))}

            <div className="pt-2 border-t border-gray-200">
              <div className="font-medium text-gray-800 mb-1">Template Variables</div>
              <div className="text-gray-600 text-xs mb-2">For use with card templates</div>
              <div className="bg-gray-50 rounded px-2 py-1.5 font-mono text-xs text-gray-700 whitespace-pre-wrap">
                {templateVariables[0].example}
              </div>
            </div>
          </div>
        </div>
      </Popover.Panel>
    </Popover>
  );
}
