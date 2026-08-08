import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { NavigationLinks } from './NavigationLinks';

describe('NavigationLinks', () => {
  it('renders Tasks and RSS links by default', () => {
    renderWithProviders(
      <NavigationLinks
        todayTasksCount={0}
        unreadRssCount={0}
        isCollapsed={false}
      />,
    );

    expect(screen.getByRole('link', { name: 'Tasks' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'RSS' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Search' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Files' })).toBeInTheDocument();
  });

  it('hides Tasks link when showTasks is false', () => {
    renderWithProviders(
      <NavigationLinks
        todayTasksCount={0}
        unreadRssCount={0}
        isCollapsed={false}
        showTasks={false}
      />,
    );

    expect(
      screen.queryByRole('link', { name: 'Tasks' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'RSS' })).toBeInTheDocument();
  });

  it('hides RSS link when showRss is false', () => {
    renderWithProviders(
      <NavigationLinks
        todayTasksCount={0}
        unreadRssCount={0}
        isCollapsed={false}
        showRss={false}
      />,
    );

    expect(screen.queryByRole('link', { name: 'RSS' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Tasks' })).toBeInTheDocument();
  });

  it('hides both Tasks and RSS links when both are false', () => {
    renderWithProviders(
      <NavigationLinks
        todayTasksCount={0}
        unreadRssCount={0}
        isCollapsed={false}
        showTasks={false}
        showRss={false}
      />,
    );

    expect(
      screen.queryByRole('link', { name: 'Tasks' }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'RSS' })).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Search' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Files' })).toBeInTheDocument();
  });
});
