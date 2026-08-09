import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Field } from './Field';
import { Input } from './Input';
import { Select } from './Select';
import { Label } from './Label';

describe('Label', () => {
  it('renders text and associates with htmlFor', () => {
    render(<Label htmlFor="name">Name</Label>);
    const label = screen.getByText('Name');
    expect(label.tagName).toBe('LABEL');
    expect(label).toHaveAttribute('for', 'name');
  });
});

describe('Input', () => {
  it('renders and passes through native props', async () => {
    const handleChange = vi.fn();
    render(
      <Input
        id="name"
        value="hello"
        onChange={handleChange}
        placeholder="Enter name"
      />,
    );
    const input = screen.getByPlaceholderText('Enter name');
    expect(input).toHaveValue('hello');
    await userEvent.type(input, 'x');
    expect(handleChange).toHaveBeenCalled();
  });

  it('uses the default border style', () => {
    render(<Input />);
    expect(screen.getByRole('textbox')).toHaveClass('border-gray-300');
  });

  it('applies error styling when hasError is set', () => {
    render(<Input hasError />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveClass('border-red-500');
    expect(input).toHaveClass('focus:ring-red-500');
  });

  it('applies custom className', () => {
    render(<Input className="w-40" />);
    expect(screen.getByRole('textbox')).toHaveClass('w-40');
  });
});

describe('Select', () => {
  it('renders options', () => {
    render(
      <Select id="status" defaultValue="active">
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
      </Select>,
    );
    const select = screen.getByRole('combobox');
    expect(select).toHaveValue('active');
    expect(
      screen.getByRole('option', { name: 'Inactive' }),
    ).toBeInTheDocument();
  });

  it('applies error styling when hasError is set', () => {
    render(
      <Select hasError>
        <option value="a">A</option>
      </Select>,
    );
    expect(screen.getByRole('combobox')).toHaveClass('border-red-500');
  });
});

describe('Field', () => {
  it('associates the label with the control via htmlFor', () => {
    render(
      <Field label="Name" htmlFor="name">
        <Input id="name" />
      </Field>,
    );
    expect(screen.getByLabelText('Name')).toBeInTheDocument();
  });

  it('renders help text when no error', () => {
    render(
      <Field label="Name" htmlFor="name" help="Shown on your profile">
        <Input id="name" />
      </Field>,
    );
    expect(screen.getByText('Shown on your profile')).toBeInTheDocument();
  });

  it('renders an error message (role=alert) and hides help', () => {
    render(
      <Field
        label="Name"
        htmlFor="name"
        help="Shown on your profile"
        error="Name is required"
      >
        <Input id="name" hasError />
      </Field>,
    );
    expect(screen.getByRole('alert')).toHaveTextContent('Name is required');
    expect(screen.queryByText('Shown on your profile')).not.toBeInTheDocument();
  });

  it('renders a required asterisk', () => {
    render(
      <Field label="Name" htmlFor="name" required>
        <Input id="name" />
      </Field>,
    );
    expect(screen.getByText('*')).toBeInTheDocument();
  });
});
