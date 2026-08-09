import React, { useState, useEffect } from 'react';
import { Modal } from '../ui/Modal';
import { Entity } from '../../models/Card';
import {
  addEntityToCard,
  fetchEntities,
  EntityListResponse,
} from '../../api/entities';

interface AddEntityDialogProps {
  isOpen: boolean;
  cardId: number;
  linkedEntityIds: number[];
  onClose: () => void;
  onEntityAdded: (entity: Entity) => void;
  onError: (error: string) => void;
}

export function AddEntityDialog({
  isOpen,
  cardId,
  linkedEntityIds,
  onClose,
  onEntityAdded,
  onError,
}: AddEntityDialogProps) {
  const [availableEntities, setAvailableEntities] = useState<Entity[]>([]);
  const [entitySearchTerm, setEntitySearchTerm] = useState<string>('');
  const [isLoadingEntities, setIsLoadingEntities] = useState<boolean>(false);

  async function handleAddEntity(entity: Entity) {
    try {
      await addEntityToCard(entity.id, cardId);
      onEntityAdded(entity);
      setEntitySearchTerm('');
      onClose();
    } catch (error) {
      onError('Failed to add entity to card');
    }
  }

  async function searchAvailableEntities(searchTerm: string) {
    if (!searchTerm.trim()) {
      setAvailableEntities([]);
      return;
    }

    setIsLoadingEntities(true);
    try {
      const response: EntityListResponse = await fetchEntities({
        search: searchTerm.trim(),
        per_page: 20,
      });
      // Filter out entities that are already linked to this card
      const linkedEntityIdsSet = new Set(linkedEntityIds);
      const filteredEntities = response.entities.filter(
        (entity) => !linkedEntityIdsSet.has(entity.id),
      );
      setAvailableEntities(filteredEntities);
    } catch (error) {
      onError('Failed to search entities');
      setAvailableEntities([]);
    } finally {
      setIsLoadingEntities(false);
    }
  }

  useEffect(() => {
    if (isOpen) {
      searchAvailableEntities(entitySearchTerm);
    }
  }, [entitySearchTerm, isOpen, linkedEntityIds]);

  // Reset state when dialog closes
  useEffect(() => {
    if (!isOpen) {
      setEntitySearchTerm('');
      setAvailableEntities([]);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <Modal open={isOpen} onClose={onClose} size="md" dialogClassName="z-50">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">Add Entity to Card</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <svg
            className="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      <div className="mb-4">
        <input
          type="text"
          placeholder="Search entities..."
          value={entitySearchTerm}
          onChange={(e) => setEntitySearchTerm(e.target.value)}
          className="w-full p-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          autoFocus
        />
      </div>

      <div className="max-h-60 overflow-y-auto">
        {isLoadingEntities ? (
          <div className="text-center py-4 text-gray-500">
            Loading entities...
          </div>
        ) : availableEntities.length > 0 ? (
          <div className="space-y-1">
            {availableEntities.map((entity) => (
              <div
                key={entity.id}
                onClick={() => handleAddEntity(entity)}
                className="p-3 border border-gray-200 rounded-lg cursor-pointer hover:bg-gray-50 transition-colors"
              >
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-medium text-gray-900">
                    {entity.name}
                  </span>
                  <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    {entity.type}
                  </span>
                </div>
                {entity.description && (
                  <div className="text-sm text-gray-600 line-clamp-2">
                    {entity.description}
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : entitySearchTerm.trim() ? (
          <div className="text-center py-4 text-gray-500">
            No entities found matching "{entitySearchTerm}"
          </div>
        ) : (
          <div className="text-center py-4 text-gray-500">
            Type to search for entities
          </div>
        )}
      </div>
    </Modal>
  );
}
