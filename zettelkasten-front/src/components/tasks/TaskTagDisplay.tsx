import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";

interface TaskTagDisplayProps {
  task: Task;
  tags: Tag[];
  onTagClick: (tag: string) => void;
  onRemoveTag?: (tag: string) => void;
  hideMatrixTags?: boolean;
}

export function TaskTagDisplay({ task, tags, onTagClick, onRemoveTag, hideMatrixTags = false }: TaskTagDisplayProps) {
  const displayTags = hideMatrixTags
    ? tags.filter(tag => !['important', 'urgent'].includes(tag.name.replace(/^#/, '').toLowerCase()))
    : tags;

  // Check if tag exists in task title
  const tagExistsInTitle = (tagName: string) => {
    const cleanTag = tagName.replace(/^#/, '');
    return task.title.includes(`#${cleanTag}`);
  };

  return (
    <span className="mr-1">
      {displayTags.length > 0 &&
        displayTags.map((tag) => (
          <span
            key={tag.name}
            className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full mr-1"
          >
            <span
              className="cursor-pointer hover:bg-purple-100 min-h-[44px] flex items-center"
              onClick={() => onTagClick(tag.name)}
            >
              {tag.name}
            </span>
            {onRemoveTag && tagExistsInTitle(tag.name) && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onRemoveTag(tag.name);
                }}
                className="ml-1.5 text-purple-400 hover:text-purple-600 min-w-[44px] min-h-[44px] flex items-center justify-center"
              >
                &times;
              </button>
            )}
          </span>
        ))}
    </span>
  );
}
