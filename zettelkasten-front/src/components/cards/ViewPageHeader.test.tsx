import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ViewPageHeader } from './ViewPageHeader';
import { UIStateProvider } from '../../contexts/UIStateContext';
import { sampleCards } from '../../tests/data';
import { ViewMode } from '../../pages/cards/ViewPageContainer';

const [card] = sampleCards();

function renderHeader(overrides: Partial<React.ComponentProps<typeof ViewPageHeader>> = {}) {
  const noop = () => {};
  return render(
    <UIStateProvider>
      <ViewPageHeader
        viewingCard={card}
        isPinned={false}
        onTogglePin={noop}
        onEditCard={noop}
        onToggleStar={noop}
        toggleCreateTaskWindow={noop}
        onResummarize={noop}
        onRecategorize={noop}
        viewMode={'normal' as ViewMode}
        onViewModeChange={noop}
        {...overrides}
      />
    </UIStateProvider>,
  );
}

describe('ViewPageHeader — info pane toggle', () => {
  it('renders the toggle button by default', () => {
    renderHeader();
    expect(screen.getByTitle('Toggle info pane')).toBeInTheDocument();
  });

  it('reflects the open state with aria-pressed', () => {
    renderHeader();
    const toggle = screen.getByRole('button', { name: 'Toggle info pane' });
    expect(toggle).toHaveAttribute('aria-pressed', 'true');
  });

  it('is hidden when hideRailToggle is set (pinned pane)', () => {
    renderHeader({ hideRailToggle: true });
    expect(screen.queryByTitle('Toggle info pane')).not.toBeInTheDocument();
  });
});
