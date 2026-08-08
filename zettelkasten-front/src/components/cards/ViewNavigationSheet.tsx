import React from 'react';
import { MobileBottomSheet } from '../layout/MobileBottomSheet';
import { Card, PartialCard } from '../../models/Card';

interface ViewNavigationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  viewingCard: Card;
  onNavigate: (cardId: number) => void;
}

export function ViewNavigationSheet({
  isOpen,
  onClose,
  parentCard,
  prevSibling,
  nextSibling,
  viewingCard,
  onNavigate,
}: ViewNavigationSheetProps) {
  const hasNavigation = parentCard || prevSibling || nextSibling;
  const hasChildren = viewingCard.children && viewingCard.children.length > 0;

  if (!hasNavigation && !hasChildren) {
    return null;
  }

  return (
    <MobileBottomSheet isOpen={isOpen} onClose={onClose} title="Navigate">
      <div className="p-4 space-y-4">
        {/* Parent Card */}
        {parentCard && (
          <button
            onClick={() => {
              onNavigate(parentCard.id);
              onClose();
            }}
            className="w-full p-3 bg-gray-50 rounded-lg text-left hover:bg-gray-100 transition-colors"
          >
            <div className="flex items-center gap-2">
              <span className="text-gray-400">↑</span>
              <div>
                <div className="text-xs text-gray-500">Parent</div>
                <div className="font-medium text-gray-900">
                  {parentCard.title}
                </div>
              </div>
            </div>
          </button>
        )}

        {/* Sibling Navigation */}
        {(prevSibling || nextSibling) && (
          <div className="flex gap-2">
            {prevSibling && (
              <button
                onClick={() => {
                  onNavigate(prevSibling.id);
                  onClose();
                }}
                className="flex-1 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors text-left"
              >
                <span className="text-xs text-gray-500">← Prev</span>
                <div className="text-sm font-medium text-gray-900">
                  {prevSibling.title}
                </div>
              </button>
            )}
            {nextSibling && (
              <button
                onClick={() => {
                  onNavigate(nextSibling.id);
                  onClose();
                }}
                className="flex-1 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors text-left"
              >
                <span className="text-xs text-gray-500">Next →</span>
                <div className="text-sm font-medium text-gray-900">
                  {nextSibling.title}
                </div>
              </button>
            )}
          </div>
        )}

        {/* Children */}
        {hasChildren && (
          <div>
            <div className="text-xs text-gray-500 mb-2">Children</div>
            <div className="space-y-1">
              {viewingCard.children.map((child) => (
                <button
                  key={child.id}
                  onClick={() => {
                    onNavigate(child.id);
                    onClose();
                  }}
                  className="w-full p-2 text-left text-sm text-blue-600 hover:bg-gray-50 rounded"
                >
                  {child.title || 'Untitled'}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </MobileBottomSheet>
  );
}
