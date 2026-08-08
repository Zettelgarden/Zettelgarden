import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MobileBottomSheet } from './MobileBottomSheet';

describe('MobileBottomSheet', () => {
  it('should not render when isOpen is false', () => {
    const { container } = render(
      <MobileBottomSheet isOpen={false} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(container.firstChild).toBe(null);
  });

  it('should render when isOpen is true', () => {
    const { container } = render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(container.firstChild).not.toBe(null);
    expect(screen.getByText('Content')).toBeInTheDocument();
  });

  it('should render title when provided', () => {
    render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()} title="Test Title">
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(screen.getByText('Test Title')).toBeInTheDocument();
  });

  it('should not render title when not provided', () => {
    render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(screen.queryByRole('heading')).not.toBeInTheDocument();
  });

  it('should render close button when showCloseButton is true', () => {
    render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()} showCloseButton={true}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    const closeButton = screen.getByLabelText('Close');
    expect(closeButton).toBeInTheDocument();
  });

  it('should not render close button when showCloseButton is false', () => {
    render(
      <MobileBottomSheet
        isOpen={true}
        onClose={vi.fn()}
        showCloseButton={false}
      >
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(screen.queryByLabelText('Close')).not.toBeInTheDocument();
  });

  it('should call onClose when close button is clicked', () => {
    const handleClose = vi.fn();

    render(
      <MobileBottomSheet
        isOpen={true}
        onClose={handleClose}
        showCloseButton={true}
      >
        <div>Content</div>
      </MobileBottomSheet>,
    );

    const closeButton = screen.getByLabelText('Close');
    fireEvent.click(closeButton);

    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when backdrop is clicked', () => {
    const handleClose = vi.fn();

    const { container } = render(
      <MobileBottomSheet isOpen={true} onClose={handleClose}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    // The backdrop is the first child (the fixed overlay div)
    const backdrop = container.firstChild as HTMLElement;
    expect(backdrop).not.toBe(null);

    fireEvent.click(backdrop);

    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('should not call onClose when content is clicked', () => {
    const handleClose = vi.fn();

    render(
      <MobileBottomSheet isOpen={true} onClose={handleClose}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    const content = screen.getByText('Content');
    fireEvent.click(content);

    expect(handleClose).not.toHaveBeenCalled();
  });

  it('should call onClose when Escape key is pressed', () => {
    const handleClose = vi.fn();

    render(
      <MobileBottomSheet isOpen={true} onClose={handleClose}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('should prevent body scroll when open', () => {
    const { rerender } = render(
      <MobileBottomSheet isOpen={false} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(document.body.style.overflow).toBe('');

    rerender(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(document.body.style.overflow).toBe('hidden');
  });

  it('should restore body scroll when closed', () => {
    const { rerender } = render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(document.body.style.overflow).toBe('hidden');

    rerender(
      <MobileBottomSheet isOpen={false} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    expect(document.body.style.overflow).toBe('');
  });

  it('should render children correctly', () => {
    render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Child 1</div>
        <div>Child 2</div>
        <div>Child 3</div>
      </MobileBottomSheet>,
    );

    expect(screen.getByText('Child 1')).toBeInTheDocument();
    expect(screen.getByText('Child 2')).toBeInTheDocument();
    expect(screen.getByText('Child 3')).toBeInTheDocument();
  });

  it('should apply custom maxHeight when provided', () => {
    const { container } = render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()} maxHeight="90vh">
        <div>Content</div>
      </MobileBottomSheet>,
    );

    const sheet = container.querySelector('[style*="max-height"]');
    expect(sheet).toBeInTheDocument();
    // The maxHeight is applied via inline style
    expect(sheet?.getAttribute('style')).toContain('max-height: 90vh');
  });

  it('should use default maxHeight when not provided', () => {
    const { container } = render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    const sheet = container.querySelector('[style*="max-height"]');
    expect(sheet).toBeInTheDocument();
    expect(sheet?.getAttribute('style')).toContain('max-height: 80vh');
  });

  it('should have drag handle for visual affordance', () => {
    const { container } = render(
      <MobileBottomSheet isOpen={true} onClose={vi.fn()}>
        <div>Content</div>
      </MobileBottomSheet>,
    );

    // Look for the drag handle (a div with specific styling)
    const dragHandle = container.querySelector(
      '.w-12.h-1\\.5.bg-gray-300.rounded-full',
    );
    expect(dragHandle).toBeInTheDocument();
  });
});
