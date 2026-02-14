import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpreadsheetContextMenu } from './SpreadsheetContextMenu';

describe('SpreadsheetContextMenu', () => {
  it('should render menu items when position is provided', () => {
    const actions = [
      { label: 'Insert Row', action: vi.fn() },
      { label: 'Delete Row', action: vi.fn() },
    ];

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={actions}
        onClose={() => {}}
      />
    );

    expect(screen.getByText('Insert Row')).toBeInTheDocument();
    expect(screen.getByText('Delete Row')).toBeInTheDocument();
  });

  it('should not render when position is null', () => {
    const { container } = render(
      <SpreadsheetContextMenu
        position={null}
        actions={[]}
        onClose={() => {}}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should call action and onClose when item clicked', () => {
    const action = vi.fn();
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Test', action }]}
        onClose={onClose}
      />
    );

    fireEvent.click(screen.getByText('Test'));
    expect(action).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should not call action for disabled item', () => {
    const action = vi.fn();
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Disabled', action, disabled: true }]}
        onClose={onClose}
      />
    );

    fireEvent.click(screen.getByText('Disabled'));
    expect(action).not.toHaveBeenCalled();
  });

  it('should close when clicking outside', () => {
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Test', action: vi.fn() }]}
        onClose={onClose}
      />
    );

    fireEvent.mouseDown(document.body);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should close when Escape key pressed', () => {
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Test', action: vi.fn() }]}
        onClose={onClose}
      />
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
