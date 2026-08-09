import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Popover } from './Popover';

describe('Popover', () => {
  it('opens on click and shows panel content', async () => {
    render(
      <Popover button={<span>?</span>}>
        <p>Help text</p>
      </Popover>,
    );
    expect(screen.queryByText('Help text')).not.toBeInTheDocument();
    await userEvent.click(screen.getByText('?'));
    expect(screen.getByText('Help text')).toBeInTheDocument();
  });

  it('closes on Escape and returns focus to the trigger', async () => {
    render(
      <Popover button={<span>?</span>}>
        <p>Help text</p>
      </Popover>,
    );
    await userEvent.click(screen.getByText('?'));
    const panel = screen.getByText('Help text');
    expect(panel).toBeInTheDocument();
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByText('Help text')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '?' })).toHaveFocus();
  });

  it('sets a tooltip on the trigger via title', () => {
    render(
      <Popover button={<span>?</span>} title="Body syntax help">
        <p>Help text</p>
      </Popover>,
    );
    expect(screen.getByTitle('Body syntax help')).toBeInTheDocument();
  });

  it('applies a replacement triggerClassName for custom triggers', () => {
    render(
      <Popover
        button={<span>Badge</span>}
        triggerClassName="inline-flex items-center px-1.5 py-0 rounded-md text-xs"
      >
        <p>content</p>
      </Popover>,
    );
    const trigger = screen.getByRole('button', { name: 'Badge' });
    expect(trigger.className).toContain('inline-flex');
    expect(trigger.className).not.toContain('min-w-[44px]');
  });

  it('exposes close() to panel children via render prop', async () => {
    const handleClose = vi.fn();
    render(
      <Popover button={<span>⋮</span>}>
        {({ close }) => (
          <button
            type="button"
            onClick={() => {
              handleClose();
              close();
            }}
          >
            Pick me
          </button>
        )}
      </Popover>,
    );
    await userEvent.click(screen.getByText('⋮'));
    await userEvent.click(screen.getByText('Pick me'));
    expect(handleClose).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('Pick me')).not.toBeInTheDocument();
  });

  it('opens on mount with initialOpen (programmatic popovers)', async () => {
    render(
      <Popover initialOpen button={<span />}>
        <p>Auto panel</p>
      </Popover>,
    );
    expect(screen.getByText('Auto panel')).toBeInTheDocument();
  });
});
