import React from 'react';
import { Menu } from '@headlessui/react';
import { Entity } from '../../models/Card';

interface EntityWithScore extends Entity {
  score: number;
}

interface EntitySimilarSectionProps {
  similarEntities: EntityWithScore[];
  isLoading: boolean;
  error: string | null;
  currentEntityName: string;
  onEntityClick: (entity: Entity) => void;
  onInitiateMerge: (
    entity: Entity,
    direction: 'into-current' | 'from-current',
  ) => void;
}

export function EntitySimilarSection({
  similarEntities,
  isLoading,
  error,
  currentEntityName,
  onEntityClick,
  onInitiateMerge,
}: EntitySimilarSectionProps) {
  return (
    <>
      <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">
        Similar Entities:
      </h4>
      <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
        {isLoading && <p>Loading similar entities...</p>}
        {error && <p className="text-red-600">{error}</p>}
        {!isLoading && similarEntities && similarEntities.length === 0 && (
          <p>No similar entities.</p>
        )}
        {!isLoading && similarEntities && similarEntities.length > 0 && (
          <ul className="space-y-1 text-sm">
            {similarEntities.map((e) => (
              <li
                key={e.id}
                className="flex items-center justify-between hover:bg-gray-100 p-1 rounded"
              >
                <span
                  onClick={() => onEntityClick(e)}
                  className="text-gray-700 cursor-pointer flex-grow"
                >
                  • {e.name}
                </span>
                <div className="flex items-center gap-2">
                  <span
                    className={`text-xs px-2 py-0.5 rounded ${
                      e.score >= 0.8
                        ? 'bg-green-100 text-green-700'
                        : e.score >= 0.5
                        ? 'bg-yellow-100 text-yellow-700'
                        : 'bg-gray-100 text-gray-600'
                    }`}
                    title="Similarity score"
                  >
                    {Math.round(e.score * 100)}%
                  </span>
                  <Menu as="div" className="relative inline-block text-left">
                    <div>
                      <Menu.Button className="inline-flex justify-center w-full rounded-md border border-gray-300 shadow-sm px-3 py-1 bg-white text-xs font-medium text-gray-700 hover:bg-gray-50 focus:outline-none">
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="h-5"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                        >
                          <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                        </svg>
                      </Menu.Button>
                    </div>
                    <Menu.Items className="absolute right-0 w-56 mt-2 origin-top-right bg-white divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                      <div className="px-1 py-1 ">
                        <Menu.Item>
                          {({ active }) => (
                            <button
                              onClick={() => onInitiateMerge(e, 'into-current')}
                              className={`${
                                active
                                  ? 'bg-blue-500 text-white'
                                  : 'text-gray-900'
                              } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                            >
                              Merge '{e.name}' into '{currentEntityName}'
                            </button>
                          )}
                        </Menu.Item>
                        <Menu.Item>
                          {({ active }) => (
                            <button
                              onClick={() => onInitiateMerge(e, 'from-current')}
                              className={`${
                                active
                                  ? 'bg-blue-500 text-white'
                                  : 'text-gray-900'
                              } group flex rounded-md items-center w-full px-2 py-2 text-sm`}
                            >
                              Merge '{currentEntityName}' into '{e.name}'
                            </button>
                          )}
                        </Menu.Item>
                      </div>
                    </Menu.Items>
                  </Menu>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
