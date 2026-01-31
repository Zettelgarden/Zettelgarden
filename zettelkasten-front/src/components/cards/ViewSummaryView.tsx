import React from "react";
import ReactMarkdown from "react-markdown";
import { SummarizeJobResponse } from "../../api/summarizer";

interface ViewSummaryViewProps {
  summary: SummarizeJobResponse | null;
}

export function ViewSummaryView({ summary }: ViewSummaryViewProps) {
  if (!summary?.result) {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
        <div className="text-yellow-800 text-center">
          <p className="font-medium">No summary available</p>
          <p className="text-sm mt-2">Generate a summary to see it here.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 shadow-sm">
      <div className="bg-yellow-100 text-yellow-800 font-semibold px-4 py-2 rounded-md mb-4">
        Summary View
      </div>
      <div className="prose prose-sm max-w-none">
        <ReactMarkdown>{summary.result}</ReactMarkdown>
      </div>
    </div>
  );
}
