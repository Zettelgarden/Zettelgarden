import React, { useEffect, createContext, useContext, useState } from 'react';

import { fetchUserTags } from '../api/tags';
import { Tag } from '../models/Tags';

interface TagContextType {
  tags: Tag[];
  setRefreshTags: (refresh: boolean) => void;
}

interface TagProviderProps {
  children: React.ReactNode;
  testing?: boolean;
  testTags?: Tag[];
}

const TagContext = createContext<TagContextType | undefined>(undefined);

export const TagProvider: React.FC<TagProviderProps> = ({
  children,
  testing = false,
  testTags = [],
}) => {
  const [tags, setTags] = useState<Tag[]>([]);

  const [refreshTags, setRefreshTags] = useState(true);

  const getTags = async () => {
    await fetchUserTags().then((data) => {
      // Handle undefined or null data
      if (!data) {
        setTags([]);
        return;
      }
      const sortedTags = data.sort((a, b) => {
        return a.name.localeCompare(b.name);
      });

      setTags(sortedTags);
    });
  };

  useEffect(() => {
    if (testing) {
      setTags(testTags);
      return;
    }
    if (refreshTags) {
      getTags();
      setRefreshTags(false);
    }
  }, [refreshTags]);
  return (
    <TagContext.Provider value={{ tags, setRefreshTags }}>
      {children}
    </TagContext.Provider>
  );
};

export const useTagContext = () => {
  const context = useContext(TagContext);
  if (context === undefined) {
    throw new Error('useTagContext must be used wtihin a TagProvider');
  }
  return context;
};
