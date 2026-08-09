import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Combobox } from './Combobox';

const options = [
  { id: 'note-1', title: 'My Note' },
  { id: 'note-2', title: 'Reading List' },
];

function renderCombobox(
  overrides: Partial<
    Parameters<typeof Combobox<{ id: string; title: string }>>[0]
  > = {},
) {
  const onChange = vi.fn();
  return {
    onChange,
    ...render(
      <Combobox
        value={null}
        onChange={onChange}
        inputValue=""
        onInputChange={() => {}}
        options={options}
        getOptionKey={(o) => o.id}
        renderOption={(o, active) => (
          <div className={active ? 'bg-blue-50' : ''}>{o.title}</div>
        )}
        {...overrides}
      />,
    ),
  };
}

describe('Combobox', () => {
  it('shows options after typing', async () => {
    renderCombobox();
    const input = screen.getByRole('combobox');
    await userEvent.type(input, 'a');
    expect(screen.getByText('My Note')).toBeInTheDocument();
    expect(screen.getByText('Reading List')).toBeInTheDocument();
  });

  it('calls onChange with the picked option and clears the list', async () => {
    const { onChange } = renderCombobox();
    await userEvent.type(screen.getByRole('combobox'), 'a');
    await userEvent.click(screen.getByText('My Note'));
    expect(onChange).toHaveBeenCalledWith(options[0]);
    expect(screen.queryByText('My Note')).not.toBeInTheDocument();
  });

  it('supports keyboard navigation (arrow down + enter)', async () => {
    const { onChange } = renderCombobox();
    const input = screen.getByRole('combobox');
    await userEvent.type(input, 'a');
    await userEvent.keyboard('{ArrowDown}');
    await userEvent.keyboard('{Enter}');
    // ArrowDown moves from the pre-highlighted first option to the second.
    expect(onChange).toHaveBeenCalledWith(options[1]);
  });

  it('shows the loading row while loading', async () => {
    renderCombobox({ isLoading: true, options: [] });
    await userEvent.type(screen.getByRole('combobox'), 'a');
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('shows the empty state with no matches', async () => {
    renderCombobox({ options: [] });
    await userEvent.type(screen.getByRole('combobox'), 'a');
    expect(screen.getByText('No results found')).toBeInTheDocument();
  });
});
