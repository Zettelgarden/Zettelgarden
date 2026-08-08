import React from 'react';
import ReactMarkdown from 'react-markdown';
import { SummarizeJobResponse } from '../../api/summarizer';

interface ViewSummaryViewProps {
  /** The most recent summary with a result to render prominently. */
  summary: SummarizeJobResponse | null;
  /** The full list of past summarization jobs, shown as a history. */
  summaries?: SummarizeJobResponse[] | null;
}

export function ViewSummaryView({ summary, summaries }: ViewSummaryViewProps) {
  const hasLatest = !!summary?.result;
  const history = (summaries ?? []).filter((s) => s.id !== summary?.id);

  return (
    <div className="space-y-4">
      {!hasLatest && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
          <div className="text-yellow-800 text-center">
            <p className="font-medium">No summary available</p>
            <p className="text-sm mt-2">Generate a summary to see it here.</p>
          </div>
        </div>
      )}

      {hasLatest && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 shadow-sm">
          <div className="bg-yellow-100 text-yellow-800 font-semibold px-4 py-2 rounded-md mb-4">
            Summary View
          </div>
          <div className="prose prose-sm max-w-none">
            <ReactMarkdown>{summary!.result}</ReactMarkdown>
          </div>
        </div>
      )}

      {history.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-gray-700 mb-2">
            Past summaries
          </h3>
          <ul className="space-y-1">
            {history.map((s) => (
              <li key={s.id} className="text-xs text-gray-500">
                #{s.id} - {s.status}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
