import React from 'react';
import { Popover } from '../ui/Popover';

interface TaskDropdownProps {
  display: {
    icon?: string;
    text: string;
    color: string;
  };
  /** Extra classes for the panel (sizing) */
  menuClassName?: string;
  /** Panel content; may be a render-prop receiving { close } to close after an action */
  children:
    | React.ReactNode
    | ((props: { close: () => void }) => React.ReactElement);
}

/**
 * Consistent dropdown menu wrapper for Task*Display components.
 *
 * Built on ui/Popover: colored-badge trigger, portal panel anchored below the
 * badge with viewport-aware flipping, Headless-managed open/close (click
 * outside, Escape, focus return). Call `close` (via the children render prop)
 * after picking an option.
 */
export function TaskDropdown({
  display,
  menuClassName = 'min-w-[140px]',
  children,
}: TaskDropdownProps) {
  return (
    <Popover
      portal
      anchor="bottom start"
      triggerClassName="cursor-pointer inline-flex items-center justify-center gap-1 px-1.5 py-0 min-w-[32px] min-h-[24px] rounded-md text-xs font-medium transition-colors hover:opacity-80"
      style={{
        backgroundColor: display.color + '20',
        color: display.color,
        border: `1px solid ${display.color}40`,
      }}
      button={
        <>
          {display.icon && <span>{display.icon}</span>}
          <span>{display.text}</span>
        </>
      }
      panelClassName={`py-1 ${menuClassName}`}
    >
      {typeof children === 'function'
        ? (children as (props: { close: () => void }) => React.ReactElement)
        : children}
    </Popover>
  );
}
