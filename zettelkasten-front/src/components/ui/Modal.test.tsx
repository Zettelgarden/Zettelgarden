import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from './Modal';

describe('Modal', () => {
  it('renders children when open', () => {
    render(
      <Modal open onClose={() => {}}>
        <p>Modal content</p>
      </Modal>,
    );
    expect(screen.getByText('Modal content')).toBeInTheDocument();
  });

  it('renders nothing when closed', async () => {
    render(
      <Modal open={false} onClose={() => {}}>
        <p>Modal content</p>
      </Modal>,
    );
    await waitFor(() => {
      expect(screen.queryByText('Modal content')).not.toBeInTheDocument();
    });
  });

  it('calls onClose when Escape is pressed', async () => {
    const handleClose = vi.fn();
    render(
      <Modal open onClose={handleClose}>
        <button>Close me</button>
      </Modal>,
    );
    await userEvent.keyboard('{Escape}');
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('moves focus into the dialog when opened', async () => {
    render(
      <Modal open onClose={() => {}}>
        <button>First button</button>
      </Modal>,
    );
    await waitFor(() => {
      expect(document.activeElement).not.toBe(document.body);
    });
    // Tab cycles to the first interactive element inside the dialog
    await userEvent.tab();
    expect(screen.getByText('First button')).toHaveFocus();
  });

  it('closes when open flips to false', async () => {
    const { rerender } = render(
      <Modal open onClose={() => {}}>
        <p>Modal content</p>
      </Modal>,
    );
    expect(screen.getByText('Modal content')).toBeInTheDocument();

    rerender(
      <Modal open={false} onClose={() => {}}>
        <p>Modal content</p>
      </Modal>,
    );
    await waitFor(() => {
      expect(screen.queryByText('Modal content')).not.toBeInTheDocument();
    });
  });

  it('applies the size preset and custom panel classes', () => {
    render(
      <Modal open onClose={() => {}} size="4xl" className="rounded-2xl">
        <p>Modal content</p>
      </Modal>,
    );
    const panel = screen.getByText('Modal content').closest('div');
    expect(panel).toHaveClass('max-w-4xl');
    expect(panel).toHaveClass('rounded-2xl');
  });
});
