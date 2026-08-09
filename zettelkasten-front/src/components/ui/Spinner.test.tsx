import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Spinner } from './Spinner';

describe('Spinner', () => {
  it('renders a status role with a default label', () => {
    render(<Spinner />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('Loading')).toBeInTheDocument();
  });

  it('renders the accessible label', () => {
    render(<Spinner label="Fetching data" />);
    expect(screen.getByText('Fetching data')).toBeInTheDocument();
  });

  it('applies the default (md) size', () => {
    render(<Spinner />);
    const svg = screen.getByRole('status').querySelector('svg');
    expect(svg).toHaveClass('h-5');
    expect(svg).toHaveClass('w-5');
  });

  it('applies each size variant', () => {
    const { rerender } = render(<Spinner size="sm" />);
    let svg = screen.getByRole('status').querySelector('svg');
    expect(svg).toHaveClass('h-4');
    expect(svg).toHaveClass('w-4');

    rerender(<Spinner size="lg" />);
    svg = screen.getByRole('status').querySelector('svg');
    expect(svg).toHaveClass('h-8');
    expect(svg).toHaveClass('w-8');

    rerender(<Spinner size="xl" />);
    svg = screen.getByRole('status').querySelector('svg');
    expect(svg).toHaveClass('h-12');
    expect(svg).toHaveClass('w-12');
  });

  it('applies custom className (e.g. text color)', () => {
    render(<Spinner className="text-blue-600" />);
    expect(screen.getByRole('status')).toHaveClass('text-blue-600');
  });

  it('animates', () => {
    render(<Spinner />);
    expect(screen.getByRole('status').querySelector('svg')).toHaveClass(
      'animate-spin',
    );
  });
});
