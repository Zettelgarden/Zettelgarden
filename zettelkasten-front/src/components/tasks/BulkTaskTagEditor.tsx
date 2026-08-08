import React, { useState } from 'react';
import { createPortal } from 'react-dom';
import { Task } from '../../models/Task';
import { saveExistingTask } from '../../api/tasks';
import { useTaskContext } from '../../contexts/TaskContext';
import { useTagContext } from '../../contexts/TagContext';
import { parseTags } from '../../utils/tasks';

interface BulkTaskTagEditorProps {
  tasks: Task[];
  setShowBulkTagEdit: (show: boolean) => void;
}

export function BulkTaskTagEditor({
  tasks,
  setShowBulkTagEdit,
}: BulkTaskTagEditorProps) {
  const { setRefreshTasks } = useTaskContext();
  const { tags } = useTagContext();
  const [operation, setOperation] = useState<'add' | 'remove'>('add');
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const [newTagInput, setNewTagInput] = useState<string>('');
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

  // Get common tags across all selected tasks
  const commonTags = getCommonTags(tasks);

  // Get all unique tags from selected tasks
  const allTagsInSelection = getAllTagsInSelection(tasks);

  async function handleApply() {
    if (selectedTags.size === 0) {
      alert('Please select at least one tag');
      return;
    }

    setIsProcessing(true);

    const promises = tasks.map((task) => {
      let updatedTitle = task.title;

      selectedTags.forEach((tag) => {
        if (operation === 'add') {
          // Add tag if it doesn't exist
          if (!task.title.includes(`#${tag}`)) {
            updatedTitle = `${updatedTitle} #${tag}`;
          }
        } else {
          // Remove tag
          const tagRegex = new RegExp(`\\s*#${tag}\\b`, 'g');
          updatedTitle = updatedTitle.replace(tagRegex, '');
        }
      });

      updatedTitle = updatedTitle.trim();
      const updatedTask = { ...task, title: updatedTitle };
      return saveExistingTask(updatedTask);
    });

    try {
      await Promise.all(promises);
      setRefreshTasks(true);
      setShowBulkTagEdit(false);
    } catch (error) {
      console.error('Failed to bulk update task tags', error);
      alert('Failed to bulk update task tags. Please try again.');
    } finally {
      setIsProcessing(false);
    }
  }

  function toggleTag(tagName: string) {
    setSelectedTags((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(tagName)) {
        newSet.delete(tagName);
      } else {
        newSet.add(tagName);
      }
      return newSet;
    });
  }

  function handleAddNewTag() {
    if (newTagInput.trim()) {
      const cleanTag = newTagInput.replace(/^#/, '').trim();
      toggleTag(cleanTag);
      setNewTagInput('');
    }
  }

  return createPortal(
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white rounded-lg shadow-xl p-6 max-w-md w-full max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold mb-4">
          Bulk Edit Tags ({tasks.length} tasks)
        </h2>

        {/* Operation selector */}
        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">Operation</label>
          <div className="flex gap-2">
            <button
              onClick={() => setOperation('add')}
              className={`flex-1 py-3 min-h-[44px] px-4 rounded ${
                operation === 'add'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 text-gray-700'
              }`}
            >
              Add Tags
            </button>
            <button
              onClick={() => setOperation('remove')}
              className={`flex-1 py-3 min-h-[44px] px-4 rounded ${
                operation === 'remove'
                  ? 'bg-red-600 text-white'
                  : 'bg-gray-200 text-gray-700'
              }`}
            >
              Remove Tags
            </button>
          </div>
        </div>

        {/* New tag input (only for add operation) */}
        {operation === 'add' && (
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2">New Tag</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={newTagInput}
                onChange={(e) => setNewTagInput(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    handleAddNewTag();
                  }
                }}
                placeholder="Enter tag name"
                className="flex-1 px-3 py-2 border border-gray-300 rounded"
              />
              <button
                onClick={handleAddNewTag}
                className="px-4 py-3 min-h-[44px] bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                Add
              </button>
            </div>
          </div>
        )}

        {/* Tag selection area */}
        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">
            {operation === 'add' ? 'Available Tags' : 'Tags to Remove'}
          </label>
          <div className="border border-gray-300 rounded p-3 max-h-48 overflow-y-auto">
            {operation === 'add' ? (
              // Show all existing tags when adding
              tags.length === 0 ? (
                <p className="text-gray-500 text-sm">No tags available</p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {tags.map((tag) => (
                    <button
                      key={tag.id}
                      onClick={() => toggleTag(tag.name.replace(/^#/, ''))}
                      className={`px-3 py-1 rounded-full text-sm ${
                        selectedTags.has(tag.name.replace(/^#/, ''))
                          ? 'bg-blue-600 text-white'
                          : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                      }`}
                    >
                      #{tag.name.replace(/^#/, '')}
                    </button>
                  ))}
                </div>
              )
            ) : // Show only tags from selected tasks when removing
            allTagsInSelection.length === 0 ? (
              <p className="text-gray-500 text-sm">No tags in selected tasks</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {allTagsInSelection.map((tagName) => {
                  const isCommon = commonTags.has(tagName);
                  return (
                    <button
                      key={tagName}
                      onClick={() => toggleTag(tagName)}
                      className={`px-3 py-1 rounded-full text-sm ${
                        selectedTags.has(tagName)
                          ? 'bg-red-600 text-white'
                          : isCommon
                          ? 'bg-purple-100 text-purple-700 hover:bg-purple-200'
                          : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                      }`}
                      title={
                        isCommon
                          ? 'Common across all selected tasks'
                          : 'Present in some tasks'
                      }
                    >
                      #{tagName}
                      {isCommon && ' ✓'}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
          {operation === 'remove' && allTagsInSelection.length > 0 && (
            <p className="text-xs text-gray-500 mt-1">
              Tags with ✓ are present in all selected tasks
            </p>
          )}
        </div>

        {/* Selected tags preview */}
        {selectedTags.size > 0 && (
          <div className="mb-4 p-3 bg-gray-50 rounded">
            <p className="text-sm font-medium mb-1">
              {operation === 'add' ? 'Tags to add:' : 'Tags to remove:'}
            </p>
            <div className="flex flex-wrap gap-1">
              {Array.from(selectedTags).map((tag) => (
                <span key={tag} className="text-xs bg-white px-2 py-1 rounded">
                  #{tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-2 justify-end">
          <button
            onClick={() => setShowBulkTagEdit(false)}
            disabled={isProcessing}
            className="px-4 py-3 min-h-[44px] border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleApply}
            disabled={selectedTags.size === 0 || isProcessing}
            className={`px-4 py-3 min-h-[44px] rounded text-white disabled:opacity-50 ${
              operation === 'add'
                ? 'bg-blue-600 hover:bg-blue-700'
                : 'bg-red-600 hover:bg-red-700'
            }`}
          >
            {isProcessing ? 'Processing...' : `Apply to ${tasks.length} tasks`}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}

// Helper functions
function getCommonTags(tasks: Task[]): Set<string> {
  if (tasks.length === 0) return new Set();

  // Start with tags from first task
  const commonTags = new Set(
    parseTags(tasks[0].title).map((t) => t.replace(/^#/, '')),
  );

  // Keep only tags present in ALL tasks
  for (let i = 1; i < tasks.length; i++) {
    const taskTags = new Set(
      parseTags(tasks[i].title).map((t) => t.replace(/^#/, '')),
    );

    // Remove tags not in this task
    commonTags.forEach((tag) => {
      if (!taskTags.has(tag)) {
        commonTags.delete(tag);
      }
    });
  }

  return commonTags;
}

function getAllTagsInSelection(tasks: Task[]): string[] {
  const allTags = new Set<string>();

  tasks.forEach((task) => {
    parseTags(task.title).forEach((tag) => {
      allTags.add(tag.replace(/^#/, ''));
    });
  });

  return Array.from(allTags).sort();
}
