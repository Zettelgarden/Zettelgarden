import React, { useState, useEffect } from 'react';
import { getJobHistory, JobRun } from '../../api/admin';
import { ExecutionHistoryTable } from './ExecutionHistoryTable';

interface ExpandableHistoryProps {
  jobName: string;
  isExpanded: boolean;
}

export function ExpandableHistory({
  jobName,
  isExpanded,
}: ExpandableHistoryProps) {
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  // Load history when first expanded
  useEffect(() => {
    if (isExpanded && runs.length === 0) {
      loadHistory(0);
    }
  }, [isExpanded]);

  const loadHistory = async (newOffset: number) => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await getJobHistory(jobName, 50, newOffset);
      if (newOffset === 0) {
        setRuns(response.runs);
      } else {
        setRuns((prev) => [...prev, ...response.runs]);
      }
      setOffset(newOffset + response.runs.length);
      setHasMore(response.has_more);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLoadMore = () => {
    loadHistory(offset);
  };

  if (!isExpanded) {
    return null;
  }

  return (
    <div className="mt-4 pl-4 border-l-2 border-gray-200">
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 text-sm rounded">
          {error}
          <button
            onClick={() => loadHistory(0)}
            className="ml-2 underline hover:no-underline"
          >
            Retry
          </button>
        </div>
      )}

      {isLoading && runs.length === 0 ? (
        <div className="py-8 text-center text-gray-500 text-sm">
          Loading job history...
        </div>
      ) : (
        <>
          <ExecutionHistoryTable runs={runs} />

          {hasMore && !isLoading && (
            <div className="mt-4 text-center">
              <button
                onClick={handleLoadMore}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg text-sm font-medium transition-colors"
              >
                Load more
              </button>
            </div>
          )}

          {isLoading && runs.length > 0 && (
            <div className="mt-4 text-center text-sm text-gray-500">
              Loading more...
            </div>
          )}
        </>
      )}
    </div>
  );
}
