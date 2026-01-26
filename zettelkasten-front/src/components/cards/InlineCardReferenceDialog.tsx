import React, { useRef, useEffect, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";
import { Z_INDEX } from "../../utils/zIndex";

interface InlineCardReferenceDialogProps {
  position: { top: number; left: number; height: number };
  onSelect: (card: PartialCard) => void;
  onClose: () => void;
  excludeCardId?: number;
}

interface PositionResult {
  top: number;
  left: number;
  width: number;
  maxHeight: number;
}

/**
 * Calculates the optimal position for the dialog based on viewport boundaries
 */
function calculatePosition(
  anchorTop: number,
  anchorLeft: number,
  anchorHeight: number,
  containerRef: React.RefObject<HTMLDivElement>
): PositionResult {
  const PADDING = 8;
  const DEFAULT_WIDTH = 350;
  const MIN_WIDTH = 280;
  const MAX_WIDTH = 450;
  const MAX_HEIGHT = 300;

  // Get viewport dimensions
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;

  // Get dialog dimensions (use default if not yet rendered)
  let dialogWidth = DEFAULT_WIDTH;
  let dialogHeight = MAX_HEIGHT;

  if (containerRef.current) {
    const rect = containerRef.current.getBoundingClientRect();
    dialogWidth = rect.width || DEFAULT_WIDTH;
    dialogHeight = rect.height || MAX_HEIGHT;
  }

  // Calculate preferred position (below the anchor)
  let top = anchorTop + anchorHeight + PADDING;
  let left = anchorLeft;

  // Adjust width based on viewport
  let width = dialogWidth;
  if (viewportWidth < 640) {
    // On small screens, use more width but leave padding
    width = Math.min(MAX_WIDTH, viewportWidth - PADDING * 2);
  } else {
    width = Math.min(MAX_WIDTH, dialogWidth);
  }
  width = Math.max(width, MIN_WIDTH);

  // Check if dialog would overflow bottom of viewport
  if (top + dialogHeight > viewportHeight - PADDING) {
    // Try to position above the anchor
    const topAbove = anchorTop - dialogHeight - PADDING;
    if (topAbove >= PADDING) {
      top = topAbove;
    } else {
      // Not enough space above either, position with max height
      top = Math.max(PADDING, top);
      dialogHeight = viewportHeight - top - PADDING;
    }
  }

  // Check if dialog would overflow right edge of viewport
  if (left + width > viewportWidth - PADDING) {
    left = viewportWidth - width - PADDING;
  }

  // Check if dialog would overflow left edge of viewport
  if (left < PADDING) {
    left = PADDING;
  }

  return {
    top,
    left,
    width,
    maxHeight: dialogHeight,
  };
}

export function InlineCardReferenceDialog({
  position,
  onSelect,
  onClose,
  excludeCardId
}: InlineCardReferenceDialogProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [calculatedPosition, setCalculatedPosition] = useState<PositionResult>(() =>
    calculatePosition(position.top, position.left, position.height, containerRef)
  );
  const [isPositioned, setIsPositioned] = useState(false);

  // Recalculate position on mount and when window is resized
  useEffect(() => {
    const updatePosition = () => {
      setCalculatedPosition(
        calculatePosition(position.top, position.left, position.height, containerRef)
      );
      setIsPositioned(true);
    };

    // Initial calculation
    updatePosition();

    // Recalculate on resize
    const handleResize = () => {
      updatePosition();
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [position]);

  // Close on Escape or Click Outside
  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
      onClose();
    }
  }, [onClose]);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
      event.stopPropagation();
    }
  }, [onClose]);

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleClickOutside, handleKeyDown]);

  const dialogStyle: React.CSSProperties = {
    position: 'fixed',
    top: `${calculatedPosition.top}px`,
    left: `${calculatedPosition.left}px`,
    width: `${calculatedPosition.width}px`,
    zIndex: Z_INDEX.popover,
    maxHeight: `${calculatedPosition.maxHeight}px`,
  };

  const content = (
    <div
      ref={containerRef}
      style={dialogStyle}
      className={`shadow-2xl rounded-lg bg-white border border-gray-200 overflow-hidden
        transition-opacity duration-150 ease-out
        ${isPositioned ? 'opacity-100' : 'opacity-0'}`}
    >
      <BacklinkInputDropdownList
        onSelect={onSelect}
        onSearch={() => {}}
        autoFocus={true}
        excludeCardId={excludeCardId}
        placeholder="Search for card..."
        className="w-full"
      />
    </div>
  );

  // Use Portal to render at document body level for proper layering
  return createPortal(content, document.body);
}
