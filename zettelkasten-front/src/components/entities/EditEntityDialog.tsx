import React, { useState, useEffect } from "react";
import { Dialog } from "@headlessui/react";
import { Entity, PartialCard } from "../../models/Card";
import { UpdateEntityRequest, updateEntity } from "../../api/entities";
import { BacklinkInput } from "../cards/BacklinkInput";
import { CardTag } from "../cards/CardTag";

interface EditEntityDialogProps {
  entity: Entity | null;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  onDelete: (entity: Entity) => void;
}

export function EditEntityDialog({ entity, isOpen, onClose, onSuccess, onDelete }: EditEntityDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState("");
  const [linkedCard, setLinkedCard] = useState<PartialCard | null>(null);
  const [showCardLink, setShowCardLink] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (entity) {
      setName(entity.name);
      setDescription(entity.description);
      setType(entity.type);
      setLinkedCard(entity.card || null);
    }
  }, [entity]);

  const handleBacklink = (card: PartialCard) => {
    setLinkedCard(card);
    setShowCardLink(false);
  };

  const handleRemoveCard = () => {
    setLinkedCard(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!entity) return;

    setError(null);
    setIsSubmitting(true);

    try {
      const data: UpdateEntityRequest = {
        name: name.trim(),
        description: description.trim(),
        type: type.trim(),
        card_pk: linkedCard?.id || null,
      };

      await updateEntity(entity.id, data);
      onSuccess();
      onClose();
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
      
      <Dialog.Panel className="bg-white rounded-xl shadow-xl w-full max-w-2xl mx-auto relative">
        <div className="px-6 py-4 border-b border-gray-200">
          <Dialog.Title className="text-xl font-semibold text-gray-900">
            Edit Entity
          </Dialog.Title>
        </div>

        <form onSubmit={handleSubmit} className="p-6">
          <div className="space-y-6">
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
                Name *
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
                style={{ fontSize: '16px' }}
                required
              />
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={4}
                className="w-full min-h-[100px] max-h-[30vh] sm:max-h-none rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors resize-y"
                style={{ fontSize: '16px' }}
              />
            </div>

            <div>
              <label htmlFor="type" className="block text-sm font-medium text-gray-700 mb-1">
                Type
              </label>
              <input
                type="text"
                id="type"
                value={type}
                onChange={(e) => setType(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
                style={{ fontSize: '16px' }}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Linked Card
              </label>
              {linkedCard ? (
                <div className="flex items-center gap-2 bg-gray-50 p-2 rounded-lg">
                  <CardTag card={linkedCard} showTitle={true} />
                  <button
                    type="button"
                    onClick={handleRemoveCard}
                    className="text-red-600 hover:text-red-800 p-1 hover:bg-red-50 rounded-full transition-colors"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </button>
                </div>
              ) : (
                <div>
                  {showCardLink ? (
                    <BacklinkInput addBacklink={handleBacklink} />
                  ) : (
                    <button
                      type="button"
                      onClick={() => setShowCardLink(true)}
                      className="text-blue-600 hover:text-blue-800 flex items-center gap-1"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                      </svg>
                      Link Card
                    </button>
                  )}
                </div>
              )}
            </div>

            {error && (
              <div className="text-red-600 text-sm bg-red-50 p-3 rounded-lg">
                {error}
              </div>
            )}
          </div>

          <div className="mt-8 pt-6 border-t border-gray-200 flex justify-between">
            <button
              type="button"
              onClick={() => entity && onDelete(entity)}
              className="px-4 py-3 min-h-[44px] bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50"
            >
              Delete Entity
            </button>

            <div className="flex gap-4">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-3 min-h-[44px] text-gray-700 hover:text-gray-900 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSubmitting || !name.trim()}
                className="px-6 py-3 min-h-[44px] bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50"
              >
                {isSubmitting ? "Saving..." : "Save"}
              </button>
            </div>
          </div>
        </form>
      </Dialog.Panel>
    </Dialog>
  );
} 