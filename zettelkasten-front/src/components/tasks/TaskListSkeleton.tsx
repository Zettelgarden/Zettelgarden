import React from "react";

interface TaskListSkeletonProps {
  count?: number;
}

export function TaskListSkeleton({ count = 5 }: TaskListSkeletonProps) {
  return (
    <ul>
      {Array.from({ length: count }).map((_, index) => (
        <li key={index} className="border-b border-slate-200 last:border-0">
          <TaskSkeletonItem />
        </li>
      ))}
    </ul>
  );
}

function TaskSkeletonItem() {
  return (
    <div className="flex items-center bg-white py-2 px-1">
      {/* Checkbox skeleton */}
      <div className="mr-2.5 flex-shrink-0">
        <div className="w-5 h-5 rounded border-2 border-slate-200 bg-slate-100 animate-pulse" />
      </div>

      {/* Content skeleton */}
      <div className="flex-grow min-w-0">
        {/* Title skeleton */}
        <div className="mb-1">
          <div className="h-4 bg-slate-200 rounded animate-pulse w-3/4" />
        </div>

        {/* Metadata skeleton */}
        <div className="flex items-center gap-2">
          {/* Status badge skeleton */}
          <div className="h-5 w-14 bg-slate-200 rounded animate-pulse" />

          {/* Date skeleton */}
          <div className="h-5 w-16 bg-slate-200 rounded animate-pulse" />

          {/* Priority skeleton */}
          <div className="h-5 w-6 bg-slate-200 rounded animate-pulse" />

          {/* Tag skeleton */}
          <div className="h-5 w-12 bg-slate-200 rounded animate-pulse" />
        </div>
      </div>

      {/* Actions skeleton */}
      <div className="ml-2.5 flex-shrink-0">
        <div className="w-6 h-6 bg-slate-200 rounded animate-pulse" />
      </div>
    </div>
  );
}

/**
 * A simpler skeleton for when we just need a basic loading placeholder
 */
export function TaskListSkeletonSimple({ count = 5 }: TaskListSkeletonProps) {
  return (
    <ul className="space-y-2">
      {Array.from({ length: count }).map((_, index) => (
        <li key={index} className="border-b border-slate-200 pb-2">
          <div className="flex items-center gap-3 py-2">
            <div className="w-5 h-5 rounded border-2 border-slate-200 bg-slate-100 animate-pulse" />
            <div className="flex-1 space-y-2">
              <div className="h-4 bg-slate-200 rounded animate-pulse w-2/3" />
              <div className="flex gap-2">
                <div className="h-3 bg-slate-200 rounded animate-pulse w-12" />
                <div className="h-3 bg-slate-200 rounded animate-pulse w-16" />
              </div>
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}
