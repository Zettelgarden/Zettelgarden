import React from 'react';
import { Card, Entity } from '../../models/Card';
import { AddEntityDialog } from '../entities/AddEntityDialog';

interface EntitiesTabProps {
  viewingCard: Card;
  entityFilterString: string;
  setEntityFilterString: (value: string) => void;
  showAddEntityDialog: boolean;
  setShowAddEntityDialog: (value: boolean) => void;
  handleOpenEntity: (entity: Entity) => void;
  handleRemoveEntity: (entityId: number) => void;
  handleEntityAdded: (entity: Entity) => void;
  setError: (error: string) => void;
}

export function EntitiesTab({
  viewingCard,
  entityFilterString,
  setEntityFilterString,
  showAddEntityDialog,
  setShowAddEntityDialog,
  handleOpenEntity,
  handleRemoveEntity,
  handleEntityAdded,
  setError,
}: EntitiesTabProps) {
  return (
    <div className="p-4">
      <div className="mb-4 flex gap-2">
        <input
          type="text"
          placeholder="Filter entities..."
          value={entityFilterString}
          onChange={(e) => setEntityFilterString(e.target.value)}
          className="flex-1 p-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          onClick={() => setShowAddEntityDialog(true)}
          className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 flex items-center gap-2"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="h-4 w-4"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fillRule="evenodd"
              d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
              clipRule="evenodd"
            />
          </svg>
          Add Entity
        </button>
      </div>
      {viewingCard.entities && viewingCard.entities.length > 0 ? (
        <div className="max-h-[500px] overflow-y-auto space-y-1">
          {viewingCard.entities
            .filter(
              (entity) =>
                entity.name
                  .toLowerCase()
                  .includes(entityFilterString.toLowerCase()) ||
                entity.description
                  ?.toLowerCase()
                  .includes(entityFilterString.toLowerCase()) ||
                entity.type
                  ?.toLowerCase()
                  .includes(entityFilterString.toLowerCase()),
            )
            .sort((a, b) => a.name.localeCompare(b.name))
            .map((entity) => (
              <div
                key={entity.id}
                className="py-1.5 px-2 hover:bg-gray-50 border border-gray-200 rounded flex justify-between items-center group transition-colors"
              >
                <div
                  className="cursor-pointer flex-grow min-w-0 flex items-center gap-2 text-sm"
                  onClick={() => handleOpenEntity(entity)}
                >
                  <span className="font-medium text-blue-600 hover:text-blue-800 shrink-0 truncate">
                    {entity.name}
                  </span>
                  <span className="text-gray-300 shrink-0">-</span>
                  <span className="text-gray-500 shrink-0 text-xs">
                    {entity.type}
                  </span>
                  <span className="text-gray-300 shrink-0">-</span>
                  <span className="text-gray-600 truncate text-xs">
                    {entity.description || '(no description)'}
                  </span>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleRemoveEntity(entity.id);
                  }}
                  className="ml-2 p-2 min-w-[44px] min-h-[44px] flex items-center justify-center text-gray-400 hover:text-red-600 hover:bg-red-50 rounded opacity-0 group-hover:opacity-100 transition-all shrink-0"
                  title="Remove entity from card"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-3.5 w-3.5"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path
                      fillRule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clipRule="evenodd"
                    />
                  </svg>
                </button>
              </div>
            ))}
        </div>
      ) : (
        <div className="text-gray-500">No entities available</div>
      )}

      <AddEntityDialog
        isOpen={showAddEntityDialog}
        cardId={viewingCard.id}
        linkedEntityIds={viewingCard.entities?.map((e) => e.id) || []}
        onClose={() => setShowAddEntityDialog(false)}
        onEntityAdded={handleEntityAdded}
        onError={setError}
      />
    </div>
  );
}
