import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Menu, MenuItem } from './Menu';
import { Dropdown } from './Dropdown';

describe('Menu', () => {
  it('opens on click and shows items', async () => {
    render(
      <Menu button={<span>⋮</span>}>
        <MenuItem onClick={() => {}}>Edit</MenuItem>
        <MenuItem onClick={() => {}}>Delete</MenuItem>
      </Menu>,
    );
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    await userEvent.click(screen.getByText('⋮'));
    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(screen.getByText('Edit')).toBeInTheDocument();
    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('calls onClick when an item is selected', async () => {
    const handleEdit = vi.fn();
    render(
      <Menu button={<span>⋮</span>}>
        <MenuItem onClick={handleEdit}>Edit</MenuItem>
      </Menu>,
    );
    await userEvent.click(screen.getByText('⋮'));
    await userEvent.click(screen.getByText('Edit'));
    expect(handleEdit).toHaveBeenCalledTimes(1);
  });

  it('closes after selecting an item', async () => {
    render(
      <Menu button={<span>⋮</span>}>
        <MenuItem onClick={() => {}}>Edit</MenuItem>
      </Menu>,
    );
    await userEvent.click(screen.getByText('⋮'));
    await userEvent.click(screen.getByText('Edit'));
    await waitFor(() => {
      expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });
  });

  it('supports keyboard navigation (arrow down + enter)', async () => {
    const handleDelete = vi.fn();
    render(
      <Menu button={<span>⋮</span>}>
        <MenuItem onClick={() => {}}>Edit</MenuItem>
        <MenuItem onClick={handleDelete}>Delete</MenuItem>
      </Menu>,
    );
    await userEvent.click(screen.getByText('⋮'));
    const menu = screen.getByRole('menu');
    fireEvent.keyDown(menu, { key: 'ArrowDown' });
    fireEvent.keyDown(menu, { key: 'ArrowDown' });
    fireEvent.keyDown(menu, { key: 'Enter' });
    expect(handleDelete).toHaveBeenCalledTimes(1);
  });
});

describe('Dropdown', () => {
  const options = [
    { value: 'active', label: 'Active' },
    { value: 'inactive', label: 'Inactive' },
  ];

  it('shows the placeholder when nothing is selected', () => {
    render(
      <Dropdown
        value={undefined}
        options={options}
        onChange={() => {}}
        placeholder="Choose status"
      />,
    );
    expect(screen.getByText('Choose status')).toBeInTheDocument();
  });

  it('shows the selected option label', () => {
    render(<Dropdown value="active" options={options} onChange={() => {}} />);
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('opens on click and calls onChange with the picked value', async () => {
    const handleChange = vi.fn();
    render(
      <Dropdown value="active" options={options} onChange={handleChange} />,
    );
    await userEvent.click(screen.getByText('Active'));
    await userEvent.click(screen.getByText('Inactive'));
    expect(handleChange).toHaveBeenCalledWith('inactive');
  });

  it('marks the selected option with a check', async () => {
    render(<Dropdown value="active" options={options} onChange={() => {}} />);
    await userEvent.click(screen.getByText('Active'));
    const check = screen.getByRole('menu').querySelector('svg');
    expect(check).toBeInTheDocument();
  });
});
