import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SchemaDialog } from './SchemaDialog';
import { SchemaDefinition } from '../../models/Schema';

const { updateSchema, createSchema } = vi.hoisted(() => ({
  updateSchema: vi.fn(),
  createSchema: vi.fn(),
}));

vi.mock('../../api/schemas', () => ({
  updateSchema,
  createSchema,
}));

const now = new Date();

const schema: SchemaDefinition = {
  id: 1,
  name: 'Book Review',
  slug: 'book-review',
  owner_id: 1,
  fields: [
    { name: 'Author', type: 'text', required: true },
    { name: 'Rating', type: 'number', required: false },
    {
      name: 'Genre',
      type: 'select',
      required: false,
      options: ['Fiction'],
    },
  ],
  created_at: now,
  updated_at: now,
  is_deleted: false,
};

function renderDialog() {
  return render(
    <SchemaDialog
      schema={schema}
      isOpen={true}
      onClose={() => {}}
      onSuccess={() => {}}
    />,
  );
}

function fieldNames(): string[] {
  return screen
    .getAllByPlaceholderText('field_name')
    .map((input) => (input as HTMLInputElement).value);
}

describe('SchemaDialog field reorder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    updateSchema.mockResolvedValue(schema);
    createSchema.mockResolvedValue(schema);
  });

  it('renders fields in schema order', () => {
    renderDialog();
    expect(fieldNames()).toEqual(['Author', 'Rating', 'Genre']);
  });

  it('moves a field down and persists the new order on save', async () => {
    renderDialog();

    fireEvent.click(
      screen.getByRole('button', { name: 'Move field Author down' }),
    );
    expect(fieldNames()).toEqual(['Rating', 'Author', 'Genre']);

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(updateSchema).toHaveBeenCalledWith(1, {
        name: 'Book Review',
        fields: [
          { name: 'Rating', type: 'number', required: false, options: [] },
          { name: 'Author', type: 'text', required: true, options: [] },
          {
            name: 'Genre',
            type: 'select',
            required: false,
            options: ['Fiction'],
          },
        ],
      });
    });
  });

  it('moves a field up and persists the new order on save', async () => {
    renderDialog();

    fireEvent.click(
      screen.getByRole('button', { name: 'Move field Genre up' }),
    );
    expect(fieldNames()).toEqual(['Author', 'Genre', 'Rating']);

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(updateSchema).toHaveBeenCalledWith(1, {
        name: 'Book Review',
        fields: [
          { name: 'Author', type: 'text', required: true, options: [] },
          {
            name: 'Genre',
            type: 'select',
            required: false,
            options: ['Fiction'],
          },
          { name: 'Rating', type: 'number', required: false, options: [] },
        ],
      });
    });
  });

  it('disables move-up on the first field and move-down on the last field', () => {
    renderDialog();
    expect(
      screen.getByRole('button', { name: 'Move field Author up' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Move field Genre down' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Move field Rating up' }),
    ).not.toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Move field Rating down' }),
    ).not.toBeDisabled();
  });
});
