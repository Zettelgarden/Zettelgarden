// zettelkasten-front/src/components/stats/NetworkHealthSection.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { NetworkHealthSection } from './NetworkHealthSection';
import { getNetworkStats } from '../../api/graph';
import { NetworkStats } from '../../models/Graph';

vi.mock('../../api/graph', () => ({
  getNetworkStats: vi.fn(),
}));

function makeStats(): NetworkStats {
  return {
    total_cards: 50,
    total_links: 25,
    avg_links_per_card: 0.5,
    orphan_count: 4,
    top_connectors: [
      {
        card: {
          id: 1,
          card_id: 'HUB-1',
          user_id: 1,
          title: 'Hub Card',
          parent_id: 0,
          created_at: new Date('2024-01-01'),
          updated_at: new Date('2024-01-01'),
          tags: [],
        },
        count: 7,
      },
    ],
    links_by_month: [
      { month: '2026-03', count: 2 },
      { month: '2026-04', count: 5 },
      { month: '2026-05', count: 0 },
      { month: '2026-06', count: 3 },
      { month: '2026-07', count: 1 },
      { month: '2026-08', count: 4 },
    ],
  };
}

describe('NetworkHealthSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(getNetworkStats).mockResolvedValue(makeStats());
  });

  function renderSection() {
    return render(
      <MemoryRouter>
        <NetworkHealthSection />
      </MemoryRouter>,
    );
  }

  it('renders headline metrics', async () => {
    renderSection();
    expect(await screen.findByText('50')).toBeInTheDocument();
    expect(screen.getByText('25')).toBeInTheDocument();
    expect(screen.getByText('0.50')).toBeInTheDocument();
    // Orphan count '4' also appears as a month-bar count; at least one match.
    expect(screen.getAllByText('4').length).toBeGreaterThanOrEqual(1);
  });

  it('renders top connectors with counts and month bars', async () => {
    renderSection();

    await waitFor(() => {
      expect(screen.getByText('- Hub Card')).toBeInTheDocument();
    });
    expect(screen.getByText('7 links')).toBeInTheDocument();
    expect(screen.getByText('2026-03')).toBeInTheDocument();
    expect(screen.getByText('2026-08')).toBeInTheDocument();
  });

  it('shows an empty state when there are no links', async () => {
    const stats = makeStats();
    stats.total_links = 0;
    stats.top_connectors = [];
    stats.links_by_month = stats.links_by_month.map((m) => ({
      ...m,
      count: 0,
    }));
    vi.mocked(getNetworkStats).mockResolvedValue(stats);

    renderSection();

    await waitFor(() => {
      expect(screen.getByText('No links yet.')).toBeInTheDocument();
    });
  });

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(getNetworkStats).mockRejectedValue(new Error('down'));
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderSection();
    expect(
      await screen.findByText('Failed to load network stats'),
    ).toBeInTheDocument();

    consoleSpy.mockRestore();
  });
});
