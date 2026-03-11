import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FileTags } from './FileTags';

describe('FileTags', () => {
  it('renders existing tags', () => {
    const tags = ['taxes', '2024'];
    render(<FileTags tags={tags} onAddTag={() => {}} onRemoveTag={() => {}} />);

    expect(screen.getByText('#taxes')).toBeInTheDocument();
    expect(screen.getByText('#2024')).toBeInTheDocument();
  });

  it('calls onAddTag when tag added', () => {
    const onAddTag = vi.fn();
    render(<FileTags tags={[]} onAddTag={onAddTag} onRemoveTag={() => {}} />);

    const input = screen.getByPlaceholderText('Add tag...');
    fireEvent.change(input, { target: { value: 'mortgage' } });
    fireEvent.keyPress(input, { key: 'Enter', charCode: 13 });

    expect(onAddTag).toHaveBeenCalledWith('mortgage');
  });

  it('calls onRemoveTag when tag removed', () => {
    const onRemoveTag = vi.fn();
    render(<FileTags tags={['taxes']} onAddTag={() => {}} onRemoveTag={onRemoveTag} />);

    const removeButton = screen.getByLabelText('Remove tag taxes');
    fireEvent.click(removeButton);

    expect(onRemoveTag).toHaveBeenCalledWith('taxes');
  });
});
