import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MobileTopBar, MobileTopBarLeftAction } from './MobileTopBar';

describe('MobileTopBar', () => {
  describe('Rendering', () => {
    it('should render title', () => {
      render(<MobileTopBar title="Test Page" />);
      expect(screen.getByText('Test Page')).toBeInTheDocument();
    });

    it('should render badge when provided', () => {
      render(<MobileTopBar title="RSS" badge="5" />);
      expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('should render numeric badge', () => {
      render(<MobileTopBar title="RSS" badge={10} />);
      expect(screen.getByText('10')).toBeInTheDocument();
    });

    it('should not render badge when value is 0', () => {
      render(<MobileTopBar title="RSS" badge={0} />);
      expect(screen.queryByText('0')).not.toBeInTheDocument();
    });

    it('should not render badge when value is empty string', () => {
      const { container } = render(<MobileTopBar title="RSS" badge="" />);
      expect(container.querySelector('.bg-red-500')).not.toBeInTheDocument();
    });

    it('should not render badge when value is undefined', () => {
      const { container } = render(<MobileTopBar title="RSS" />);
      expect(container.querySelector('.bg-red-500')).not.toBeInTheDocument();
    });

    it('should render back button when onBack is provided', () => {
      render(<MobileTopBar title="Back Test" onBack={() => {}} />);
      expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument();
    });

    it('should render menu button when onMenuClick is provided without onBack', () => {
      render(<MobileTopBar title="Menu Test" onMenuClick={() => {}} />);
      expect(screen.getByRole('button', { name: /open menu/i })).toBeInTheDocument();
    });

    it('should prioritize back button over menu button', () => {
      render(
        <MobileTopBar
          title="Priority Test"
          onBack={() => {}}
          onMenuClick={() => {}}
        />
      );
      expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /open menu/i })).not.toBeInTheDocument();
    });

    it('should render actions on the right side', () => {
      render(
        <MobileTopBar
          title="Actions Test"
          actions={<button data-testid="custom-action">Action</button>}
        />
      );
      expect(screen.getByTestId('custom-action')).toBeInTheDocument();
    });

    it('should render multiple actions', () => {
      render(
        <MobileTopBar
          title="Multiple Actions"
          actions={
            <>
              <button data-testid="action-1">Action 1</button>
              <button data-testid="action-2">Action 2</button>
            </>
          }
        />
      );
      expect(screen.getByTestId('action-1')).toBeInTheDocument();
      expect(screen.getByTestId('action-2')).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const { container } = render(
        <MobileTopBar title="Class Test" className="custom-class" />
      );
      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('should apply custom zIndex', () => {
      const { container } = render(
        <MobileTopBar title="Z-Index Test" zIndex={100} />
      );
      expect(container.firstChild).toHaveStyle({ zIndex: 100 });
    });

    it('should have mobileOnly class by default', () => {
      const { container } = render(<MobileTopBar title="Mobile Test" />);
      expect(container.firstChild).toHaveClass('md:hidden');
    });

    it('should not have mobileOnly class when mobileOnly is false', () => {
      const { container } = render(
        <MobileTopBar title="Desktop Test" mobileOnly={false} />
      );
      expect(container.firstChild).not.toHaveClass('md:hidden');
    });

    it('should have sticky positioning', () => {
      const { container } = render(<MobileTopBar title="Sticky Test" />);
      expect(container.firstChild).toHaveClass('sticky', 'top-0');
    });
  });

  describe('Interactions', () => {
    it('should call onBack when back button is clicked', () => {
      const onBack = vi.fn();
      render(<MobileTopBar title="Back Test" onBack={onBack} />);

      const backButton = screen.getByRole('button', { name: /go back/i });
      fireEvent.click(backButton);

      expect(onBack).toHaveBeenCalledTimes(1);
    });

    it('should call onMenuClick when menu button is clicked', () => {
      const onMenuClick = vi.fn();
      render(<MobileTopBar title="Menu Test" onMenuClick={onMenuClick} />);

      const menuButton = screen.getByRole('button', { name: /open menu/i });
      fireEvent.click(menuButton);

      expect(onMenuClick).toHaveBeenCalledTimes(1);
    });

    it('should trigger action button clicks', () => {
      const actionCallback = vi.fn();
      render(
        <MobileTopBar
          title="Actions Test"
          actions={<button onClick={actionCallback}>Action</button>}
        />
      );

      const actionButton = screen.getByRole('button', { name: 'Action' });
      fireEvent.click(actionButton);

      expect(actionCallback).toHaveBeenCalledTimes(1);
    });
  });

  describe('Accessibility', () => {
    it('should have aria-label on back button', () => {
      render(<MobileTopBar title="A11y Test" onBack={() => {}} />);
      expect(screen.getByRole('button', { name: /go back/i })).toHaveAccessibleName(/go back/i);
    });

    it('should have aria-label on menu button', () => {
      render(<MobileTopBar title="A11y Test" onMenuClick={() => {}} />);
      expect(screen.getByRole('button', { name: /open menu/i })).toHaveAccessibleName(/open menu/i);
    });
  });

  describe('Edge Cases', () => {
    it('should handle very long titles with truncation', () => {
      const longTitle = 'This is a very long title that should be truncated with an ellipsis when it overflows the available space';
      const { container } = render(<MobileTopBar title={longTitle} />);
      expect(container.querySelector('.truncate')).toBeInTheDocument();
    });

    it('should handle badge value of 99+', () => {
      render(<MobileTopBar title="RSS" badge="99+" />);
      expect(screen.getByText('99+')).toBeInTheDocument();
    });

    it('should handle empty actions', () => {
      const { container } = render(<MobileTopBar title="Empty Actions" actions={undefined} />);
      expect(container.querySelector('.flex.items-center.gap-1')).toBeInTheDocument();
    });

    it('should handle null actions', () => {
      const { container } = render(
        <MobileTopBar title="Null Actions" actions={null} />
      );
      expect(container.querySelector('.flex.items-center.gap-1')).toBeInTheDocument();
    });
  });
});

describe('MobileTopBarLeftAction', () => {
  it('should render children', () => {
    render(
      <MobileTopBarLeftAction onClick={() => {}}>
        <span data-testid="custom-content">Custom</span>
      </MobileTopBarLeftAction>
    );
    expect(screen.getByTestId('custom-content')).toBeInTheDocument();
  });

  it('should call onClick when clicked', () => {
    const onClick = vi.fn();
    render(
      <MobileTopBarLeftAction onClick={onClick}>
        <span>Click me</span>
      </MobileTopBarLeftAction>
    );

    const button = screen.getByRole('button');
    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('should use custom ariaLabel', () => {
    render(
      <MobileTopBarLeftAction onClick={() => {}} ariaLabel="Custom action">
        <span>Action</span>
      </MobileTopBarLeftAction>
    );
    expect(screen.getByRole('button', { name: 'Custom action' })).toBeInTheDocument();
  });

  it('should use default ariaLabel when not provided', () => {
    render(
      <MobileTopBarLeftAction onClick={() => {}}>
        <span>Action</span>
      </MobileTopBarLeftAction>
    );
    expect(screen.getByRole('button', { name: /action/i })).toBeInTheDocument();
  });
});
