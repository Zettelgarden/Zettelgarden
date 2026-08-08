import React, { useState, useEffect, ChangeEvent } from 'react';
import { Task } from '../../models/Task';
import { Tag } from '../../models/Tags';
import { saveExistingTask } from '../../api/tasks';
import { useTaskContext } from '../../contexts/TaskContext';

interface RemoveTagMenuProps {
  task: Task;
  tags: Tag[];
  setShowRemoveMenu: (showMenu: boolean) => void;
  handleRemoveTag: (tagName: string) => void;
}

export function RemoveTagMenu({
  task,
  tags,
  setShowRemoveMenu,
  handleRemoveTag,
}: RemoveTagMenuProps) {
  function handleExistingTagClick(tag: string) {
    handleRemoveTag('#' + tag);
  }

  useEffect(() => {
    console.log('remove', tags);
  }, []);

  return (
    <div className="w-24">
      {tags &&
        tags.map((tag) => (
          <button onClick={() => handleExistingTagClick(tag.name)}>
            {tag.name}
          </button>
        ))}
    </div>
  );
}
