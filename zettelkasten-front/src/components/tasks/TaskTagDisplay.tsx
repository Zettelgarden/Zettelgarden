import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";

interface TaskTagDisplayProps {
  task: Task;
  tags: Tag[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
}

export function TaskTagDisplay({ task, tags, onTagClick, hideMatrixTags = false }: TaskTagDisplayProps) {
  const displayTags = hideMatrixTags
    ? tags.filter(tag => !['important', 'urgent'].includes(tag.name.replace(/^#/, '').toLowerCase()))
    : tags;

  return (
    <span className="mr-1">
      {displayTags.length > 0 &&
        displayTags.map((tag) => (
          <span
            key={tag.name}
            className="inline-block text-purple-500 text-xs px-2 cursor-pointer"
            onClick={() => onTagClick(tag.name)}
          >
            {tag.name}
          </span>
        ))}
    </span>
  );
}
