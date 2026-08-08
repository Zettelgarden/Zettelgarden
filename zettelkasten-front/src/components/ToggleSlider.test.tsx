import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToggleSlider } from './ToggleSlider';

describe('ToggleSlider', () => {
  it('renders with label', () => {
    render(
      <ToggleSlider
        label="Enable feature"
        initialState={false}
        onToggle={vi.fn()}
      />,
    );
    expect(screen.getByText('Enable feature')).toBeInTheDocument();
  });

  it('initializes with initial state', () => {
    render(
      <ToggleSlider label="Test" initialState={true} onToggle={vi.fn()} />,
    );
    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toBeChecked();
  });

  it('calls onToggle when clicked', async () => {
    const handleToggle = vi.fn();
    render(
      <ToggleSlider
        label="Test"
        initialState={false}
        onToggle={handleToggle}
      />,
    );

    await userEvent.click(screen.getByRole('checkbox'));
    expect(handleToggle).toHaveBeenCalledWith(true);
  });

  it('toggles state on click', async () => {
    render(
      <ToggleSlider label="Test" initialState={false} onToggle={vi.fn()} />,
    );
    const checkbox = screen.getByRole('checkbox');

    expect(checkbox).not.toBeChecked();
    await userEvent.click(checkbox);
    expect(checkbox).toBeChecked();
    await userEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();
  });

  it('calls onToggle with correct value on multiple toggles', async () => {
    const handleToggle = vi.fn();
    render(
      <ToggleSlider
        label="Test"
        initialState={false}
        onToggle={handleToggle}
      />,
    );

    await userEvent.click(screen.getByRole('checkbox'));
    expect(handleToggle).toHaveBeenCalledWith(true);

    await userEvent.click(screen.getByRole('checkbox'));
    expect(handleToggle).toHaveBeenCalledWith(false);
  });
});
