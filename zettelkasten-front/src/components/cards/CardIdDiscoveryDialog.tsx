import React, { useState, useEffect } from "react";
import { Dialog } from "@headlessui/react";
import { PartialCard } from "../../models/Card";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { CardTag } from "./CardTag";
import { Button } from "../Button";
import { getCardChildren, getNextRootId } from "../../api/cards";

interface CardIdDiscoveryDialogProps {
  onClose: () => void;
  onSelectId: (cardId: string) => void;
}

export function CardIdDiscoveryDialog({ onClose, onSelectId }: CardIdDiscoveryDialogProps) {
  const [currentCard, setCurrentCard] = useState<PartialCard | null>(null);
  const [children, setChildren] = useState<PartialCard[]>([]);
  const [loadingChildren, setLoadingChildren] = useState(false);
  const [childrenError, setChildrenError] = useState<string | null>(null);
  const [breadcrumb, setBreadcrumb] = useState<PartialCard[]>([]);

  useEffect(() => {
    if (currentCard) {
      setLoadingChildren(true);
      setChildrenError(null);
      getCardChildren(currentCard.id.toString())
        .then((allChildren) => {
          // Filter to only show direct children (where parent_id matches current card's id)
          const directChildren = allChildren.filter(child => child.parent_id === currentCard.id);
          setChildren(directChildren);
        })
        .catch(() => setChildrenError("Failed to load children"))
        .finally(() => setLoadingChildren(false));
    } else {
      setChildren([]);
    }
  }, [currentCard]);

  function handleCardSelect(card: PartialCard) {
    setCurrentCard(card);
    setBreadcrumb([card]);
  }

  function handleChildClick(child: PartialCard) {
    setCurrentCard(child);
    setBreadcrumb(prev => [...prev, child]);
  }

  function handleBreadcrumbClick(card: PartialCard, index: number) {
    setCurrentCard(card);
    setBreadcrumb(prev => prev.slice(0, index + 1));
  }

  async function handleAddChild(parentCard: PartialCard) {
    try {
      // Get children of the parent card to generate the correct child ID
      const parentChildren = await getCardChildren(parentCard.id.toString());
      const directChildren = parentChildren.filter(child => child.parent_id === parentCard.id);

      // Generate a suggested child ID pattern
      const suggestedId = generateChildId(parentCard.card_id, directChildren);
      onSelectId(suggestedId);
      onClose();
    } catch (error) {
      console.error("Failed to get children for ID generation:", error);
      // Fallback to simple pattern if API call fails
      const suggestedId = `${parentCard.card_id}.1`;
      onSelectId(suggestedId);
      onClose();
    }
  }

  function generateChildId(parentId: string, existingChildren: PartialCard[]): string {
    // Find the highest numbered child and increment
    // Supports both old format (alternating separators) and new format (any separators)

    if (existingChildren.length === 0) {
      // No children exist, start with 1
      return `${parentId}.1`;
    }

    // Extract numeric suffixes from all children
    const childNumbers: number[] = [];

    for (const child of existingChildren) {
      const childId = child.card_id;

      // Make sure this is actually a direct child by checking it starts with parent ID
      if (!childId.startsWith(parentId)) {
        continue;
      }

      // Get the part after the parent ID
      const suffix = childId.substring(parentId.length);

      // Extract the first number after any separator
      const match = suffix.match(/^[.\/-]+(\d+)/);
      if (match) {
        const num = parseInt(match[1], 10);
        if (!isNaN(num)) {
          childNumbers.push(num);
        }
      }
    }

    if (childNumbers.length === 0) {
      // No numbered children found, start with 1
      return `${parentId}.1`;
    }

    // Find the highest number and increment
    const maxNumber = Math.max(...childNumbers);
    return `${parentId}.${maxNumber + 1}`;
  }

  async function handleGetNextId() {
    try {
      const response = await getNextRootId();
      if (!response.error) {
        onSelectId(response.new_id);
        onClose();
      }
    } catch (error) {
      console.error("Failed to get next ID:", error);
    }
  }

  return (
    <Dialog open={true} onClose={onClose} className="relative z-[90]">
      <div className="fixed inset-0 bg-black/30" aria-hidden="true" />

      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel className="w-full max-w-2xl transform max-h-[90vh] rounded-2xl bg-white p-6 shadow-xl transition-all flex flex-col">
          <Dialog.Title className="text-lg font-medium leading-6 text-gray-900 mb-4">
            Discover Card ID
          </Dialog.Title>

          <div className="overflow-y-auto flex-1">
            <div className="space-y-4">
              {!currentCard ? (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Search for a card to explore:
                  </label>
                  <BacklinkInputDropdownList
                    onSelect={handleCardSelect}
                    onSearch={() => {}}
                    placeholder="Search cards..."
                    className="w-full"
                  />
                </div>
              ) : (
                <div className="space-y-4">
                  {/* Breadcrumb */}
                  <div className="flex items-center gap-2 text-sm text-gray-600">
                    <span>Navigation:</span>
                    {breadcrumb.map((card, index) => (
                      <React.Fragment key={card.id}>
                        <button
                          onClick={() => handleBreadcrumbClick(card, index)}
                          className="text-blue-600 hover:text-blue-800 hover:underline"
                        >
                          [{card.card_id}]
                        </button>
                        {index < breadcrumb.length - 1 && <span>→</span>}
                      </React.Fragment>
                    ))}
                  </div>

                  {/* Current Card */}
                  <div className="border-t pt-4">
                    <div className="flex items-center justify-between mb-4">
                      <h4 className="text-md font-medium text-gray-800">
                        Current Card:
                      </h4>
                      <Button
                        onClick={() => handleAddChild(currentCard)}
                        variant="primary"
                        size="small"
                      >
                        Add Child
                      </Button>
                    </div>
                    <div className="bg-blue-50 p-3 rounded-lg mb-4">
                      <CardTag card={currentCard} showTitle={true} />
                    </div>

                    {/* Children */}
                    <h4 className="text-md font-medium text-gray-800 mb-2">
                      Children:
                    </h4>
                    <div className="min-h-[100px] max-h-[300px] overflow-y-auto border rounded-lg bg-gray-50">
                      {loadingChildren && (
                        <p className="p-3 text-gray-500">Loading children...</p>
                      )}
                      {childrenError && (
                        <p className="p-3 text-red-600">{childrenError}</p>
                      )}
                      {!loadingChildren && children.length === 0 && !childrenError && (
                        <p className="p-3 text-gray-500">No children found.</p>
                      )}
                      {!loadingChildren && children.length > 0 && (
                        <div className="p-2 space-y-1">
                          {children.map((child) => (
                            <div
                              key={child.id}
                              className="flex items-center justify-between p-2 hover:bg-blue-50 rounded transition-colors group"
                            >
                              <div
                                onClick={() => handleChildClick(child)}
                                className="cursor-pointer flex-1"
                              >
                                <CardTag card={child} showTitle={true} />
                              </div>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleAddChild(child);
                                }}
                                className="opacity-0 group-hover:opacity-100 transition-opacity ml-2 px-2 py-1 text-xs bg-white border border-gray-300 rounded hover:bg-gray-50"
                              >
                                Add Child
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>

                    {/* Back button */}
                    <div className="mt-4">
                      <Button
                        onClick={() => {
                          setCurrentCard(null);
                          setBreadcrumb([]);
                        }}
                        variant="outline"
                      >
                        ← Start Over
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="mt-6 flex justify-between items-center border-t pt-4">
            <Button onClick={handleGetNextId} variant="outline">
              Get Next ID
            </Button>
            <div className="flex gap-3">
              <Button onClick={onClose} variant="outline">
                Cancel
              </Button>
            </div>
          </div>
        </Dialog.Panel>
      </div>
    </Dialog>
  );
}