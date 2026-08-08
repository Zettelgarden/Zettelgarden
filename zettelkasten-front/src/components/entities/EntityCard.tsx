import React from 'react';
import { Entity } from '../../models/Card';
import { Link } from 'react-router-dom';
import { CardTag } from '../cards/CardTag';

interface EntityCardProps {
  entity: Entity;
  isSelected: boolean;
  selectionInfo: string | null;
  selectionMode: boolean;
  onEdit: (entity: Entity, event: React.MouseEvent) => void;
  onClick: (entity: Entity, event: React.MouseEvent) => void;
}

export function EntityCard({
  entity,
  isSelected,
  selectionInfo,
  selectionMode,
  onEdit,
  onClick,
}: EntityCardProps) {
  return (
    <div
      className={`bg-white rounded-lg shadow-md p-4 hover:shadow-lg transition-all cursor-pointer
        ${
          isSelected
            ? 'ring-2 ring-blue-500 bg-blue-50'
            : selectionMode
            ? 'hover:ring-2 hover:ring-blue-200'
            : ''
        }`}
      onClick={(e) => onClick(entity, e)}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {selectionMode && (
            <div
              className={`w-5 h-5 border-2 rounded-md flex items-center justify-center transition-colors
                ${
                  isSelected
                    ? 'border-blue-500 bg-blue-500'
                    : 'border-gray-300 bg-white hover:border-blue-400'
                }`}
            >
              {isSelected && (
                <svg
                  className="w-3 h-3 text-white"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path
                    fillRule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
            </div>
          )}
          <h3
            className={`text-lg font-semibold ${
              isSelected ? 'text-blue-800' : 'text-gray-800'
            }`}
          >
            {entity.name}
          </h3>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => onEdit(entity, e)}
            className="text-gray-600 hover:text-blue-600 p-1 rounded-full hover:bg-gray-100"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
            </svg>
          </button>
          <span className="text-sm px-2 py-1 bg-gray-100 rounded-full text-gray-600">
            {entity.type}
          </span>
        </div>
      </div>

      <p className="text-gray-600 text-sm mb-2">{entity.description}</p>

      <div className="flex justify-between items-center text-xs text-gray-500">
        <span>Cards: {entity.card_count}</span>
        {entity.card && (
          <Link
            to={`/app/card/${entity.card.id}`}
            onClick={(e) => e.stopPropagation()}
            className="text-blue-600 hover:text-blue-800"
          >
            <CardTag card={entity.card} showTitle={false} />
          </Link>
        )}
      </div>

      {selectionInfo && (
        <div
          className={`mt-2 text-sm font-medium ${
            selectionInfo === 'Primary'
              ? 'text-green-600 bg-green-50 px-2 py-1 rounded-md inline-block'
              : 'text-blue-600'
          }`}
        >
          {selectionInfo}
        </div>
      )}
    </div>
  );
}
