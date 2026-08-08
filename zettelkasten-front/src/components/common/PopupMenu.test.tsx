import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PopupMenu } from './PopupMenu';

describe('PopupMenu', () => {
  const mockOptions = [
    { label: 'Option 1', onClick: vi.fn() },
    { label: 'Option 2', onClick: vi.fn() },
    { label: 'Option 3', onClick: vi.fn() },
  ];

  it('renders nothing when isOpen is false', () => {
    const { container } = render(
      <PopupMenu options={mockOptions} isOpen={false} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders menu when isOpen is true', () => {
    render(<PopupMenu options={mockOptions} isOpen={true} />);
    expect(screen.getByText('Option 1')).toBeInTheDocument();
    expect(screen.getByText('Option 2')).toBeInTheDocument();
    expect(screen.getByText('Option 3')).toBeInTheDocument();
  });

  it('calls onClick when option is clicked', async () => {
    const handleClick = vi.fn();
    const options = [{ label: 'Click me', onClick: handleClick }];

    render(<PopupMenu options={options} isOpen={true} />);
    await userEvent.click(screen.getByText('Click me'));

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('renders all options', () => {
    render(<PopupMenu options={mockOptions} isOpen={true} />);
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(3);
  });

  it('applies custom className to menu', () => {
    const { container } = render(
      <PopupMenu
        options={mockOptions}
        isOpen={true}
        className="custom-class"
      />,
    );
    const menu = container.querySelector('.custom-class');
    expect(menu).toBeInTheDocument();
  });

  it('applies custom className to individual option', () => {
    const options = [
      { label: 'Red Option', onClick: vi.fn(), className: 'text-red-500' },
    ];

    render(<PopupMenu options={options} isOpen={true} />);
    const button = screen.getByText('Red Option');
    expect(button).toHaveClass('text-red-500');
  });

  it('handles empty options array', () => {
    render(<PopupMenu options={[]} isOpen={true} />);
    const buttons = screen.queryAllByRole('button');
    expect(buttons).toHaveLength(0);
  });
});
