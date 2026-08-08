import React, { useState } from 'react';
import { Task } from '../../models/Task';
import { BulkTaskDateDisplay } from './BulkTaskDateDisplay';
import { BulkTaskTagEditor } from './BulkTaskTagEditor';
import { Button } from '../Button';

interface TaskSelectionOverlayProps {
  tasks: Task[];
  selectMode: boolean;
  selectedTaskIds: Set<number>;
  onSelectAll: () => void;
  onClearSelection: () => void;
  onToggleSelectMode: () => void;
}

export function TaskSelectionOverlay({
  tasks,
  selectMode,
  selectedTaskIds,
  onSelectAll,
  onClearSelection,
  onToggleSelectMode,
}: TaskSelectionOverlayProps) {
  const [showBulkEdit, setShowBulkEdit] = useState<boolean>(false);
  const [showBulkTagEdit, setShowBulkTagEdit] = useState<boolean>(false);

  if (!selectMode) return null;

  return (
    <div className="fixed bottom-6 left-1/2 transform -translate-x-1/2 bg-white border border-slate-300 shadow-2xl rounded-lg p-3 z-50 flex flex-wrap items-center justify-center gap-3 animate-in fade-in slide-in-from-bottom-4 duration-200 backdrop-blur-sm bg-opacity-95 safe-bottom-fixed">
      <div className="text-sm font-semibold text-slate-700 whitespace-nowrap px-1">
        {selectedTaskIds.size} selected
      </div>

      <div className="hidden sm:block h-6 w-px bg-slate-300 mx-1" />

      <div className="flex flex-wrap justify-center gap-2">
        <Button
          onClick={() => setShowBulkTagEdit(true)}
          disabled={selectedTaskIds.size === 0}
          className="text-sm px-4 py-3 min-h-[44px] bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 shadow-sm border-transparent"
        >
          Edit Tags
        </Button>
        <Button
          onClick={() => setShowBulkEdit(true)}
          disabled={selectedTaskIds.size === 0}
          className="text-sm px-4 py-3 min-h-[44px] bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 shadow-sm border-transparent"
        >
          Edit Date
        </Button>
        <div className="w-px h-6 bg-slate-300 mx-1 self-center hidden sm:block" />
        <Button
          onClick={onSelectAll}
          variant="outline"
          className="text-sm px-4 py-3 min-h-[44px] bg-white hover:bg-slate-50 shadow-sm"
        >
          Select All
        </Button>
        <Button
          onClick={onClearSelection}
          variant="outline"
          disabled={selectedTaskIds.size === 0}
          className="text-sm px-4 py-3 min-h-[44px] bg-transparent hover:bg-slate-100 border-transparent shadow-none"
        >
          Clear
        </Button>
      </div>

      <div className="hidden sm:block h-6 w-px bg-slate-300 mx-1" />

      <Button
        onClick={onToggleSelectMode}
        variant="outline"
        className="text-sm px-4 py-3 min-h-[44px] bg-red-50 hover:bg-red-100 text-red-700 border-red-200 font-medium"
      >
        Done
      </Button>

      {showBulkEdit && (
        <BulkTaskDateDisplay
          tasks={tasks.filter((t) => selectedTaskIds.has(t.id))}
          setShowBulkEdit={setShowBulkEdit}
        />
      )}
      {showBulkTagEdit && (
        <BulkTaskTagEditor
          tasks={tasks.filter((t) => selectedTaskIds.has(t.id))}
          setShowBulkTagEdit={setShowBulkTagEdit}
        />
      )}
    </div>
  );
}
