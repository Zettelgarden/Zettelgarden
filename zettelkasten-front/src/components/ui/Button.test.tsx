import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('renders children', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('calls onClick when clicked', async () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click</Button>);

    await userEvent.click(screen.getByText('Click'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    render(<Button disabled>Disabled</Button>);
    const button = screen.getByText('Disabled');
    expect(button).toBeDisabled();
  });

  it('does not call onClick when disabled', async () => {
    const handleClick = vi.fn();
    render(
      <Button onClick={handleClick} disabled>
        Click
      </Button>,
    );

    await userEvent.click(screen.getByText('Click'));
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('renders with primary variant by default and keeps the brand color', () => {
    render(<Button>Primary</Button>);
    const button = screen.getByText('Primary');
    expect(button).toHaveClass('bg-palette-dark');
  });

  it('renders with secondary variant', () => {
    render(<Button variant="secondary">Secondary</Button>);
    const button = screen.getByText('Secondary');
    expect(button).toHaveClass('bg-gray-200');
  });

  it('renders with outline variant', () => {
    render(<Button variant="outline">Outline</Button>);
    const button = screen.getByText('Outline');
    expect(button).toHaveClass('border');
  });

  it('renders with danger variant (red treatment)', () => {
    render(<Button variant="danger">Delete</Button>);
    const button = screen.getByText('Delete');
    expect(button).toHaveClass('bg-red-600');
    expect(button).toHaveClass('text-white');
  });

  it('keeps the min-h-[44px] touch-target convention on medium and large', () => {
    render(<Button>Medium</Button>);
    expect(screen.getByText('Medium')).toHaveClass('min-h-[44px]');

    render(<Button size="large">Large</Button>);
    expect(screen.getByText('Large')).toHaveClass('min-h-[44px]');
  });

  it('renders with small size (44px only on mobile)', () => {
    render(<Button size="small">Small</Button>);
    const button = screen.getByText('Small');
    expect(button).toHaveClass('min-h-[44px]');
    expect(button).toHaveClass('md:min-h-[32px]');
    expect(button).toHaveClass('text-sm');
  });

  it('renders with medium size by default', () => {
    render(<Button>Medium</Button>);
    expect(screen.getByText('Medium')).toHaveClass('px-4');
  });

  it('renders with large size', () => {
    render(<Button size="large">Large</Button>);
    expect(screen.getByText('Large')).toHaveClass('text-lg');
  });

  it('applies custom className', () => {
    render(<Button className="custom-class">Custom</Button>);
    const button = screen.getByText('Custom');
    expect(button).toHaveClass('custom-class');
  });

  it('passes through type and aria-pressed', () => {
    render(
      <Button type="submit" aria-pressed>
        Toggle
      </Button>,
    );
    const button = screen.getByText('Toggle');
    expect(button).toHaveAttribute('type', 'submit');
    expect(button).toHaveAttribute('aria-pressed', 'true');
  });
});
