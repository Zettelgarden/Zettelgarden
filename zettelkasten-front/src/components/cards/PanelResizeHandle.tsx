import React, { useCallback, useEffect, useRef } from 'react';

interface PanelResizeHandleProps {
  /** Current width (px) of the panel being resized. */
  width: number;
  /** Invoked with the new width (already clamped to [min, max]) on each move. */
  onResize: (width: number) => void;
  /** Minimum allowed panel width. */
  minWidth?: number;
  /** Maximum allowed panel width. */
  maxWidth?: number;
}

/**
 * Vertical drag handle for resizing the card view's right info panel.
 * The panel sits to the right of the handle, so dragging left widens it
 * and dragging right narrows it. Desktop only (hidden below the md
 * breakpoint, where the layout stacks vertically).
 */
export function PanelResizeHandle({
  width,
  onResize,
  minWidth = 280,
  maxWidth = 640,
}: PanelResizeHandleProps) {
  const startXRef = useRef(0);
  const startWidthRef = useRef(width);

  // Keep the latest props in refs so the global listeners below keep stable
  // identities across re-renders. If `onResize` (e.g. the context's
  // `setRightPaneWidth`) is a new function identity on each render, memoizing
  // on it would recreate the listeners every time and the unmount cleanup
  // would tear them off mid-drag, killing the resize after the first move.
  const onResizeRef = useRef(onResize);
  onResizeRef.current = onResize;
  const widthRef = useRef(width);
  widthRef.current = width;

  const clamp = useCallback(
    (value: number) => Math.min(maxWidth, Math.max(minWidth, value)),
    [minWidth, maxWidth],
  );

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      // Panel is on the right: dragging the handle left increases width.
      const delta = e.clientX - startXRef.current;
      onResizeRef.current(clamp(startWidthRef.current - delta));
    },
    [clamp],
  );

  const handleMouseUp = useCallback(() => {
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  }, [handleMouseMove]);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      startXRef.current = e.clientX;
      startWidthRef.current = widthRef.current;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    },
    [handleMouseMove, handleMouseUp],
  );

  // Ensure global listeners are removed if the component unmounts mid-drag.
  // handleMouseMove/handleMouseUp are stable, so this cleanup only runs on
  // unmount and never tears the listeners off during an active drag.
  useEffect(() => {
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [handleMouseMove, handleMouseUp]);

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize info panel"
      title="Drag to resize"
      onMouseDown={handleMouseDown}
      onDoubleClick={() => onResize((minWidth + maxWidth) / 2)}
      className="hidden md:flex w-4 shrink-0 cursor-col-resize items-center justify-center group relative"
    >
      <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-px bg-gray-200 group-hover:bg-blue-400 transition-colors" />
    </div>
  );
}
