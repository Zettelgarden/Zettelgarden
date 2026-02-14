import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';

describe('DeleteConfirmDialog', () => {
  it('should render dialog when isOpen is true', () => {
    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="row"
        index={2}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText('Delete row 3?')).toBeInTheDocument();
    expect(screen.getByText(/This row contains data/)).toBeInTheDocument();
  });

  it('should not render when isOpen is false', () => {
    const { container } = render(
      <DeleteConfirmDialog
        isOpen={false}
        itemType="column"
        index={1}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should call onConfirm when Delete button clicked', () => {
    const onConfirm = vi.fn();

    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="row"
        index={0}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Delete row/i }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('should call onCancel when Cancel button clicked', () => {
    const onCancel = vi.fn();

    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="column"
        index={5}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
