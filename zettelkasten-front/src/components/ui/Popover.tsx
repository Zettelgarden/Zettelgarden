import React from 'react';
import { Popover as HeadlessPopover } from '@headlessui/react';

const defaultTriggerClassName =
  'p-1.5 md:p-1 rounded hover:bg-gray-100 transition-colors min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500';

export interface PopoverProps {
  /** Trigger content rendered inside the popover button */
  button: React.ReactNode;
  /** Extra classes for the trigger button (appended to the standard look) */
  buttonClassName?: string;
  /** Replaces the standard trigger-button styling entirely (custom triggers like colored badges) */
  triggerClassName?: string;
  /** Inline styles for the trigger button (e.g. dynamic badge colors) */
  style?: React.CSSProperties;
  /** Tooltip on the trigger button */
  title?: string;
  /** Extra classes for the panel (positioning + sizing) */
  panelClassName?: string;
  /** Inline styles for the panel (e.g. z-index from Z_INDEX constants) */
  panelStyle?: React.CSSProperties;
  /** Render the panel in a portal at body level (auto-enabled when `anchor` is set) */
  portal?: boolean;
  /** Open the popover on mount (for programmatically-opened, anchor-positioned popovers) */
  initialOpen?: boolean;
  /** Called when the popover closes (Escape, click-outside, or close()) */
  onClose?: () => void;
  /** Anchor the panel to the trigger with viewport-aware flipping, e.g. 'bottom start' (see Headless UI anchors) */
  anchor?: string;
  /** Panel content; may be a render-prop receiving { open, close } */
  children:
    | React.ReactNode
    | ((props: { open: boolean; close: () => void }) => React.ReactElement);
}

/**
 * Popover — the shared popover shell on Headless UI Popover.
 *
 * Provides the trigger button and a panel with Headless-managed open/close
 * (click outside, Escape, focus return). Panels are NOT menus — no
 * role="menu" semantics; use ui/Menu for action menus. Position the panel
 * with `anchor` (viewport-aware) or panelClassName (absolute/left-full etc).
 */
export function Popover({
  button,
  buttonClassName = '',
  triggerClassName,
  style,
  title,
  panelClassName = '',
  panelStyle,
  portal,
  anchor,
  initialOpen = false,
  onClose,
  children,
}: PopoverProps) {
  const buttonRef = React.useRef<HTMLButtonElement>(null);
  const hasAutoOpened = React.useRef(false);

  // Programmatic open: Headless Popover toggles on Button click. Guarded so
  // StrictMode's double effect run doesn't toggle it back closed.
  React.useEffect(() => {
    if (initialOpen && !hasAutoOpened.current && buttonRef.current) {
      hasAutoOpened.current = true;
      buttonRef.current.click();
    }
  }, [initialOpen]);

  return (
    <HeadlessPopover as="div" className="relative flex-shrink-0">
      {({ open }) => (
        <>
          <TrackOpen open={open} onClose={onClose} />
          <HeadlessPopover.Button
            ref={buttonRef}
            type="button"
            title={title}
            style={style}
            className={
              triggerClassName !== undefined
                ? triggerClassName
                : `${defaultTriggerClassName} ${buttonClassName}`
            }
          >
            {button}
          </HeadlessPopover.Button>
          <HeadlessPopover.Panel
            anchor={anchor as never}
            portal={portal}
            style={panelStyle}
            className={`bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none ${panelClassName}`}
          >
            {typeof children === 'function'
              ? (children as (props: {
                  open: boolean;
                  close: () => void;
                }) => React.ReactElement)
              : children}
          </HeadlessPopover.Panel>
        </>
      )}
    </HeadlessPopover>
  );
}

/** Notifies onClose when the Headless popover transitions open -> closed. */
function TrackOpen({ open, onClose }: { open: boolean; onClose?: () => void }) {
  const prev = React.useRef(open);
  React.useEffect(() => {
    if (prev.current && !open) onClose?.();
    prev.current = open;
  }, [open, onClose]);
  return null;
}
