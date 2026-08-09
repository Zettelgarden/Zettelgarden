import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge } from './Badge';

describe('Badge', () => {
  it('renders children', () => {
    render(<Badge>Completed</Badge>);
    expect(screen.getByText('Completed')).toBeInTheDocument();
  });

  it('maps each color to its pill classes', () => {
    const cases: Array<[Parameters<typeof Badge>[0]['color'], string]> = [
      ['success', 'bg-green-100 text-green-800'],
      ['warning', 'bg-yellow-100 text-yellow-800'],
      ['error', 'bg-red-100 text-red-800'],
      ['info', 'bg-blue-100 text-blue-800'],
      ['neutral', 'bg-gray-100 text-gray-800'],
    ];
    for (const [color, classes] of cases) {
      const { unmount } = render(<Badge color={color}>{color}</Badge>);
      const el = screen.getByText(color as string);
      for (const cls of classes.split(' ')) {
        expect(el).toHaveClass(cls);
      }
      unmount();
    }
  });

  it('defaults to neutral', () => {
    render(<Badge>Free</Badge>);
    expect(screen.getByText('Free')).toHaveClass('bg-gray-100');
  });

  it('renders a colored dot matching the color', () => {
    render(
      <Badge color="success" dot>
        Completed
      </Badge>,
    );
    const badge = screen.getByText('Completed');
    const dot = badge.parentElement!.querySelector('span span');
    expect(dot).toHaveClass('bg-green-500');
  });

  it('pulses the dot when pulse is set', () => {
    render(
      <Badge color="warning" pulse>
        Running
      </Badge>,
    );
    const badge = screen.getByText('Running');
    const dot = badge.parentElement!.querySelector('span span');
    expect(dot).toHaveClass('animate-pulse');
    expect(dot).toHaveClass('bg-yellow-500');
  });

  it('applies custom className', () => {
    render(<Badge className="uppercase">Trial</Badge>);
    expect(screen.getByText('Trial')).toHaveClass('uppercase');
  });
});
