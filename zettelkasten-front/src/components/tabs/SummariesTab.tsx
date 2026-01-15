import React from "react";
import { SummarizeJobResponse } from "../../api/summarizer";

interface SummariesTabProps {
  summaries: SummarizeJobResponse[] | null;
}

export function SummariesTab({ summaries }: SummariesTabProps) {
  return (
    <div className="p-4">
      <div className="mt-2 space-y-2">
        {summaries && summaries.length > 0 ? (
          summaries.map((s) => (
            <div key={s.id} className="border-b pb-2">
              <div className="text-xs text-gray-500">
                #{s.id} - {s.status}
              </div>
            </div>
          ))
        ) : (
          <div className="text-gray-500">No summaries available</div>
        )}
      </div>
    </div>
  );
}