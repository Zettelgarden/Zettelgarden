import React, { useRef, useEffect } from "react";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";

interface InlineCardReferenceDialogProps {
  position: { top: number; left: number; height: number };
  onSelect: (card: PartialCard) => void;
  onClose: () => void;
  excludeCardId?: number;
}

export function InlineCardReferenceDialog({
  position,
  onSelect,
  onClose,
  excludeCardId
}: InlineCardReferenceDialogProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on Escape or Click Outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        onClose();
      }
    }

    // We capture keydown at document level to ensure Escape always works
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose();
        // Prevent default behavior to avoid bubbling up to other handlers
        event.stopPropagation();
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose]);

  const style: React.CSSProperties = {
    position: 'absolute',
    top: position.top + position.height,
    left: position.left,
    zIndex: 50,
    width: '350px',
  };

  return (
    <div ref={containerRef} style={style} className="shadow-2xl rounded-lg bg-white border border-gray-200">
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
}
