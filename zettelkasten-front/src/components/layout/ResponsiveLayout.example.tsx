import React from 'react';
import { ResponsiveLayout } from './ResponsiveLayout';
import type { MobileView } from '../../hooks/useResponsiveLayout';

/**
 * Example usage of ResponsiveLayout component
 *
 * This demonstrates how to use ResponsiveLayout to create responsive pages
 * with different layouts for mobile and desktop.
 */

// Example 1: Basic usage with conditional rendering
export function BasicResponsiveExample() {
  const [mobileView, setMobileView] = React.useState<MobileView>('list');

  return (
    <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
      {(isMobile) => (
        <div>
          {isMobile ? (
            <div>Mobile Layout - View: {mobileView}</div>
          ) : (
            <div>Desktop Layout - 3 Column Layout</div>
          )}
        </div>
      )}
    </ResponsiveLayout>
  );
}

// Example 2: Search Page pattern (from design doc)
export function SearchPageExample() {
  const [mobileView, setMobileView] = React.useState<MobileView>('list');
  const [selectedCard, setSelectedCard] = React.useState<any>(null);

  const handleCardClick = (card: any) => {
    setSelectedCard(card);
    if (mobileView === 'list') {
      setMobileView('detail');
    }
  };

  return (
    <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
      {(isMobile) => (
        <div className="flex h-screen">
          {isMobile ? (
            // Mobile rendering based on mobileView
            <>
              {mobileView === 'list' && (
                <div className="w-full">
                  {/* Mobile list view */}
                  <button onClick={() => setMobileView('filters')}>Show Filters</button>
                  <div>Search Results List</div>
                </div>
              )}
              {mobileView === 'detail' && (
                <div className="w-full">
                  <button onClick={() => setMobileView('list')}>Back to List</button>
                  <div>Card Detail View</div>
                </div>
              )}
              {mobileView === 'filters' && (
                <div className="w-full">
                  <button onClick={() => setMobileView('list')}>Close Filters</button>
                  <div>Filter Options</div>
                </div>
              )}
            </>
          ) : (
            // Desktop multi-column layout
            <>
              <div className="w-64 border-r">Filters Sidebar</div>
              <div className="w-80 border-r">Results List</div>
              <div className="flex-1">Card Detail Panel</div>
            </>
          )}
        </div>
      )}
    </ResponsiveLayout>
  );
}

// Example 3: Task Page pattern
export function TaskPageExample() {
  const [mobileView, setMobileView] = React.useState<MobileView>('list');

  return (
    <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
      {(isMobile) => (
        <div className="flex h-screen">
          {isMobile ? (
            // Mobile view with filters bottom sheet
            <>
              {mobileView === 'list' && (
                <div className="w-full">
                  <button onClick={() => setMobileView('filters')}>Show Filters</button>
                  <div>Task List</div>
                </div>
              )}
              {mobileView === 'filters' && (
                <div className="w-full">
                  <button onClick={() => setMobileView('list')}>Close Filters</button>
                  <div>Task Filters</div>
                </div>
              )}
            </>
          ) : (
            // Desktop with optional right panel
            <>
              <div className="flex-1">Main Task Content</div>
              <div className="w-64 border-l">Quick Lists (optional)</div>
            </>
          )}
        </div>
      )}
    </ResponsiveLayout>
  );
}
