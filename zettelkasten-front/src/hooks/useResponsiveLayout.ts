import { useState, useEffect, useCallback } from 'react';

export type MobileView = 'list' | 'detail' | 'filters';

export interface UseResponsiveLayoutReturn {
  isMobile: boolean;
  mobileView: MobileView;
  setMobileView: (view: MobileView) => void;
}

const BREAKPOINT = 768;

/**
 * Hook for managing responsive layout state
 *
 * Provides:
 * - isMobile: boolean indicating if viewport is below breakpoint (768px)
 * - mobileView: current mobile view state ('list' | 'detail' | 'filters')
 * - setMobileView: function to update mobile view state
 *
 * @example
 * function SearchPage() {
 *   const { isMobile, mobileView, setMobileView } = useResponsiveLayout();
 *
 *   const handleCardClick = (card: Card) => {
 *     setSelectedCard(card);
 *     if (isMobile) {
 *       setMobileView('detail');
 *     }
 *   };
 *
 *   return (
 *     <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
 *       {(isMobile) => (
 *         isMobile ? (
 *           mobileView === 'list' && <MobileResultsList />
 *         ) : (
 *           <DesktopLayout />
 *         )
 *       )}
 *     </ResponsiveLayout>
 *   );
 * }
 */
export function useResponsiveLayout(): UseResponsiveLayoutReturn {
  // Initialize isMobile with lazy state to avoid unnecessary re-renders
  const [isMobile, setIsMobile] = useState<boolean>(() => {
    if (typeof window !== 'undefined') {
      return window.innerWidth < BREAKPOINT;
    }
    return false;
  });

  // Mobile view state for navigation
  const [mobileView, setMobileView] = useState<MobileView>('list');

  // Handle window resize with useCallback to prevent unnecessary re-renders
  const handleResize = useCallback(() => {
    setIsMobile(window.innerWidth < BREAKPOINT);
  }, []);

  useEffect(() => {
    // Only add event listener on client side
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', handleResize);
      return () => window.removeEventListener('resize', handleResize);
    }
  }, [handleResize]);

  return {
    isMobile,
    mobileView,
    setMobileView,
  };
}
