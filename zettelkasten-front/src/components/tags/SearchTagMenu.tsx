import React from 'react';

import { Tag } from '../../models/Tags';
import { SearchTagDropdown } from './SearchTagDropdown';

interface SearchTagMenuProps {
  tags: Tag[];
  handleTagClick: (tag: string) => void;
}
export function SearchTagMenu({ tags, handleTagClick }: SearchTagMenuProps) {
  return <SearchTagDropdown tags={tags} handleTagClick={handleTagClick} />;
}
