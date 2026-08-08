import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Collapsible } from './Collapsible';

describe('Collapsible', () => {
  it('renders the title and children when defaultOpen', () => {
    render(
      <Collapsible title="Linked references">
        <div>list content</div>
      </Collapsible>,
    );
    expect(screen.getByText('Linked references')).toBeInTheDocument();
    expect(screen.getByText('list content')).toBeInTheDocument();
  });

  it('hides children when defaultOpen is false', () => {
    render(
      <Collapsible title="Tags" defaultOpen={false}>
        <div>hidden content</div>
      </Collapsible>,
    );
    expect(screen.queryByText('hidden content')).not.toBeInTheDocument();
  });

  it('toggles open/closed on header click', () => {
    render(
      <Collapsible title="Section" defaultOpen={false}>
        <div>revealable</div>
      </Collapsible>,
    );
    expect(screen.queryByText('revealable')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Section'));
    expect(screen.getByText('revealable')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Section'));
    expect(screen.queryByText('revealable')).not.toBeInTheDocument();
  });

  it('sets aria-expanded to reflect state', () => {
    render(
      <Collapsible title="Section">
        <div>x</div>
      </Collapsible>,
    );
    expect(screen.getByText('Section').getAttribute('aria-expanded')).toBe(
      'true',
    );
    fireEvent.click(screen.getByText('Section'));
    expect(screen.getByText('Section').getAttribute('aria-expanded')).toBe(
      'false',
    );
  });

  it('shows a count badge when count > 0', () => {
    render(
      <Collapsible title="Refs" count={3}>
        <div>x</div>
      </Collapsible>,
    );
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('omits the count badge when count is 0', () => {
    render(
      <Collapsible title="Refs" count={0}>
        <div>x</div>
      </Collapsible>,
    );
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it('does not toggle when the right element is clicked', () => {
    const onRightClick = vi.fn();
    render(
      <Collapsible
        title="Section"
        defaultOpen={false}
        rightElement={<button onClick={onRightClick}>add</button>}
      >
        <div>kid</div>
      </Collapsible>,
    );
    fireEvent.click(screen.getByText('add'));
    expect(onRightClick).toHaveBeenCalledTimes(1);
    // Section should still be closed
    expect(screen.queryByText('kid')).not.toBeInTheDocument();
  });
});
