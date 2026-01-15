import React from "react";
import { Card, Entity } from "../../models/Card";
import { AddEntityDialog } from "../entities/AddEntityDialog";

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
          <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
          Add Entity
        </button>
      </div>
      {viewingCard.entities && viewingCard.entities.length > 0 ? (
        <div className="max-h-[500px] overflow-y-auto space-y-1">
          {viewingCard.entities
            .filter((entity) =>
              entity.name.toLowerCase().includes(entityFilterString.toLowerCase()) ||
              entity.description?.toLowerCase().includes(entityFilterString.toLowerCase()) ||
              entity.type?.toLowerCase().includes(entityFilterString.toLowerCase())
            )
            .sort((a, b) => a.name.localeCompare(b.name))
            .map((entity) => (
              <div
                key={entity.id}
                className="mb-1 p-3 hover:bg-gray-50 border border-gray-200 rounded-lg flex justify-between items-start group transition-colors"
              >
                <div
                  className="cursor-pointer flex-grow min-w-0"
                  onClick={() => handleOpenEntity(entity)}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium text-gray-900 truncate">{entity.name}</span>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 shrink-0">
                      {entity.type}
                    </span>
                  </div>
                  {entity.description && (
                    <div className="text-sm text-gray-600 line-clamp-2">{entity.description}</div>
                  )}
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleRemoveEntity(entity.id);
                  }}
                  className="ml-3 p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md opacity-0 group-hover:opacity-100 transition-all shrink-0"
                  title="Remove entity from card"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
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
        linkedEntityIds={viewingCard.entities?.map(e => e.id) || []}
        onClose={() => setShowAddEntityDialog(false)}
        onEntityAdded={handleEntityAdded}
        onError={setError}
      />
    </div>
  );
}