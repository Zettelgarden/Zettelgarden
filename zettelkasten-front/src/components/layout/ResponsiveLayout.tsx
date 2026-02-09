import React, { ReactNode } from 'react';
import { useResponsiveLayout } from '../../hooks/useResponsiveLayout';
import type { MobileView } from '../../hooks/useResponsiveLayout';

export type { MobileView } from '../../hooks/useResponsiveLayout';

export interface ResponsiveLayoutProps {
  mobileView: MobileView;
  setMobileView: (view: MobileView) => void;
  children: (isMobile: boolean) => ReactNode;
}

/**
 * ResponsiveLayout - A wrapper component for responsive page layouts
 *
 * This component uses the useResponsiveLayout hook to provide breakpoint detection
 * (768px) and renders children with an isMobile boolean flag. It enables conditional
 * mobile/desktop rendering for pages that need different layouts based on screen size.
 *
 * NOTE: This component is currently not used in SearchPage or TaskPage. Those pages
 * use the useResponsiveLayout hook directly for more control over state management.
 * This component is kept for potential future use cases where a wrapper pattern
 * might be preferred over direct hook usage.
 *
 * @example
 * ```tsx
 * function SearchPage() {
 *   const { isMobile, mobileView, setMobileView } = useResponsiveLayout();
 *
 *   return (
 *     <ResponsiveLayout mobileView={mobileView} setMobileView={setMobileView}>
 *       {(isMobile) => (
 *         isMobile ? (
 *           mobileView === 'list' && <MobileResultsList />
 *           mobileView === 'detail' && <MobileCardDetail />
 *           mobileView === 'filters' && <MobileFiltersSheet />
 *         ) : (
 *           <>
 *             <FiltersSidebar />
 *             <ResultsList />
 *             <CardDetailPanel />
 *           </>
 *         )
 *       )}
 *     </ResponsiveLayout>
 *   );
 * }
 * ```
 */
export function ResponsiveLayout({
  children,
}: ResponsiveLayoutProps) {
  const { isMobile } = useResponsiveLayout();

  return <>{children(isMobile)}</>;
}
