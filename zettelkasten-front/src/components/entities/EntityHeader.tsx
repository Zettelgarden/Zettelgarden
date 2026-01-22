import React from "react";
import { Link } from "react-router-dom";
import { Entity } from "../../models/Card";
import { CardIcon } from "../../assets/icons/CardIcon";
import { CardTag } from "../cards/CardTag";
import { Button } from "../Button";

interface EntityHeaderProps {
  entity: Entity;
  onClose: () => void;
  onEdit?: () => void;
  onTurnIntoCard?: () => void;
}

export function EntityHeader({ entity, onClose, onEdit, onTurnIntoCard }: EntityHeaderProps) {
  return (
    <>
      <div className="text-lg font-medium leading-6 text-gray-900 mb-2 flex items-center gap-2 flex-wrap">
        <span>Entity: {entity.name}</span>
        {entity.card && entity.card.id > 0 && (
          <>
            <span className="text-gray-400">→</span>
            <Link
              to={`/app/card/${entity.card.id}`}
              className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800 hover:underline"
              onClick={onClose}
            >
              <div className="w-3 h-3 mr-1 text-gray-400">
                <CardIcon />
              </div>
              <CardTag card={entity.card} showTitle={true} />
            </Link>
          </>
        )}
      </div>

      <div className="mb-4 space-y-2 text-sm">
        {entity.description && (
          <p className="text-gray-700">{entity.description}</p>
        )}
        <div className="text-xs text-gray-500">
          <p>Created: {new Date(entity.created_at).toLocaleDateString()}</p>
          <p>Updated: {new Date(entity.updated_at).toLocaleDateString()}</p>
        </div>
      </div>

      <div className="mt-6 flex justify-end gap-3">
        {!entity.card_pk && onTurnIntoCard && (
          <Button
            onClick={onTurnIntoCard}
            className="bg-green-500 text-white hover:bg-green-600"
          >
            Create Card
          </Button>
        )}
        {onEdit && (
          <Button
            onClick={onEdit}
            className="bg-blue-500 text-white hover:bg-blue-600"
          >
            Edit
          </Button>
        )}
        <Button onClick={onClose}>
          Close
        </Button>
      </div>
    </>
  );
}
