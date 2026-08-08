// zettelkasten-front/src/components/cards/ViewMobileAccordion.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ViewMobileAccordion } from './ViewMobileAccordion';

describe('ViewMobileAccordion', () => {
  it('renders title in header', () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Content</div>
      </ViewMobileAccordion>,
    );
    expect(screen.getByText('Tags')).toBeInTheDocument();
  });

  it('shows content when expanded by default', () => {
    render(
      <ViewMobileAccordion title="Tags" defaultExpanded>
        <div>Test Content</div>
      </ViewMobileAccordion>,
    );
    expect(screen.getByText('Test Content')).toBeVisible();
  });

  it('hides content when collapsed', () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Test Content</div>
      </ViewMobileAccordion>,
    );
    expect(screen.queryByText('Test Content')).not.toBeInTheDocument();
  });

  it('toggles content on header click', () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Test Content</div>
      </ViewMobileAccordion>,
    );

    // Initially collapsed
    expect(screen.queryByText('Test Content')).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByText('Tags'));
    expect(screen.getByText('Test Content')).toBeVisible();

    // Click to collapse
    fireEvent.click(screen.getByText('Tags'));
    expect(screen.queryByText('Test Content')).not.toBeInTheDocument();
  });

  it('renders right element in header', () => {
    render(
      <ViewMobileAccordion title="Tags" rightElement={<button>Edit</button>}>
        <div>Content</div>
      </ViewMobileAccordion>,
    );
    expect(screen.getByText('Edit')).toBeInTheDocument();
  });
});
