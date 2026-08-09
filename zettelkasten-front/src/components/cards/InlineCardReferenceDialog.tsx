import React from 'react';
import { Popover } from '../ui/Popover';
import { BacklinkInputDropdownList } from './BacklinkInputDropdownList';
import { PartialCard } from '../../models/Card';
import { Z_INDEX } from '../../utils/zIndex';

interface InlineCardReferenceDialogProps {
  /** Viewport position of the reference anchor (a card reference in the body) */
  position: { top: number; left: number; height: number };
  onSelect: (card: PartialCard) => void;
  onClose: () => void;
  excludeCardId?: number;
}

/**
 * Anchor-positioned card-search popover. The parent mounts it open at a
 * viewport position (the reference in the body text); ui/Popover anchors the
 * panel to an invisible fixed trigger at that point (Headless/floating-ui
 * handles viewport-flipping + repositioning on scroll/resize, Escape and
 * click-outside).
 */
export function InlineCardReferenceDialog({
  position,
  onSelect,
  onClose,
  excludeCardId,
}: InlineCardReferenceDialogProps) {
  return (
    <Popover
      initialOpen
      portal
      onClose={onClose}
      anchor="bottom start"
      style={{
        position: 'fixed',
        top: position.top,
        left: position.left,
        height: position.height,
        width: 0,
        overflow: 'hidden',
      }}
      triggerClassName="sr-only"
      panelClassName="w-[350px] max-w-[calc(100vw_-_16px)] min-w-[280px] max-h-[300px] overflow-y-auto shadow-2xl rounded-lg"
      panelStyle={{ zIndex: Z_INDEX.popover }}
      button={<span aria-hidden="true" />}
    >
      <BacklinkInputDropdownList
        onSelect={onSelect}
        onSearch={() => {}}
        autoFocus={true}
        excludeCardId={excludeCardId}
        placeholder="Search for card..."
        className="w-full"
      />
    </Popover>
  );
}
