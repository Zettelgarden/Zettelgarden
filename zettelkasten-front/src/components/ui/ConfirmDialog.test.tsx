import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfirmDialog, useConfirmDialog } from './ConfirmDialog';

describe('ConfirmDialog', () => {
  it('renders title and message', () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={() => {}}
        title="Delete Task"
        message="Are you sure?"
      />,
    );
    expect(screen.getByText('Delete Task')).toBeInTheDocument();
    expect(screen.getByText('Are you sure?')).toBeInTheDocument();
  });

  it('calls onConfirm and onClose when confirm is clicked', async () => {
    const handleConfirm = vi.fn();
    const handleClose = vi.fn();
    render(
      <ConfirmDialog
        isOpen
        onClose={handleClose}
        onConfirm={handleConfirm}
        title="Delete Task"
        message="Are you sure?"
        confirmText="Delete"
      />,
    );
    await userEvent.click(screen.getByText('Delete'));
    expect(handleConfirm).toHaveBeenCalledTimes(1);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when cancel is clicked', async () => {
    const handleClose = vi.fn();
    render(
      <ConfirmDialog
        isOpen
        onClose={handleClose}
        onConfirm={() => {}}
        title="Delete Task"
        message="Are you sure?"
        cancelText="Cancel"
      />,
    );
    await userEvent.click(screen.getByText('Cancel'));
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('closes via Escape', async () => {
    const handleClose = vi.fn();
    render(
      <ConfirmDialog
        isOpen
        onClose={handleClose}
        onConfirm={() => {}}
        title="Delete Task"
        message="Are you sure?"
      />,
    );
    await userEvent.keyboard('{Escape}');
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('does not auto-close on confirm when closeOnConfirm is false', async () => {
    const handleConfirm = vi.fn();
    const handleClose = vi.fn();
    render(
      <ConfirmDialog
        isOpen
        onClose={handleClose}
        onConfirm={handleConfirm}
        title="Confirm Restore"
        message="Restore this version?"
        confirmText="Restore"
        closeOnConfirm={false}
      />,
    );
    await userEvent.click(screen.getByText('Restore'));
    expect(handleConfirm).toHaveBeenCalledTimes(1);
    expect(handleClose).not.toHaveBeenCalled();
  });

  it('disables both buttons while isLoading', () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={() => {}}
        title="Confirm Restore"
        message="Restore this version?"
        confirmText="Restore"
        cancelText="Cancel"
        isLoading
      />,
    );
    expect(screen.getByText('Restore')).toBeDisabled();
    expect(screen.getByText('Cancel')).toBeDisabled();
  });

  it('exposes the title as the accessible name', () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={() => {}}
        title="Delete Task"
        message="Are you sure?"
      />,
    );
    expect(
      screen.getByRole('dialog', { name: 'Delete Task' }),
    ).toBeInTheDocument();
  });

  it('renders custom children instead of the default message', () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={() => {}}
        title="Confirm Restore"
        message="Should not appear"
      >
        <p>Custom change preview</p>
      </ConfirmDialog>,
    );
    expect(screen.getByText('Custom change preview')).toBeInTheDocument();
    expect(screen.queryByText('Should not appear')).not.toBeInTheDocument();
  });

  it('requires checkbox before confirming', async () => {
    const handleConfirm = vi.fn();
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={handleConfirm}
        title="Delete Task"
        message="Are you sure?"
        confirmText="Delete"
        requireCheckbox
        checkboxLabel="I understand"
      />,
    );
    const confirmButton = screen.getByText('Delete');
    expect(confirmButton).toBeDisabled();
    await userEvent.click(screen.getByLabelText('I understand'));
    expect(confirmButton).toBeEnabled();
    await userEvent.click(confirmButton);
    expect(handleConfirm).toHaveBeenCalledTimes(1);
  });
});

describe('useConfirmDialog', () => {
  function Harness({ onResult }: { onResult: (v: boolean) => void }) {
    const { confirm, Dialog } = useConfirmDialog();
    return (
      <>
        <button
          onClick={async () => {
            const result = await confirm({
              title: 'Delete Task',
              message: 'Are you sure?',
              onConfirm: () => {},
            });
            onResult(result);
          }}
        >
          Open
        </button>
        <Dialog />
      </>
    );
  }

  it('resolves true when confirmed', async () => {
    const onResult = vi.fn();
    render(<Harness onResult={onResult} />);
    await userEvent.click(screen.getByText('Open'));
    await userEvent.click(screen.getByText('Confirm'));
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(true));
  });

  it('resolves false when cancelled (no hanging await)', async () => {
    const onResult = vi.fn();
    render(<Harness onResult={onResult} />);
    await userEvent.click(screen.getByText('Open'));
    await userEvent.click(screen.getByText('Cancel'));
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false));
  });

  it('resolves false on Escape', async () => {
    const onResult = vi.fn();
    render(<Harness onResult={onResult} />);
    await userEvent.click(screen.getByText('Open'));
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(false));
  });
});
